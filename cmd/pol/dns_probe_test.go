package main

import "testing"

func TestDNSProbeRejectsMalformedRequests(t *testing.T) {
	for _, request := range []object{{"type": "unknown"}, {"type": "doh", "host": "bad host"}, {"type": "udp"}, {"type": "udp", "ips": []string{"not-an-ip"}}} {
		if _, e := probeDNS(object{"request": request}); e == nil {
			t.Fatal("invalid probe accepted")
		}
	}
}

func TestAsyncMethodsAreRegistered(t *testing.T) {
	for _, method := range asyncMethods {
		if _, ok := methods()[method]; !ok {
			t.Fatalf("unregistered job %s", method)
		}
	}
	if !contains(asyncMethods, "dnsProbe") || !contains(asyncMethods, "torBridges") || !contains(asyncMethods, "cfDelete") {
		t.Fatal("new jobs missing")
	}
}
