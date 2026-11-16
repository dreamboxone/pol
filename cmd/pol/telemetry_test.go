package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func freeTestPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestRealCoreTrafficAndImmediateSocketClose(t *testing.T) {
	core := os.Getenv("POL_TEST_XRAY")
	if core == "" {
		t.Skip("set POL_TEST_XRAY for real traffic and shutdown verification")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stream" {
			fmt.Fprint(w, "hello")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		w.Header().Set("Content-Length", "65536")
		_, _ = io.Copy(w, strings.NewReader(strings.Repeat("x", 65536)))
	}))
	defer server.Close()
	s := defaults()
	s.Mode = "serverless"
	s.Transparent = false
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	proxyPort, apiPort := freeTestPort(t), freeTestPort(t)
	cfg["inbounds"] = []any{object{"tag": "http-in", "listen": "127.0.0.1", "port": proxyPort, "protocol": "http", "settings": object{}}, object{"tag": "api-in", "listen": "127.0.0.1", "port": apiPort, "protocol": "dokodemo-door", "settings": object{"address": "127.0.0.1"}}}
	b, _ := json.Marshal(cfg)
	path := filepath.Join(t.TempDir(), "xray.json")
	_ = os.WriteFile(path, b, 0600)
	cmd := exec.Command(core, "run", "-config", path)
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	ready := false
	for i := 0; i < 50; i++ {
		c, e := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort), 50*time.Millisecond)
		if e == nil {
			c.Close()
			ready = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !ready {
		t.Fatal("test core did not listen")
	}
	proxy, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
	transport := &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 4 * time.Second}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || len(data) != 65536 {
		t.Fatalf("traffic request: %d %v", len(data), err)
	}
	stats, err := exec.Command(core, "api", "statsquery", fmt.Sprintf("--server=127.0.0.1:%d", apiPort), "--pattern=inbound").Output()
	if err != nil {
		t.Fatal(err)
	}
	counters, err := trafficCounters(stats)
	if err != nil || counters.Down < 65536 || counters.Up == 0 {
		t.Fatalf("real core traffic counters missing: %+v %v\n%s", counters, err, stats)
	}
	if memory, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", cmd.Process.Pid)); err == nil {
		for _, line := range strings.Split(string(memory), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				t.Log(line)
			}
		}
	}
	response, err = client.Get(server.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	first := make([]byte, 5)
	if _, err = io.ReadFull(response.Body, first); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_ = cmd.Process.Kill()
	_, _ = io.ReadAll(response.Body)
	if time.Since(start) > time.Second {
		t.Fatal("proxy socket did not close immediately")
	}
	t.Logf("real bytes: up=%d down=%d; socket closed in %s", counters.Up, counters.Down, time.Since(start))
}

func TestCancelledConnectivityProbeHasNoRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	info := probeConnection(ctx)
	if info.Connected || time.Since(start) > 500*time.Millisecond {
		t.Fatal("cancelled connectivity probe continued or reported connected")
	}
}
