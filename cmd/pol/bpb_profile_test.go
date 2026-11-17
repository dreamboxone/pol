package main

import "testing"

func TestBPBPublicImportPreservesPrivateState(t *testing.T) {
	s := defaults()
	s.UUID = "private-id"
	s.Host = "private.workers.dev"
	s.Enabled = true
	s.Warp = &WarpAccount{PrivateKey: "private-key"}
	next, e := importConfig(s, `{"format":"bpb-settings","profile":{"schemaVersion":1,"vless":{"cleanIps":["104.16.1.1"],"remoteDns":{"server":"https://dns.google/dns-query","bootstrapIps":["8.8.8.8"]}},"general":{"enableFakeDns":true},"warp":{"account":{"privateKey":"untrusted"}}}}`)
	if e != nil {
		t.Fatal(e)
	}
	if next.UUID != s.UUID || next.Host != s.Host || !next.Enabled || next.Warp.PrivateKey != "private-key" {
		t.Fatal("private state replaced")
	}
	if !next.FakeDNS || next.CleanIPs[0] != "104.16.1.1" || next.DNSHosts["dns.google"][0] != "8.8.8.8" {
		t.Fatal("public settings lost")
	}
}
func TestFailedBPBImportIsAtomic(t *testing.T) {
	s := defaults()
	s.DNSByMode = map[string]string{"worker": "1.1.1.1"}
	next, e := importConfig(s, `{"format":"bpb-settings","profile":{"vless":{"remoteDns":{"server":"8.8.8.8"}},"routing":{"block":{"presets":{"unknown-preset":true}}}}}`)
	if e == nil {
		_, e = buildConfig(next)
	}
	if e == nil {
		t.Fatal("unsupported preset silently discarded")
	}
	if s.DNSByMode["worker"] != "1.1.1.1" {
		t.Fatal("failed import mutated existing profile")
	}
}

func TestCustomCDNAndRelayKeepTLSIdentity(t *testing.T) {
	s := defaults()
	s.Host = "origin.workers.dev"
	s.UUID = "12345678-1234-1234-1234-123456789abc"
	s.Address = "104.16.1.1"
	s.CustomCDN = object{"enabled": true, "addresses": []any{"104.16.1.1"}, "host": "cdn.example", "sni": "tls.example"}
	s.UpstreamRelay = "relay.example:8443"
	out, e := proxyOutbound(s)
	if e != nil {
		t.Fatal(e)
	}
	stream := out["streamSettings"].(object)
	if stream["wsSettings"].(object)["host"] != "cdn.example" || stream["tlsSettings"].(object)["serverName"] != "tls.example" {
		t.Fatal("CDN identity lost")
	}
	target := out["settings"].(object)["vnext"].([]any)[0].(object)
	if target["address"] != "relay.example" || target["port"] != 8443 {
		t.Fatal("relay not applied")
	}
	s.Variant = "fragment"
	out, e = proxyOutbound(s)
	if e != nil {
		t.Fatal(e)
	}
	if out["streamSettings"].(object)["wsSettings"].(object)["host"] != s.Host {
		t.Fatal("fragment must use original host")
	}
}
