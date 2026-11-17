package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Opt-in probes run separate loopback listeners and never change LAN routing.
func TestSavedOutboundHTTPS(t *testing.T) {
	if os.Getenv("POL_TEST_LIVE") != "1" {
		t.Skip("set POL_TEST_LIVE on the router to probe saved accounts")
	}
	saved, err := load()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"worker", "warp", "masque", "serverless"} {
		t.Run(mode, func(t *testing.T) {
			s := saved
			s.Mode = mode
			if ep := s.EndpointByMode[mode]; ep != "" {
				s.Endpoint = ep
			}
			s.DNS = defaultDNSForMode(mode)
			s.ExitMode = "none"
			if mode == "worker" && os.Getenv("POL_TEST_EXIT") != "" {
				s.ExitMode = os.Getenv("POL_TEST_EXIT")
			}
			s.Chain = ""
			s.BestPing = false
			s.RouteIran = false
			s.Transparent = false
			s.Variant = "normal"
			s.Transport = "h2"
			s.ServerlessMode = "packet"
			if mode == "warp" && s.Warp == nil || mode == "masque" && s.Masque == nil || mode == "worker" && (s.Host == "" || s.UUID == "") {
				t.Skip("no saved account")
			}
			cfg, e := buildConfig(s)
			if e != nil {
				t.Fatal(e)
			}
			listener, e := net.Listen("tcp4", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			port := listener.Addr().(*net.TCPAddr).Port
			listener.Close()
			cfg["inbounds"] = []any{object{"tag": "http-in", "listen": "127.0.0.1", "port": port, "protocol": "http", "settings": object{}}}
			if os.Getenv("POL_TEST_MARK") == "255" {
				for _, raw := range cfg["outbounds"].([]any) {
					out := raw.(map[string]any)
					stream, _ := out["streamSettings"].(map[string]any)
					if stream == nil {
						stream = object{}
						out["streamSettings"] = stream
					}
					sock, _ := stream["sockopt"].(map[string]any)
					if sock == nil {
						sock = object{}
						stream["sockopt"] = sock
					}
					sock["mark"] = 255
				}
			}
			b, _ := json.Marshal(cfg)
			path := filepath.Join(t.TempDir(), "live.json")
			if e = os.WriteFile(path, b, 0600); e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 135*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, corePath, "run", "-config", path)
			cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+s.GeoDirectory)
			var coreLog bytes.Buffer
			cmd.Stdout = &coreLog
			cmd.Stderr = &coreLog
			if e = cmd.Start(); e != nil {
				t.Fatal(e)
			}
			defer func() {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				if t.Failed() {
					t.Logf("core: %s", coreLog.String())
				}
			}()
			proxy, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
			tr := &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true}
			defer tr.CloseIdleConnections()
			client := &http.Client{Transport: tr, Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

			if s.ExitMode == "psiphon" || s.ExitMode == "tor" {
				oldRuntime := runtimeDir
				runtimeDir = t.TempDir()
				defer func() { runtimeDir = oldRuntime }()
				if s.ExitMode == "tor" {
					bridgeCtx, bridgeCancel := context.WithTimeout(context.Background(), 20*time.Second)
					bridges, e := fetchTorBridges(bridgeCtx, "auto")
					bridgeCancel()
					if e != nil {
						t.Fatal(e)
					}
					s.TorBridges, _ = bridges["bridges"].([]string)
				}
				if e = prepareExit(s); e != nil {
					t.Fatal(e)
				}
				nativeLog := filepath.Join(t.TempDir(), "native.log")
				file, e := os.Create(nativeLog)
				if e != nil {
					t.Fatal(e)
				}
				defer file.Close()
				native := exec.Command("/usr/lib/pol/psiphon", "-keepalive", "-config", runtimeDir+"/psiphon.json")
				if s.ExitMode == "tor" {
					native = exec.Command("/usr/sbin/tor", "-f", runtimeDir+"/torrc")
				}
				native.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				native.Stdout = file
				native.Stderr = file
				if e = native.Start(); e != nil {
					t.Fatal(e)
				}
				defer func() { _ = syscall.Kill(-native.Process.Pid, syscall.SIGKILL); _ = native.Wait() }()
				ready := false
				for i := 0; i < 35; i++ {
					data, _ := os.ReadFile(nativeLog)
					if strings.Contains(string(data), `"event":"ready"`) || strings.Contains(string(data), "Bootstrapped 100%") {
						ready = true
						break
					}
					time.Sleep(2 * time.Second)
				}
				if !ready {
					t.Fatal("native exit did not become ready within 70 seconds")
				}
				t.Log("native exit ready")
			}
			time.Sleep(time.Second)
			for _, target := range []string{"https://www.gstatic.com/generate_204", "https://chatgpt.com/", "https://www.google.com/"} {
				req, _ := http.NewRequestWithContext(ctx, "GET", target, nil)
				start := time.Now()
				resp, e := client.Do(req)
				if e != nil {
					t.Errorf("%s: request failed: %v", target, e)
					continue
				}
				resp.Body.Close()
				t.Logf("%s: HTTP %d in %s", target, resp.StatusCode, time.Since(start).Round(time.Millisecond))
				if resp.StatusCode >= 500 {
					t.Errorf("upstream returned HTTP %d", resp.StatusCode)
				}
			}
		})
	}
}
