package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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
	for i, want := range []string{"api", "dns-out", "proxy", "direct", "direct", "block", "direct", "proxy"} {
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
	if len(cfg["inbounds"].([]any)) != 7 {
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

func TestAdvancedSettingsReachCore(t *testing.T) {
	s := defaults()
	s.Host = "test.workers.dev"
	s.UUID = "12345678-1234-1234-1234-123456789abc"
	s.WorkerSNI = "front.example"
	s.TLSFingerprint = "firefox"
	s.ECHDNS = "2606:4700:4700::1111"
	s.DNSStrategy = "UseIP"
	s.TCPFastOpen = true
	if err := validate(s, true); err != nil {
		t.Fatal(err)
	}
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(cfg)
	for _, want := range []string{`"serverName":"front.example"`, `"fingerprint":"firefox"`, `"echConfigList":"cloudflare-ech.com+udp://[2606:4700:4700::1111]"`, `"tcpFastOpen":true`, `"queryStrategy":"UseIP"`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing %s", want)
		}
	}
	s.Variant = "fragment"
	if err := validate(s, true); err == nil {
		t.Fatal("ECH and fragmentation accepted together")
	}
}

func TestLegacySettingsReceiveAdvancedDefaults(t *testing.T) {
	s := defaults()
	if err := json.Unmarshal([]byte(`{"mode":"serverless","fragmentLength":"100-200"}`), &s); err != nil {
		t.Fatal(err)
	}
	if err := validate(s, true); err != nil {
		t.Fatal(err)
	}
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(cfg)
	if !strings.Contains(string(b), `"type":"noise"`) || !strings.Contains(string(b), `"type":"fragment"`) {
		t.Fatal("serverless must mask both TCP and UDP")
	}
}

func TestWarpTuningAndReservedBytes(t *testing.T) {
	s := defaults()
	s.Mode, s.Variant = "warp", "obfuscated"
	s.Warp = &WarpAccount{PrivateKey: "private", PublicKey: "public", IPv6: "2606:4700::1/128", Reserved: []int{1, 2, 3}}
	s.WarpMTU, s.WarpKeepAlive, s.WarpReserved = 1400, 10, false
	s.NoiseSize, s.NoiseDelay, s.NoiseReset = "20-40", "2-5", "8-12"
	out, err := proxyOutbound(s)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(out)
	for _, want := range []string{`"mtu":1400`, `"keepAlive":10`, `"reserved":[0,0,0]`, `"rand":"20-40"`, `"delay":"2-5"`, `"reset":"8-12"`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing %s", want)
		}
	}
	s.NoiseDelay = "5-2"
	if validate(s, false) == nil {
		t.Fatal("reversed noise range accepted")
	}
}

