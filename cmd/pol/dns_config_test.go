package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
