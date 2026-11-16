package main

import (
	"strings"
	"testing"
)

func TestDirectAddressValidation(t *testing.T) {
	for _, value := range []string{"1.2.3.4", "10.0.0.0/8", "2001:db8::/32", "geoip:ir", "ext:geoip_IR.dat:ir"} {
		if !validDirectIP(value) {
			t.Errorf("rejected %q", value)
		}
	}
	for _, value := range []string{"1.2.3.999", "1.2.3.4; flush ruleset", "ext:../geoip.dat:ir", "ext:/tmp/geoip.dat:ir", "example.com", "geoip:ir\n"} {
		if validDirectIP(value) {
			t.Errorf("accepted %q", value)
		}
	}
}

func TestExplicitDirectAddressesBeforeFiltering(t *testing.T) {
	s := defaults()
	s.Mode = "serverless"
	s.Transparent = true
	s.DirectIPs = []string{"203.0.113.9", "2001:db8::1/64", "geoip:ir"}
	s.BypassDomains = []string{"example.com"}
	s.BlockDomains = []string{"example.com"}
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	rules := cfg["routing"].(map[string]any)["rules"].([]any)
	if rules[2].(map[string]any)["outboundTag"] != "direct" || rules[3].(map[string]any)["outboundTag"] != "direct" || rules[4].(map[string]any)["outboundTag"] != "block" {
		t.Fatal("direct exceptions must precede filtering")
	}
	text := literalDirectRules(s.DirectIPs)
	if !strings.Contains(text, "203.0.113.9/32") || !strings.Contains(text, "2001:db8::/64") || strings.Contains(text, "geoip:") {
		t.Fatalf("invalid literal rules: %s", text)
	}
}

func TestRebindExceptionsRestrictedToDomains(t *testing.T) {
	for _, domain := range []string{"example.com\nstop-dns-rebind", "http://example.com", "/example.com/", "*.example.com"} {
		s := defaults()
		s.RebindDomains = []string{domain}
		if validate(s, false) == nil {
			t.Errorf("accepted %q", domain)
		}
	}
}
