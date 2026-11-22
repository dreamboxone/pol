package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSSHConfigWithNativeDNSFallback(t *testing.T) {
	s := defaults()
	s.Mode, s.ExitMode, s.DNS = "ssh", "tor", "2606:4700:4700::1111"
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	servers := cfg["dns"].(object)["servers"].([]any)
	if servers[0].(object)["address"] != "tcp://[2606:4700:4700::1111]:53" ||
		servers[1].(object)["port"] != 24818 {
		t.Fatalf("unexpected SSH DNS servers: %v", servers)
	}
	if core := os.Getenv("POL_TEST_XRAY"); core != "" {
		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "ssh.json")
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(core, "run", "-test", "-config", path).CombinedOutput(); err != nil {
			t.Fatalf("SSH config rejected: %s", out)
		}
	}
}

func TestDNSModesWithCore(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		fake       bool
	}{{"parallel", "packet", false}, {"private", "dns", false}, {"alternate-sni", "dns", true}} {
		t.Run(tc.name, func(t *testing.T) {
			s := defaults()
			s.Mode = "serverless"
			s.ServerlessMode = tc.mode
			s.FakeSNIDNS = tc.fake
			cfg, err := buildConfig(s)
			if err != nil {
				t.Fatal(err)
			}
			if core := os.Getenv("POL_TEST_XRAY"); core != "" {
				data, _ := json.Marshal(cfg)
				path := filepath.Join(t.TempDir(), "dns.json")
				if err = os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
				if out, e := exec.Command(core, "run", "-test", "-config", path).CombinedOutput(); e != nil {
					t.Fatalf("DNS config rejected: %s", out)
				}
			}
		})
	}
}

func TestBPBFeaturesWithCore(t *testing.T) {
	core := os.Getenv("POL_TEST_XRAY")
	if core == "" {
		t.Skip("requires router core")
	}
	s := defaults()
	s.Mode = "worker"
	s.Host = "test.workers.dev"
	s.UUID = "12345678-1234-1234-1234-123456789abc"
	s.Address = "104.16.1.1"
	s.CustomCDN = object{"enabled": true, "addresses": []any{s.Address}, "host": "cdn.example", "sni": "tls.example"}
	s.UpstreamRelay = "relay.example:8443"
	s.BPBRouting = object{"bypass": object{"domains": []any{"example.ir"}}, "sanctionBypass": object{"domains": []any{"example.net"}}, "block": object{"domains": []any{"blocked.example"}, "presets": object{"quic": true}}}
	cfg, e := buildConfig(s)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(cfg)
	path := filepath.Join(t.TempDir(), "bpb.json")
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
	if out, e := exec.Command(core, "run", "-test", "-config", path).CombinedOutput(); e != nil {
		t.Fatalf("BPB features rejected: %s", out)
	}
}
