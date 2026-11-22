package main

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Public BPB settings never replace router credentials or private accounts.
func importBPBProfile(s Settings, data object) (Settings, error) {
	p, ok := data["profile"].(map[string]any)
	if !ok {
		return s, fmt.Errorf("invalid BPB profile")
	}
	if v, ok := p["schemaVersion"].(float64); ok && v != 1 {
		return s, fmt.Errorf("unsupported BPB profile schema")
	}
	var n Settings
	original, err := json.Marshal(s)
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(original, &n); err != nil {
		return s, err
	}
	apply := func(src object, mapping map[string]string) error {
		dst := object{}
		for from, to := range mapping {
			if value, exists := src[from]; exists {
				dst[to] = value
			}
		}
		b, e := json.Marshal(dst)
		if e != nil {
			return e
		}
		return json.Unmarshal(b, &n)
	}
	general := mapAt(p, "general")
	if e := apply(general, map[string]string{"localDns": "localDns"}); e != nil {
		return s, e
	}
	if v, ok := general["enableFakeDns"].(bool); ok {
		n.FakeDNS = v
	}
	if v, ok := general["enableIpv6"].(bool); ok {
		if v {
			n.DNSStrategy = "UseIP"
		} else {
			n.DNSStrategy = "UseIPv4"
		}
	}
	vless := mapAt(p, "vless")
	if e := apply(vless, map[string]string{"ports": "workerPorts", "cleanIps": "cleanIps", "tlsFingerprint": "tlsFingerprint", "speedTestInterval": "probeInterval", "enableTcpFastOpen": "tcpFastOpen", "proxyIps": "workerProxyIps", "fallbackUrl": "workerFallback"}); e != nil {
		return s, e
	}
	if dns := mapAt(vless, "remoteDns"); len(dns) > 0 {
		if n.DNSByMode == nil {
			n.DNSByMode = map[string]string{}
		}
		address, _ := dns["server"].(string)
		if address != "" {
			n.DNSByMode["worker"] = address
			if n.Mode == "worker" {
				n.DNS = address
			}
		}
		if ips, ok := dns["bootstrapIps"].([]any); ok && len(ips) > 0 {
			u, e := url.Parse(address)
			if e != nil {
				return s, e
			}
			if n.DNSHosts == nil {
				n.DNSHosts = map[string][]string{}
			}
			var values []string
			for _, ip := range ips {
				v, ok := ip.(string)
				if !ok {
					return s, fmt.Errorf("invalid BPB bootstrap IP")
				}
				values = append(values, v)
			}
			n.DNSHosts[u.Hostname()] = values
		}
	}
	ech := mapAt(vless, "ech")
	if enabled, ok := ech["enabled"].(bool); ok {
		n.ECHDNS = ""
		if enabled {
			n.ECHDNS, _ = ech["dns"].(string)
			if n.ECHDNS == "" {
				n.ECHDNS = "8.8.8.8"
			}
		}
	}
	for _, section := range []string{"warp", "masque"} {
		cfg := mapAt(p, section)
		if address, ok := cfg["remoteDns"].(string); ok {
			if n.DNSByMode == nil {
				n.DNSByMode = map[string]string{}
			}
			n.DNSByMode[section] = address
			if n.Mode == section {
				n.DNS = address
			}
		}
		if n.Mode == section {
			if e := apply(cfg, map[string]string{"endpoints": "endpoints", "speedTestInterval": "probeInterval"}); e != nil {
				return s, e
			}
			if len(n.Endpoints) > 0 {
				n.Endpoint = n.Endpoints[0]
			}
		}
		mapping := map[string]string{"keepAlive": "warpKeepAlive", "enableReservedBytes": "warpReserved"}
		if section == "masque" {
			mapping = map[string]string{"transport": "transport", "sni": "sni", "tlsFingerprint": "tlsFingerprint", "fragments": "masqueFragments", "udpNoises": "masqueNoises"}
		}
		if e := apply(cfg, mapping); e != nil {
			return s, e
		}
	}
	if e := apply(p, map[string]string{"fragments": "fragments", "udpNoises": "udpNoises"}); e != nil {
		return s, e
	}
	if e := apply(mapAt(p, "serverless"), map[string]string{"fragments": "serverlessFragments", "udpNoises": "serverlessNoises"}); e != nil {
		return s, e
	}
	dnsSettings := mapAt(mapAt(p, "serverless"), "dnsSettings")
	if e := apply(dnsSettings, map[string]string{"dnsType": "serverlessDnsType", "fakeSniServers": "fakeSniDns"}); e != nil {
		return s, e
	}
	if servers, exists := dnsSettings["servers"]; exists {
		n.ServerlessDNSServers = nil
		for _, raw := range asArray(servers) {
			server, _ := raw.(object)
			for _, address := range asArray(server["addresses"]) {
				if value, ok := address.(string); ok {
					n.ServerlessDNSServers = append(n.ServerlessDNSServers, value)
				}
			}
		}
	}
	if reset, ok := p["udpNoiseReset"]; ok {
		b, _ := json.Marshal(reset)
		var r MaskRange
		if e := json.Unmarshal(b, &r); e != nil || !r.valid() {
			return s, fmt.Errorf("invalid BPB noise reset")
		}
		n.NoiseReset = r.text()
	}
	n.BPBRouting = mapAt(p, "routing")
	if value, ok := general["antiSanctionDns"].(string); ok {
		n.AntiSanctionDNS = value
	}
	if e := apply(p, map[string]string{"chains": "chains"}); e != nil {
		return s, e
	}
	n.UpstreamRelay, _ = vless["upstreamProxy"].(string)
	n.CustomCDN = mapAt(vless, "customCdn")
	if remote, ok := general["remoteDns"].(string); ok {
		n.DNS = remote
	}
	migrateRemoteDNS(&n)
	if e := validate(n, false); e != nil {
		return s, e
	}
	return n, nil
}
