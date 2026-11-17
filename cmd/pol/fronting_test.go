package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDomainFrontingTemplateCore(t *testing.T) {
	core := os.Getenv("POL_TEST_XRAY")
	if core == "" {
		t.Skip("set POL_TEST_XRAY for installed-core validation")
	}
	if _, err := os.Stat("/usr/share/pol/domain-fronting.json"); err != nil {
		t.Skip("domain-fronting asset is not installed")
	}
	if err := ensureFrontingCA(); err != nil {
		t.Fatal(err)
	}
	for _, exit := range []string{"none", "tor", "psiphon"} {
		s := defaults()
		s.Mode = "serverless"
		s.ServerlessMode = "domain-fronting"
		s.ExitMode = exit
		cfg, err := buildConfig(s)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, raw := range cfg["inbounds"].([]any) {
			in := raw.(map[string]any)
			if in["tag"] == "tls-decrypt-h11" {
				found = true
				if in["listen"] != "127.0.0.1" {
					t.Fatal("TLS helper exposed outside loopback")
				}
			}
		}
		if !found {
			t.Fatal("TLS helper listener was discarded")
		}
		b, _ := json.Marshal(cfg)
		path := filepath.Join(t.TempDir(), "fronting.json")
		if err = os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
		if out, e := exec.Command(core, "run", "-test", "-config", path).CombinedOutput(); e != nil {
			t.Fatalf("fronting config rejected: %s", out)
		}
	}
}
