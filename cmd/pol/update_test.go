package main

import "testing"

func TestReleaseVersionOrder(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"v26.11.9", "26.11.09", 0}, {"v26.11.10", "26.11.9", 1}, {"v1.3.0", "1.3.0", 0}, {"v1.2.9", "1.3.0", -1}, {"v1.3.0", "1.3.0-rc.2", 1}, {"v1.3.0-rc.10", "1.3.0-rc.2", 1}, {"1.3.0+build", "1.3.0", 0}} {
		got, ok := versionOrder(c.a, c.b)
		if !ok || got != c.want {
			t.Errorf("%s vs %s: %d %t", c.a, c.b, got, ok)
		}
	}
	if _, ok := versionOrder("unknown", "1.3.0"); ok {
		t.Fatal("unknown version accepted")
	}
}

func TestReleaseChecksUseOnlyRequestedRepository(t *testing.T) {
	old := lookupRelease
	defer func() { lookupRelease = old }()
	calls := []string{}
	lookupRelease = func(repo, tag string) (Release, error) {
		calls = append(calls, repo)
		return Release{Tag: "v1.3.0"}, nil
	}
	result, err := releases(defaults(), "app")
	if err != nil {
		t.Fatal(err)
	}
	r := result.(map[string]any)
	if len(calls) != 1 || calls[0] != "dreamboxone/pol" || r["appUpToDate"] != true || r["appNewer"] != false {
		t.Fatalf("wrong app result: %v %v", calls, r)
	}
	calls = nil
	result, err = releases(defaults(), "core")
	if err != nil {
		t.Fatal(err)
	}
	r = result.(map[string]any)
	if len(calls) != 1 || calls[0] != "XTLS/Xray-core" {
		t.Fatalf("wrong core repo: %v", calls)
	}
	if _, ok := r["app"]; ok {
		t.Fatal("core check included app result")
	}
	if _, err = releases(defaults(), "other"); err == nil {
		t.Fatal("invalid component accepted")
	}
}
