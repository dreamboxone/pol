package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWorkerImportPreservesNetworkPolicy(t *testing.T) {
	s := defaults()
	s.Transparent = false
	s.BypassDomains = []string{"example.ir"}
	next, err := importConfig(s, "vless://12345678-1234-1234-1234-123456789abc@104.16.1.1:8443?type=ws&security=tls&host=test.workers.dev&path=%2F%3Fed%3D2560#test")
	if err != nil {
		t.Fatal(err)
	}
	if next.Host != "test.workers.dev" || next.Address != "104.16.1.1" || next.Port != 8443 || next.Path != "/?ed=2560" {
		t.Fatalf("incorrect Worker import: %+v", next)
	}
	if next.Transparent || len(next.BypassDomains) != 1 {
		t.Fatal("import changed network policy")
	}
}

func TestIncompleteWorkerCannotStart(t *testing.T) {
	s := defaults()
	if err := validate(s, false); err != nil {
		t.Fatal(err)
	}
	if err := validate(s, true); err == nil {
		t.Fatal("incomplete Worker accepted")
	}
	s.Host = "test.workers.dev"
	s.UUID = "12345678-1234-1234-1234-123456789abc"
	if err := validate(s, true); err != nil {
		t.Fatal(err)
	}
	s.LAN = "br-lan\"; flush ruleset"
	if err := validate(s, false); err == nil {
		t.Fatal("unsafe interface accepted")
	}
}

func TestDNSAndPrivateNetworkRulesPrecedeCatchAll(t *testing.T) {
	s := defaults()
	s.Mode = "serverless"
	s.BypassDomains = []string{"example.ir"}
	s.BlockDomains = []string{"ads.example"}
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	rules := cfg["routing"].(map[string]any)["rules"].([]any)
	for i, want := range []string{"dns-out", "block", "direct", "direct", "proxy"} {
		if rules[i].(map[string]any)["outboundTag"] != want {
			t.Fatalf("rule %d has wrong precedence", i)
		}
	}
	for _, in := range cfg["inbounds"].([]any) {
		in := in.(map[string]any)
		if in["tag"] == "socks-in" && in["listen"] != "127.0.0.1" {
			t.Fatal("SOCKS exposed beyond loopback")
		}
	}
}

func TestCustomImportCannotReplaceRouterInbounds(t *testing.T) {
	s, err := importConfig(defaults(), `{"inbounds":[{"listen":"0.0.0.0","port":22}],"outbounds":[{"protocol":"freedom"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg["inbounds"].([]any)) != 6 {
		t.Fatal("import replaced router inbounds")
	}
	if cfg["outbounds"].([]any)[0].(map[string]any)["tag"] != "proxy" {
		t.Fatal("imported outbound not retagged")
	}
}

func TestProxyURIRealityAndChainTransport(t *testing.T) {
	out, err := outboundFromURI("vless://12345678-1234-1234-1234-123456789abc@vpn.example:443?security=reality&sni=front.example&pbk=key&sid=abc&type=tcp")
	if err != nil {
		t.Fatal(err)
	}
	stream := out["streamSettings"].(map[string]any)
	if stream["realitySettings"].(map[string]any)["serverName"] != "front.example" {
		t.Fatal("SNI lost")
	}
	if _, err = outboundFromURI("file:///etc/shadow"); err == nil {
		t.Fatal("unsupported URI accepted")
	}
}

func TestTunnelConfigContainsCredentialsAndObfuscation(t *testing.T) {
	s := defaults()
	s.Mode = "warp"
	s.Variant = "obfuscated"
	s.Warp = &WarpAccount{PrivateKey: "private", PublicKey: "public", IPv6: "2606:4700::1/128", Reserved: []int{1, 2, 3}}
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(cfg)
	for _, want := range []string{`"secretKey":"private"`, `"reserved":[1,2,3]`, `"type":"noise"`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing %s", want)
		}
	}
}

func TestSettingsImportDoesNotEnableService(t *testing.T) {
	s, err := importConfig(defaults(), `{"mode":"serverless","enabled":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if s.Enabled {
		t.Fatal("import enabled service")
	}
}
