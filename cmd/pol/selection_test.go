package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEndpointPoolAndChainCore(t *testing.T) {
	s := defaults()
	s.Host = "test.workers.dev"
	s.UUID = "12345678-1234-1234-1234-123456789abc"
	s.Address = "104.16.1.1"
	s.BestPing = true
	s.CleanIPs = []string{"104.16.1.2"}
	s.Chain = "socks5://127.0.0.1:9999"
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	if cfg["burstObservatory"] == nil {
		t.Fatal("best-ping monitoring missing")
	}
	routing := cfg["routing"].(map[string]any)
	if len(routing["balancers"].([]any)) != 1 {
		t.Fatal("pool balancer missing")
	}
	if core := os.Getenv("POL_TEST_XRAY"); core != "" {
		b, _ := json.Marshal(cfg)
		path := filepath.Join(t.TempDir(), "xray.json")
		if err = os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
		if out, e := exec.Command(core, "run", "-test", "-config", path).CombinedOutput(); e != nil {
			t.Fatalf("pool config rejected: %s", out)
		}
	}
}

func TestUnboundedNoiseRejected(t *testing.T) {
	s := defaults()
	s.UDPNoises = []NoiseRule{{Type: "rand", Count: 1000000}}
	if err := validate(s, false); err == nil {
		t.Fatal("unbounded noise amplification accepted")
	}
}
