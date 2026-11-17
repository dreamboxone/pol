package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func validateSelection(s Settings) error {
	if len(s.CleanIPs) > 24 || len(s.Endpoints) > 24 || len(s.WorkerPorts) > 12 {
		return fmt.Errorf("too many connection candidates")
	}
	for _, port := range s.WorkerPorts {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid Worker port")
		}
	}
	if s.ProbeInterval != 0 && (s.ProbeInterval < 10 || s.ProbeInterval > 3600) {
		return fmt.Errorf("probe interval must be 10-3600 seconds")
	}
	for _, value := range s.CleanIPs {
		if len(value) > 253 || value == "" || strings.ContainsAny(value, " /\r\n\t") {
			return fmt.Errorf("invalid clean IP or domain")
		}
	}
	for _, value := range s.Endpoints {
		host, port, err := net.SplitHostPort(value)
		if err != nil || host == "" {
			return fmt.Errorf("endpoint must be host:port")
		}
		p, e := strconv.Atoi(port)
		if e != nil || p < 1 || p > 65535 || strings.ContainsAny(host, " /\r\n\t") {
			return fmt.Errorf("invalid endpoint")
		}
	}
	return nil
}
func applyEndpointPool(cfg object, s Settings) (object, error) {
	if !s.BestPing || !contains([]string{"worker", "warp", "masque"}, s.Mode) {
		return cfg, nil
	}
	var candidates []Settings
	if s.Mode == "worker" {
		addresses := append([]string{s.Address}, s.CleanIPs...)
		if enabled, _ := s.CustomCDN["enabled"].(bool); enabled && s.Variant != "fragment" {
			for _, raw := range asArray(s.CustomCDN["addresses"]) {
				if address, ok := raw.(string); ok {
					addresses = append(addresses, address)
				}
			}
		}
		ports := s.WorkerPorts
		if len(ports) == 0 {
			ports = []int{s.Port}
		}
		seen := map[string]bool{}
		for _, address := range addresses {
			if address == "" {
				address = s.Host
			}
			for _, port := range ports {
				key := net.JoinHostPort(address, strconv.Itoa(port))
				if seen[key] {
					continue
				}
				seen[key] = true
				next := s
				next.Address = address
				next.Port = port
				candidates = append(candidates, next)
			}
		}
	} else {
		seen := map[string]bool{}
		for _, endpoint := range append([]string{s.Endpoint}, s.Endpoints...) {
			if seen[endpoint] {
				continue
			}
			seen[endpoint] = true
			next := s
			next.Endpoint = endpoint
			candidates = append(candidates, next)
		}
	}
	if len(candidates) > 24 {
		return nil, fmt.Errorf("best-ping pool is limited to 24 endpoints")
	}
	outs := cfg["outbounds"].([]any)
	for i, raw := range outs {
		if raw.(map[string]any)["tag"] == "proxy" {
			outs[i] = object{"tag": "proxy", "protocol": "loopback", "settings": object{"inboundTag": "best-ping-upstream"}}
			break
		}
	}
	for i, candidate := range candidates {
		out, err := proxyOutbound(candidate)
		if err != nil {
			return nil, err
		}
		out["tag"] = fmt.Sprintf("pol-pool-%d", i)
		outs = append(outs, out)
	}
	cfg["outbounds"] = outs
	routing := cfg["routing"].(map[string]any)
	routing["rules"] = append([]any{object{"type": "field", "inboundTag": []string{"best-ping-upstream"}, "balancerTag": "best-ping"}}, routing["rules"].([]any)...)
	routing["balancers"] = []any{object{"tag": "best-ping", "selector": []string{"pol-pool-"}, "strategy": object{"type": "leastPing"}, "fallbackTag": "pol-pool-0"}}
	interval := s.ProbeInterval
	if interval == 0 {
		interval = 30
	}
	cfg["burstObservatory"] = object{"subjectSelector": []string{"pol-pool-"}, "pingConfig": object{"destination": "https://www.gstatic.com/generate_204", "connectivity": "http://cp.cloudflare.com/generate_204", "interval": fmt.Sprint(interval) + "s", "sampling": 3, "timeout": "5s"}}
	return cfg, nil
}