func TestScanCandidateValidation(t *testing.T) {
	s := defaults()
	candidates, err := scanCandidates(s, "104.16.1.1\n104.16.1.1,2606:4700::1 test.example")
	if err != nil || len(candidates) != 3 {
		t.Fatalf("deduplication failed: %v %v", candidates, err)
	}
	for _, bad := range []string{"", "https://example.com", "example;touch", strings.Repeat("a ", 1) + strings.Join([]string{"b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m"}, " ")} {
		if _, err = scanCandidates(s, bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	s.Mode = "warp"
	if _, err = scanCandidates(s, "[2606:4700::1]:2408\n162.159.192.1:2408"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"162.159.192.1", "test:99999", "[::1]:0"} {
		if _, err = scanCandidates(s, bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	s.Mode = "custom"
	if _, err = scanCandidates(s, "test.example"); err == nil {
		t.Fatal("custom scan accepted")
	}
}

func TestGeneratedConfigsWithXray(t *testing.T) {
	core := os.Getenv("POL_TEST_XRAY")
	if core == "" {
		t.Skip("set POL_TEST_XRAY to validate generated configs with the installed core")
	}
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalECPrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ mode, variant, transport string }{
		{"worker", "normal", "h2"}, {"worker", "fragment", "h2"},
		{"warp", "normal", "h2"}, {"warp", "obfuscated", "h2"},
		{"masque", "normal", "h2"}, {"masque", "fragment", "h2"},
		{"masque", "normal", "h3"}, {"masque", "obfuscated", "h3"},
		{"serverless", "fragment", "h2"}, {"tor", "normal", "h2"},
	} {
		t.Run(tc.mode+"/"+tc.variant+"/"+tc.transport, func(t *testing.T) {
			s := defaults()
			s.Mode, s.Variant, s.Transport = tc.mode, tc.variant, tc.transport
			s.Host, s.UUID = "test.workers.dev", "12345678-1234-1234-1234-123456789abc"
			s.Warp = &WarpAccount{PrivateKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE=", PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE=", IPv6: "2606:4700::1/128", Reserved: []int{1, 2, 3}}
			s.Masque = &MasqueAccount{PrivateKey: base64.StdEncoding.EncodeToString(privateDER), PublicKey: base64.StdEncoding.EncodeToString(publicDER), IPv4: "172.16.0.2/32", IPv6: "2606:4700::1/128"}
			if tc.mode == "worker" && tc.variant == "normal" {
				s.ECHDNS = "1.1.1.1"
			}
			cfg, err := buildConfig(s)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(cfg)
			path := filepath.Join(t.TempDir(), "xray.json")
			if err = os.WriteFile(path, b, 0600); err != nil {
				t.Fatal(err)
			}
			if output, err := exec.Command(core, "run", "-test", "-config", path).CombinedOutput(); err != nil {
				t.Fatalf("core rejected config: %v\n%s", err, output)
			}
		})
	}
}

func TestRouterUptimeParsing(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  int64
	}{
		{"172861.25 345722.50\n", 172861}, {"59.99 10", 59},
		{"", 0}, {"NaN 0", 0}, {"+Inf 0", 0}, {"-3 0", 0}, {"invalid", 0},
	} {
		if got := parseUptime(tc.value); got != tc.want {
			t.Errorf("parseUptime(%q) = %d, want %d", tc.value, got, tc.want)
		}
	}
}

func TestImportedConfigPreservesRoutingGraph(t *testing.T) {
	s := defaults()
	s.Mode = "custom"
	s.Custom = json.RawMessage(`{"dns":{"servers":["https://dns.example/dns-query"]},"routing":{"domainStrategy":"AsIs","balancers":[{"tag":"pool","selector":["edge"]}],"rules":[{"domain":["domain:example.com"],"balancerTag":"pool"}]},"outbounds":[{"tag":"edge","protocol":"freedom"}],"observatory":{"subjectSelector":["edge"]}}`)
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	routing := cfg["routing"].(map[string]any)
	if routing["domainStrategy"] != "AsIs" || len(routing["balancers"].([]any)) != 1 {
		t.Fatal("imported routing was discarded")
	}
	if cfg["outbounds"].([]any)[0].(map[string]any)["tag"] != "edge" {
		t.Fatal("outbound graph tag changed")
	}
	if cfg["dns"].(map[string]any)["servers"].([]any)[0] != "https://dns.example/dns-query" {
		t.Fatal("imported DNS was discarded")
	}
	if cfg["observatory"] == nil {
		t.Fatal("observatory discarded")
	}
}

func TestManagedExitDoesNotLoopBackIntoItself(t *testing.T) {
	for _, mode := range []string{"tor", "psiphon"} {
		s := defaults()
		s.Mode = "serverless"
		s.ExitMode = mode
		cfg, err := buildConfig(s)
		if err != nil {
			t.Fatal(err)
		}
		rules := cfg["routing"].(map[string]any)["rules"].([]any)
		if rules[0].(map[string]any)["outboundTag"] != "proxy" {
			t.Fatal("exit upstream does not use the underlying connection")
		}
		if rules[len(rules)-1].(map[string]any)["outboundTag"] != "managed-exit" {
			t.Fatal("client traffic bypasses final exit")
		}
		found := false
		for _, raw := range cfg["inbounds"].([]any) {
			in := raw.(map[string]any)
			if in["tag"] == "exit-upstream" {
				found = true
				if in["listen"] != "127.0.0.1" {
					t.Fatal("upstream listener is exposed to LAN")
				}
			}
		}
		if !found {
			t.Fatal("missing upstream listener")
		}
	}
}

func TestVlessXHTTPKeepsTransportOptions(t *testing.T) {
	out, err := outboundFromURI("vless://12345678-1234-1234-1234-123456789abc@vpn.example:443?type=xhttp&security=tls&mode=packet-up&host=front.example&path=%2Ftest&fp=firefox")
	if err != nil {
		t.Fatal(err)
	}
	stream := out["streamSettings"].(map[string]any)
	if stream["xhttpSettings"].(map[string]any)["mode"] != "packet-up" {
		t.Fatal("XHTTP mode lost")
	}
	s, err := importConfig(defaults(), "vless://12345678-1234-1234-1234-123456789abc@vpn.example:443?type=ws&security=tls&host=front.example&fp=firefox")
	if err != nil || s.Mode != "custom" {
		t.Fatal("generic VLESS link was incorrectly treated as a Worker")
	}
}
func TestAdditionalProxyImports(t *testing.T) {
	for _, uri := range []string{
		"ss://YWVzLTEyOC1nY206c2VjcmV0@vpn.example:8388",
		"wireguard://AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE%3D@vpn.example:2408?publickey=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE%3D&address=172.16.0.2%2F32&reserved=1,2,3",
		"hysteria2://secret@vpn.example:443?sni=front.example&obfs=salamander&obfs-password=test&mport=443,8443",
	} {
		s, err := importConfig(defaults(), uri)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := buildConfig(s)
		if err != nil {
			t.Fatal(err)
		}
		if len(cfg["outbounds"].([]any)) < 1 {
			t.Fatal("proxy missing")
		}
		if core := os.Getenv("POL_TEST_XRAY"); core != "" {
			data, _ := json.Marshal(cfg)
			path := filepath.Join(t.TempDir(), "import.json")
			if e := os.WriteFile(path, data, 0600); e != nil {
				t.Fatal(e)
			}
			if out, e := exec.Command(core, "run", "-test", "-config", path).CombinedOutput(); e != nil {
				t.Fatalf("import rejected: %s", out)
			}
		}
	}
}

func findRuleWithValue(rules []any, key, value string) int {
	for i, raw := range rules {
		r := raw.(map[string]any)
		switch values := r[key].(type) {
		case []string:
			for _, v := range values {
				if v == value {
					return i
				}
			}
		case []any:
			for _, v := range values {
				if v == value {
					return i
				}
			}
		}
	}
	return -1
}

func TestWarpUsesIsolatedTunAndIPv4DNS(t *testing.T) {
	s := defaults()
	s.Mode = "warp"
	s.Warp = &WarpAccount{PrivateKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE=", PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE=", IPv6: "2606:4700::1/128"}
	out, err := proxyOutbound(s)
	if err != nil {
		t.Fatal(err)
	}
	settings := out["settings"].(map[string]any)
	if settings["noKernelTun"] != true {
		t.Fatal("WARP must not install kernel routes")
	}
	for _, server := range settings["remoteDNS"].([]string) {
		if strings.Contains(server, ":") {
			t.Fatal("WARP bootstrap unexpectedly requires host IPv6")
		}
	}
}
