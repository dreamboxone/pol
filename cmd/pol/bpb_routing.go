package main

import "fmt"

func applyBPBRouting(cfg object, s Settings) error {
	if len(s.BPBRouting) == 0 {
		return nil
	}
	assets := map[string]map[string]string{
		"bypass":         {"iran": "ir", "china": "cn", "russia": "category-ru"},
		"block":          {"ads": "category-ads-all", "porn": "nsfw", "malware": "malware", "phishing": "phishing", "cryptominers": "cryptominers"},
		"sanctionBypass": {"openAi": "openai", "googleAi": "google-deepmind", "microsoft": "microsoft", "oracle": "oracle", "docker": "docker", "adobe": "adobe", "epicGames": "epicgames", "intel": "intel", "amd": "amd", "nvidia": "nvidia", "asus": "asus", "hp": "hp", "lenovo": "lenovo"},
	}
	var rules []any
	var sanction []string
	for _, group := range []string{"bypass", "sanctionBypass", "block"} {
		value := mapAt(s.BPBRouting, group)
		tag := "direct"
		if group == "block" {
			tag = "block"
		}
		var domains, ips []string
		for _, raw := range asArray(value["domains"]) {
			d, ok := raw.(string)
			if !ok || !validGeoDomain(d) {
				return fmt.Errorf("invalid BPB routing domain")
			}
			domains = append(domains, "domain:"+d)
		}
		for _, raw := range asArray(value["ips"]) {
			ip, ok := raw.(string)
			if !ok || !validDirectIP(ip) {
				return fmt.Errorf("invalid BPB routing IP")
			}
			ips = append(ips, ip)
		}
		for key, on := range mapAt(value, "presets") {
			if on != true {
				continue
			}
			if group == "block" && key == "quic" {
				rules = append(rules, object{"type": "field", "network": "udp", "port": "443", "outboundTag": tag})
				continue
			}
			if group == "bypass" && key == "bittorrent" {
				rules = append(rules, object{"type": "field", "protocol": []string{"bittorrent"}, "outboundTag": tag})
				continue
			}
			asset, ok := assets[group][key]
			if !ok {
				return fmt.Errorf("unknown BPB routing preset: %s", key)
			}
			domains = append(domains, "geosite:"+asset)
			if group == "bypass" {
				country := map[string]string{"iran": "ir", "china": "cn", "russia": "ru"}[key]
				ips = append(ips, "geoip:"+country)
			}
		}
		if len(domains) > 0 {
			rules = append(rules, object{"type": "field", "domain": domains, "outboundTag": tag})
		}
		if len(ips) > 0 {
			rules = append(rules, object{"type": "field", "ip": ips, "outboundTag": tag})
		}
		if group == "sanctionBypass" {
			sanction = append(sanction, domains...)
		}
	}
	routing := cfg["routing"].(map[string]any)
	old := routing["rules"].([]any)
	index := 2
	for index < len(old) {
		r := old[index].(map[string]any)
		if findTag, _ := r["outboundTag"].(string); findTag != "proxy" && findTag != "direct" {
			break
		}
		if r["inboundTag"] == nil {
			break
		}
		index++
	}
	routing["rules"] = append(append(old[:index:index], rules...), old[index:]...)
	if strategy, ok := s.BPBRouting["domainStrategy"].(string); ok {
		if strategy != "AsIs" && strategy != "IPIfNonMatch" {
			return fmt.Errorf("invalid BPB routing strategy")
		}
		routing["domainStrategy"] = strategy
	}
	if len(sanction) > 0 {
		dns := cfg["dns"].(map[string]any)
		address := s.AntiSanctionDNS
		if address == "" {
			address = "178.22.122.100"
		}
		dns["servers"] = append([]any{object{"address": address, "domains": sanction, "skipFallback": true, "tag": "pol-local-dns"}}, dns["servers"].([]any)...)
	}
	return nil
}
