package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Preserve the imported routing, DNS, balancers and outbound graph while replacing
// device-specific listeners with the router's managed listeners.
func buildImportedConfig(s Settings) (object, error) {
	var cfg object
	if err := json.Unmarshal(s.Custom, &cfg); err != nil {
		return nil, err
	}
	outs, ok := cfg["outbounds"].([]any)
	if !ok || len(outs) == 0 {
		return nil, fmt.Errorf("Xray config must contain outbounds")
	}
	for i, raw := range outs {
		out, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid outbound")
		}
		tag, _ := out["tag"].(string)
		if i == 0 && tag == "" {
			out["tag"] = "proxy"
		}
		if strings.HasPrefix(tag, "pol-managed-") {
			return nil, fmt.Errorf("reserved outbound tag: %s", tag)
		}
	}
	base := s
	base.Mode = "serverless"
	base.ServerlessMode = "dns"
	base.FakeDNS = false
	base.FakeSNIDNS = false
	base.DNSServers = []string{s.DNS}
	base.ExitMode = "none"
	base.BestPing = false
	base.Chain = ""
	managed, err := buildConfig(base)
	if err != nil {
		return nil, err
	}
	inboundMap := map[string][]string{}
	var retained []any
	for _, raw := range asArray(cfg["inbounds"]) {
		in, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		tag, _ := in["tag"].(string)
		if tag == "" {
			continue
		}
		listen, _ := in["listen"].(string)
		port, _ := in["port"].(float64)
		if (listen == "127.0.0.1" || listen == "::1") && port >= 1024 && (port < 24800 || port >= 24840) {
			retained = append(retained, in)
			continue
		}
		if strings.Contains(strings.ToLower(tag), "dns") {
			inboundMap[tag] = []string{"dns-in", "dns6-in"}
		} else {
			inboundMap[tag] = []string{"lan-in", "lan6-in", "socks-in", "http-in"}
		}
	}
	oldAPI := ""
	if api, ok := cfg["api"].(map[string]any); ok {
		oldAPI, _ = api["tag"].(string)
	}
	cfg["inbounds"], cfg["api"], cfg["stats"] = append(managed["inbounds"].([]any), retained...), managed["api"], managed["stats"]
	if _, ok := cfg["dns"]; !ok {
		cfg["dns"] = object{"queryStrategy": s.DNSStrategy, "servers": []any{object{"address": s.DNS, "tag": "remote-dns"}}}
	}
	if _, ok := cfg["log"]; !ok {
		cfg["log"] = managed["log"]
	}
	policy, _ := cfg["policy"].(map[string]any)
	if policy == nil {
		policy = object{}
	}
	system, _ := policy["system"].(map[string]any)
	if system == nil {
		system = object{}
	}
	system["statsInboundUplink"], system["statsInboundDownlink"] = true, true
	policy["system"] = system
	cfg["policy"] = policy
	routing, _ := cfg["routing"].(map[string]any)
	if routing == nil {
		routing = object{"domainStrategy": "IPIfNonMatch"}
	}
	var rules []any
	mr := managed["routing"].(map[string]any)["rules"].([]any)
	for _, raw := range mr[:len(mr)-1] {
		r := raw.(map[string]any)
		switch r["outboundTag"] {
		case "proxy":
			r["outboundTag"] = outs[0].(map[string]any)["tag"]
		case "direct":
			r["outboundTag"] = "pol-managed-direct"
		case "block":
			r["outboundTag"] = "pol-managed-block"
		case "dns-out":
			r["outboundTag"] = "pol-managed-dns"
		}
		rules = append(rules, r)
	}
	for _, raw := range asArray(routing["rules"]) {
		r, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid routing rule")
		}
		if oldAPI != "" && r["outboundTag"] == oldAPI {
			continue
		}
		if tags, ok := r["inboundTag"].([]any); ok {
			var mapped []string
			for _, v := range tags {
				tag, _ := v.(string)
				if replace, ok := inboundMap[tag]; ok {
					mapped = append(mapped, replace...)
				} else {
					mapped = append(mapped, tag)
				}
			}
			r["inboundTag"] = mapped
		}
		rules = append(rules, r)
	}
	routing["rules"] = rules
	cfg["routing"] = routing
	cfg["outbounds"] = append(outs, object{"tag": "pol-managed-direct", "protocol": "freedom"}, object{"tag": "pol-managed-block", "protocol": "blackhole"}, object{"tag": "pol-managed-dns", "protocol": "dns"})
	return cfg, nil
}

func asArray(v any) []any { a, _ := v.([]any); return a }
