package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ScanResult struct {
	Address string `json:"address"`
	Latency int64  `json:"latency"`
	Error   string `json:"error,omitempty"`
}

func scanCandidates(s Settings, text string) ([]string, error) {
	if !contains([]string{"worker", "warp", "masque"}, s.Mode) {
		return nil, errors.New("scanner supports Worker, WARP and MASQUE")
	}
	seen := map[string]bool{}
	var candidates []string
	for _, value := range strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' || r == ' ' || r == '\t' }) {
		if seen[value] {
			continue
		}
		host := value
		if s.Mode != "worker" {
			var port string
			var err error
			host, port, err = net.SplitHostPort(value)
			if err != nil {
				return nil, errors.New("use host:port or [IPv6]:port for tunnel endpoints")
			}
			p, err := strconv.Atoi(port)
			if err != nil || p < 1 || p > 65535 {
				return nil, errors.New("invalid endpoint port")
			}
		}
		if net.ParseIP(host) == nil {
			if len(host) > 253 || host == "" {
				return nil, errors.New("invalid scanner hostname")
			}
			for _, r := range host {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-') {
					return nil, errors.New("invalid scanner hostname")
				}
			}
		}
		seen[value] = true
		candidates = append(candidates, value)
	}
	if len(candidates) == 0 || len(candidates) > 12 {
		return nil, errors.New("enter between 1 and 12 unique candidates")
	}
	return candidates, nil
}

// Each candidate must carry an HTTPS request through the actual selected outbound.
// A separate loopback-only core leaves the running service and firewall untouched.
func probeCandidate(s Settings, address string) (int64, error) {
	if s.Mode == "worker" {
		s.Address = address
	} else {
		s.Endpoint = address
	}
	s.Transparent = false
	if err := validate(s, true); err != nil {
		return 0, err
	}
	cfg, err := buildConfig(s)
	if err != nil {
		return 0, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	cfg["inbounds"] = []any{object{"tag": "http-in", "listen": "127.0.0.1", "port": port, "protocol": "http", "settings": object{}}}
	dir, err := os.MkdirTemp("", "pol-scan-*")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(dir)
	data, err := json.Marshal(cfg)
	if err != nil {
		return 0, err
	}
	path := filepath.Join(dir, "xray.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, corePath, "run", "-config", path)
	cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+geoDir)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err = cmd.Start(); err != nil {
		return 0, err
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	ready := false
	for i := 0; i < 30; i++ {
		c, e := net.DialTimeout("tcp", proxyURL.Host, 100*time.Millisecond)
		if e == nil {
			c.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		return 0, errors.New("scanner core did not start")
	}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true, TLSHandshakeTimeout: 6 * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, _ := http.NewRequestWithContext(ctx, "GET", "https://www.gstatic.com/generate_204", nil)
	start := time.Now()
	response, err := client.Do(request)
	if err != nil {
		return 0, errors.New("HTTPS probe failed or timed out")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return 0, fmt.Errorf("probe returned HTTP %d", response.StatusCode)
	}
	return time.Since(start).Milliseconds(), nil
}

func scan(s Settings, text string) (any, error) {
	candidates, err := scanCandidates(s, text)
	if err != nil {
		return nil, err
	}
	if err = validate(s, true); err != nil {
		return nil, err
	}
	results := make([]ScanResult, 0, len(candidates))
	// Sequential probes bound memory use on small routers.
	for _, candidate := range candidates {
		latency, e := probeCandidate(s, candidate)
		r := ScanResult{Address: candidate, Latency: latency}
		if e != nil {
			r.Error = e.Error()
		}
		results = append(results, r)
	}
	sort.SliceStable(results, func(i, j int) bool {
		if (results[i].Error == "") != (results[j].Error == "") {
			return results[i].Error == ""
		}
		return results[i].Latency < results[j].Latency
	})
	event("Endpoint scan completed")
	return object{"results": results, "mode": s.Mode, "variant": s.Variant, "transport": s.Transport}, nil
}
