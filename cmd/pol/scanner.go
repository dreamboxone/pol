package main

import (
	bpbscanner "bpb/scanner"
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
type routerHTTPProbe struct {
	settings Settings
	port     string
	slots    chan struct{}
}

func (p routerHTTPProbe) Close() error { return nil }

func (p routerHTTPProbe) Open(ctx context.Context, ip string) (bpbscanner.HTTPSession, error) {
	select {
	case p.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	s := p.settings
	if s.Mode == "worker" {
		s.Address = ip
	} else {
		s.Endpoint = net.JoinHostPort(ip, p.port)
	}
	session, err := openRouterProbe(ctx, s)
	if err != nil {
		<-p.slots
		return nil, err
	}
	stop := session.stop
	session.stop = func() { stop(); <-p.slots }
	return session, nil
}

type routerHTTPSession struct {
	client *http.Client
	stop   func()
}

func (s *routerHTTPSession) Close() error { s.client.CloseIdleConnections(); s.stop(); return nil }

func probeCandidate(s Settings, address string) (int64, error) {
	if s.Mode == "worker" {
		s.Address = address
	} else {
		s.Endpoint = address
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	session, err := openRouterProbe(ctx, s)
	if err != nil {
		return 0, err
	}
	defer session.Close()
	elapsed, err := session.Ping(ctx)
	return elapsed.Milliseconds(), err
}

func openRouterProbe(ctx context.Context, s Settings) (*routerHTTPSession, error) {
	s.Transparent = false
	s.LANSharing = false
	s.BestPing = false
	s.ExitMode = "none"
	if err := validate(s, true); err != nil {
		return nil, err
	}
	cfg, err := buildConfig(s)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	cfg["inbounds"] = []any{object{"tag": "http-in", "listen": "127.0.0.1", "port": port, "protocol": "http", "settings": object{}}}
	dir, err := os.MkdirTemp("", "pol-scan-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	data, err := json.Marshal(cfg)
	if err != nil {
		cleanup()
		return nil, err
	}
	path := filepath.Join(dir, "xray.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		cleanup()
		return nil, err
	}
	cmd := exec.CommandContext(ctx, corePath, "run", "-config", path)
	cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+geoDir)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err = cmd.Start(); err != nil {
		cleanup()
		return nil, err
	}
	stop := func() { _ = cmd.Process.Kill(); _ = cmd.Wait(); cleanup() }
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
		stop()
		return nil, errors.New("scanner core did not start")
	}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true, TLSHandshakeTimeout: 6 * time.Second}
	client := &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &routerHTTPSession{client: client, stop: stop}, nil
}

func (s *routerHTTPSession) Ping(ctx context.Context) (time.Duration, error) {
	request, _ := http.NewRequestWithContext(ctx, "GET", "https://www.gstatic.com/generate_204", nil)
	start := time.Now()
	response, err := s.client.Do(request)
	if err != nil {
		return 0, errors.New("HTTPS probe failed or timed out")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return 0, fmt.Errorf("probe returned HTTP %d", response.StatusCode)
	}
	return time.Since(start), nil
}

func scan(s Settings, text string, options object) (any, error) {
	if strings.TrimSpace(text) == "auto" {
		return scanRanges(s, options)
	}
	candidates, err := scanCandidates(s, text)
	if err != nil {
		return nil, err
	}
	if err = validate(s, true); err != nil {
		return nil, err
	}
	results := make([]ScanResult, 0, len(candidates))
	scanID := readJob().ID
	started := time.Now()
	reportUpdateProgress("scanning", 0, int64(len(candidates)))
	// Sequential probes bound memory use on small routers.
	for i, candidate := range candidates {
		stopped := false
		for {
			control := readScanControl(scanID)
			if control == "stop" || time.Since(started) > 8*time.Minute {
				stopped = true
				break
			}
			if control != "pause" {
				break
			}
			reportUpdateProgress("paused", int64(i), int64(len(candidates)))
			time.Sleep(250 * time.Millisecond)
		}
		if stopped {
			break
		}
		reportUpdateProgress("scanning", int64(i), int64(len(candidates)))
		latency, e := probeCandidate(s, candidate)
		r := ScanResult{Address: candidate, Latency: latency}
		if e != nil {
			r.Error = e.Error()
		}
		results = append(results, r)
		reportUpdateProgress("scanning", int64(i+1), int64(len(candidates)))
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

func scanRanges(s Settings, input object) (any, error) {
	opts := bpbscanner.Options{ProbeCount: 100, OutputCount: 10, AttemptCount: 3, AttemptInterval: 50, ProbeInterval: 100}
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, err
	}
	if opts.ProbeCount > 1000 || len(opts.CIDRs) > 256 {
		return nil, errors.New("router scans allow at most 1000 probes and 256 ranges")
	}
	opts.EnableIPv6 = s.DNSStrategy != "UseIPv4"
	opts.Mode = s.Variant
	if opts.Mode == "obfuscated" {
		opts.Mode = "noise"
	}
	probe := routerHTTPProbe{settings: s, slots: make(chan struct{}, 2)}
	name := "warp-cidrs.txt"
	switch s.Mode {
	case "worker":
		name = "vless-cidrs.txt"
		opts.Protocol, opts.SNI = bpbscanner.VLESS, s.Host
		if opts.Port == 0 {
			opts.Port = s.Port
		}
		probe.settings.Port = opts.Port
	case "warp", "masque":
		opts.Protocol, opts.SNI = bpbscanner.WARP, "engage.cloudflareclient.com"
		if s.Mode == "masque" {
			opts.Protocol, opts.SNI = bpbscanner.Protocol("masque-"+s.Transport), s.SNI
		}
		if opts.Port == 0 {
			_, port, _ := net.SplitHostPort(s.Endpoint)
			opts.Port, _ = strconv.Atoi(port)
			if opts.Port == 0 {
				opts.Port = 443
				if s.Mode == "warp" {
					opts.Port = 2408
				}
			}
		}
		probe.port = strconv.Itoa(opts.Port)
		probe.settings.Endpoint = net.JoinHostPort("127.0.0.1", probe.port)
	default:
		return nil, errors.New("scanner supports Worker, WARP and MASQUE")
	}
	if len(opts.CIDRs) == 0 {
		b, err := os.ReadFile("/usr/share/pol/scanner/" + name)
		if err != nil {
			return nil, fmt.Errorf("read scanner CIDRs: %w", err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				opts.CIDRs = append(opts.CIDRs, line)
			}
		}
	}
	if err := validate(probe.settings, true); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	scanner := bpbscanner.New(probe)
	if err := scanner.Start(ctx, opts); err != nil {
		return nil, err
	}
	reportUpdateProgress("scanning", 0, int64(opts.ProbeCount))
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	scanID := readJob().ID
	var completed, total int
	var paused bool
	for {
		select {
		case <-ticker.C:
			switch readScanControl(scanID) {
			case "pause":
				if !paused && scanner.Pause() == nil {
					paused = true
					reportUpdateProgress("paused", int64(completed), int64(total))
				}
			case "resume":
				if paused && scanner.Resume() == nil {
					paused = false
					reportUpdateProgress("scanning", int64(completed), int64(total))
				}
			case "stop":
				cancel()
				_ = scanner.Stop()
			}
		case event, ok := <-scanner.Events():
			if !ok {
				return nil, errors.New("scanner ended without a result")
			}
			switch event.Type {
			case "progress":
				if event.Completed > completed {
					completed = event.Completed
				}
				total = event.Total
				if !paused {
					reportUpdateProgress("scanning", int64(completed), int64(total))
				}
			case "error":
				return nil, errors.New(event.Message)
			case "done", "stopped":
				results := make([]ScanResult, 0, len(event.Results))
				for _, result := range event.Results {
					address := result.IP
					if s.Mode != "worker" {
						address = net.JoinHostPort(address, probe.port)
					}
					results = append(results, ScanResult{Address: address, Latency: int64(result.HTTPMillis)})
				}
				return object{"results": results, "mode": s.Mode, "variant": s.Variant, "transport": s.Transport}, nil
			}
		}
	}
}

func readScanControl(id string) string {
	var control struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	}
	b, _ := os.ReadFile(runtimeDir + "/scan-control.json")
	if json.Unmarshal(b, &control) != nil || control.ID != id {
		return ""
	}
	return control.Action
}
func controlScan(action string) (any, error) {
	if action != "pause" && action != "resume" && action != "stop" {
		return nil, errors.New("invalid scan control")
	}
	job := readJob()
	if job.Method != "scan" || (job.State != "running" && job.State != "pending") {
		return nil, errors.New("scanner is not running")
	}
	b, _ := json.Marshal(object{"id": job.ID, "action": action})
	if err := atomicWrite(runtimeDir+"/scan-control.json", b, 0600); err != nil {
		return nil, err
	}
	return object{"ok": true}, nil
}
