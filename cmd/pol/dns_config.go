package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

func dnsAddress(raw string, local bool) string {
	if strings.HasPrefix(raw, "quic://") {
		return "quic+local://" + strings.TrimPrefix(raw, "quic://")
	}
	if local {
		for _, prefix := range []string{"https://", "tcp://"} {
			if strings.HasPrefix(raw, prefix) {
				return strings.Replace(raw, "://", "+local://", 1)
			}
		}
	}
	return raw
}
func applyDNSConfiguration(cfg object, s Settings) error {
	if len(s.DNSServers) > 64 || len(s.DNSHosts) > 128 {
		return fmt.Errorf("too many DNS servers or bootstrap hosts")
	}
	packet := s.Mode == "serverless" && (s.ServerlessMode == "" || s.ServerlessMode == "packet")
	dnsMode := s.Mode == "serverless" && s.ServerlessMode == "dns"
	dns := cfg["dns"].(map[string]any)
	hosts := object{}
	var known map[string][]string
	if b, e := os.ReadFile("/usr/share/pol/dns/public-ips.json"); e == nil {
		_ = json.Unmarshal(b, &known)
	}
	if known == nil {
		known = map[string][]string{"dns.google": {"8.8.8.8", "8.8.4.4"}, "cloudflare-dns.com": {"104.16.248.249", "104.16.249.249"}}
	}
	for host, ips := range known {
		var allowed []string
		for _, ip := range ips {
			if s.DNSStrategy != "UseIPv4" || net.ParseIP(ip).To4() != nil {
				allowed = append(allowed, ip)
			}
		}
		if len(allowed) > 0 {
			hosts[host] = allowed
		}
	}
	for host, ips := range s.DNSHosts {
		if host == "" || len(host) > 253 || strings.ContainsAny(host, " /\r\n\t") {
			return fmt.Errorf("invalid DNS bootstrap host")
		}
		if len(ips) == 0 || len(ips) > 16 {
			return fmt.Errorf("invalid DNS bootstrap addresses")
		}
		for _, ip := range ips {
			if net.ParseIP(ip) == nil {
				return fmt.Errorf("DNS bootstrap entries must be IP addresses")
			}
		}
		hosts[host] = ips
	}
	for _, p := range s.WorkerProfiles {
		if net.ParseIP(p.Address) != nil {
			hosts[p.Host] = []string{p.Address}
		}
	}
	addresses := append([]string{}, s.DNSServers...)
	if len(addresses) == 0 {
		switch {
		case dnsMode:
			if !s.FakeSNIDNS {
				for _, p := range s.WorkerProfiles {
					if p.Host != "" && p.UUID != "" {
						addresses = append(addresses, "https://"+p.Host+"/"+p.UUID)
					}
				}
				if len(addresses) == 0 {
					addresses = []string{s.DNS}
				}
			}
		case packet:
			addresses = []string{"https://dns.google/dns-query", "https://cloudflare-dns.com/dns-query"}
		default:
			addresses = []string{s.DNS}
		}
	}
	var servers []any
	if s.FakeDNS || packet {
		servers = append(servers, "fakedns")
		cfg["fakedns"] = []any{object{"ipPool": "198.18.0.0/15", "poolSize": 65535}}
		for _, raw := range cfg["inbounds"].([]any) {
			in := raw.(map[string]any)
			if sniff, ok := in["sniffing"].(map[string]any); ok {
				sniff["destOverride"] = []string{"http", "tls", "quic", "fakedns"}
				sniff["routeOnly"] = false
			}
		}
	}
	for _, raw := range addresses {
		if len(raw) > 2048 || strings.ContainsAny(raw, "\r\n\x00") || raw == "" {
			return fmt.Errorf("invalid DNS server")
		}
		if ip := net.ParseIP(raw); ip != nil && (s.Mode == "worker" || s.Mode == "tor") {
			raw = "tcp://" + net.JoinHostPort(raw, "53")
		}
		tag := "remote-dns"
		parsed, _ := url.Parse(raw)
		if parsed != nil && net.ParseIP(parsed.Hostname()) != nil && net.ParseIP(parsed.Hostname()).IsLoopback() {
			tag = "pol-local-dns"
		}
		servers = append(servers, object{"address": dnsAddress(raw, dnsMode), "tag": tag})
	}
	routing := cfg["routing"].(map[string]any)
	var extraRules []any
	if dnsMode && s.FakeSNIDNS {
		for _, server := range []struct {
			name, address, sni string
			ips                []string
		}{
			{"cloudflare", "1.1.1.1", "one.one.one.one", []string{"1.1.1.1", "1.0.0.1"}},
			{"google", "8.8.8.8", "dns.google", []string{"8.8.8.8", "8.8.4.4"}},
			{"quad9", "9.9.9.10", "dns10.quad9.net", []string{"9.9.9.10", "149.112.112.10"}},
		} {
			servers = append(servers, object{"address": "h2c://" + server.address + "/dns-query", "tag": server.name + "-dns"})
			hosts[server.name+".bootstraps"] = server.ips
			cfg["outbounds"] = append(cfg["outbounds"].([]any), object{"tag": "tls-wrapper-" + server.name, "protocol": "freedom", "settings": object{"redirect": server.name + ".bootstraps:443"}, "streamSettings": object{"security": "tls", "tlsSettings": object{"serverName": "www.google.com", "alpn": []string{"h2"}, "fingerprint": "chrome", "verifyPeerCertByName": server.sni}, "sockopt": object{"domainStrategy": "ForceIP"}}})
			extraRules = append(extraRules, object{"type": "field", "inboundTag": []string{server.name + "-dns"}, "outboundTag": "tls-wrapper-" + server.name})
		}
	}
	extraRules = append(extraRules, object{"type": "field", "inboundTag": []string{"remote-dns"}, "outboundTag": "proxy"}, object{"type": "field", "inboundTag": []string{"pol-local-dns"}, "outboundTag": "direct"})
	old := routing["rules"].([]any)
	// API and intercepted DNS keep their precedence; generated DNS requests must
	// bootstrap the underlying connection rather than wait for its final exit.
	routing["rules"] = append(append(old[:2:2], extraRules...), old[2:]...)
	dns["hosts"] = hosts
	dns["servers"] = servers
	if packet || dnsMode {
		dns["enableParallelQuery"] = true
	}
	return nil
}
