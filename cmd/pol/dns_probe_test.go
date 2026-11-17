package main

import "testing"

func TestDNSProbeRejectsMalformedRequests(t *testing.T) {
	for _, request := range []object{{"type": "unknown"}, {"type": "doh", "host": "bad host"}, {"type": "udp"}, {"type": "udp", "ips": []string{"not-an-ip"}}} {
		if _, e := probeDNS(object{"request": request}); e == nil {
			t.Fatal("invalid probe accepted")
		}
	}
}
