package main

import "testing"

func TestExitModeSelection(t *testing.T) {
	for _, c := range []struct{ mode, exit, want string }{{"tor", "psiphon", "tor"}, {"psiphon", "tor", "psiphon"}, {"serverless", "tor", "tor"}, {"worker", "psiphon", "psiphon"}} {
		s := defaults()
		s.Mode = c.mode
		s.ExitMode = c.exit
		if got := exitMode(s); got != c.want {
			t.Fatalf("%s/%s: %s", c.mode, c.exit, got)
		}
	}
}
func TestInvalidExitCountryCannotBeSaved(t *testing.T) {
	s := defaults()
	s.ExitRegion = "US\nSocksPort 0.0.0.0:9050"
	if err := validate(s, false); err == nil {
		t.Fatal("unsafe exit country accepted")
	}
}
