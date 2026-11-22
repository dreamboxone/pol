package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"

	sshproxy "pol-openwrt/third_party/bpb/ssh"
)

func TestImportedChainSelectionIsPerConnection(t *testing.T) {
	s, err := importConfig(defaults(), `{"format":"bpb-settings","profile":{"schemaVersion":1,"chains":[{"id":"test","name":"Test","link":"socks5://127.0.0.1:1080"}]}}`)
	if err != nil {
		t.Fatal(err)
	}
	s.ExitChains = map[string]string{"serverless": "test"}
	if effectiveChain(s) != "" {
		t.Fatal("a saved chain was applied to the wrong mode")
	}
	s.Mode = "serverless"
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	rules := cfg["routing"].(object)["rules"].([]any)
	if rules[len(rules)-1].(object)["outboundTag"] != "chain" {
		t.Fatal("selected chain was not applied")
	}
}

func TestSSHTCPOnRouter(t *testing.T) {
	if os.Getenv("POL_TEST_LIVE_ROUTER") != "1" {
		t.Skip("requires an unused SSH proxy port on the router")
	}
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "SSH TCP fixture")
	}))
	defer fixture.Close()
	cfg := &ssh.ServerConfig{PasswordCallback: func(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		if meta.User() != "fixture" || string(password) != "fixture-password" {
			return nil, io.ErrUnexpectedEOF
		}
		return &ssh.Permissions{}, nil
	}}
	cfg.AddHostKey(signer)
	go func() {
		raw, err := listener.Accept()
		if err != nil {
			return
		}
		defer raw.Close()
		conn, channels, requests, err := ssh.NewServerConn(raw, cfg)
		if err != nil {
			return
		}
		defer conn.Close()
		go ssh.DiscardRequests(requests)
		for incoming := range channels {
			if incoming.ChannelType() != "direct-tcpip" {
				_ = incoming.Reject(ssh.UnknownChannelType, "fixture only supports TCP forwarding")
				continue
			}
			channel, requests, err := incoming.Accept()
			if err != nil {
				return
			}
			go ssh.DiscardRequests(requests)
			go func() {
				defer channel.Close()
				upstream, err := net.Dial("tcp", fixture.Listener.Addr().String())
				if err != nil {
					return
				}
				defer upstream.Close()
				go func() { _, _ = io.Copy(upstream, channel) }()
				_, _ = io.Copy(channel, upstream)
			}()
		}
	}()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	portNumber, _ := strconv.Atoi(port)
	server, err := sshproxy.Start(sshproxy.Config{
		Host: "127.0.0.1", Port: portNumber, User: "fixture", Password: "fixture-password",
		HostKey: ssh.FingerprintSHA256(signer.PublicKey()),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Stop()
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:24807", nil, &net.Dialer{Timeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: &http.Transport{DialContext: dialer.(proxy.ContextDialer).DialContext}, Timeout: 5 * time.Second}
	response, err := client.Get(fixture.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || string(body) != "SSH TCP fixture" {
		t.Fatalf("unexpected forwarded response: %q, %v", body, err)
	}
}

func TestServerlessDNSProtocolsWithCore(t *testing.T) {
	core := os.Getenv("POL_TEST_XRAY")
	for _, test := range []struct{ kind, server, want string }{
		{"udp", "8.8.8.8", "8.8.8.8"},
		{"tcp", "1.1.1.1", "tcp+local://1.1.1.1"},
		{"doh", "dns.google", "https+local://dns.google/dns-query"},
		{"doq", "dns.quad9.net", "quic+local://dns.quad9.net"},
	} {
		t.Run(test.kind, func(t *testing.T) {
			s := defaults()
			s.Mode, s.ServerlessMode, s.ServerlessDNSType = "serverless", "dns", test.kind
			s.ServerlessDNSServers = []string{test.server}
			cfg, err := buildConfig(s)
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg["dns"].(object)["servers"].([]any)[0].(object)["address"]; got != test.want {
				t.Fatalf("DNS server = %v, want %q", got, test.want)
			}
			if core != "" {
				path := writeParityConfig(t, cfg)
				if out, err := exec.Command(core, "run", "-test", "-config", path).CombinedOutput(); err != nil {
					t.Fatalf("Xray rejected config: %s", out)
				}
			}
		})
	}
}

func writeParityConfig(t *testing.T, cfg object) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "xray.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLANProxyOnRouter(t *testing.T) {
	iface, core := os.Getenv("POL_TEST_LAN_INTERFACE"), os.Getenv("POL_TEST_XRAY")
	if iface == "" || core == "" {
		t.Skip("requires OpenWrt LAN and Xray")
	}
	s := defaults()
	s.LAN, s.LANSharing, s.LANUsername, s.LANPassword = iface, true, "fixture", "fixture-password"
	inbounds, err := sharedProxyInbounds(s)
	if err != nil {
		t.Fatal(err)
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "LAN proxy fixture")
	}))
	defer backend.Close()
	in := inbounds[1].(object)
	listener, err := net.Listen("tcp4", net.JoinHostPort(in["listen"].(string), "0"))
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	in["port"] = listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	cfg := object{"inbounds": []any{in}, "outbounds": []any{object{"protocol": "freedom"}}}
	cmd := exec.Command(core, "run", "-config", writeParityConfig(t, cfg))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	for attempt := 0; attempt < 30; attempt++ {
		conn, err := net.DialTimeout("tcp4", address, time.Second)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, authenticated := range []bool{false, true} {
		proxy := &url.URL{Scheme: "http", Host: address}
		want := http.StatusProxyAuthRequired
		if authenticated {
			proxy.User = url.UserPassword(s.LANUsername, s.LANPassword)
			want = http.StatusOK
		}
		tr := &http.Transport{Proxy: http.ProxyURL(proxy)}
		client := &http.Client{Transport: tr, Timeout: 3 * time.Second}
		resp, err := client.Get(backend.URL)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		tr.CloseIdleConnections()
		if resp.StatusCode != want {
			t.Fatalf("authenticated = %v: status %d, want %d", authenticated, resp.StatusCode, want)
		}
	}
}

func TestConfiguredTunnelsOnRouter(t *testing.T) {
	if os.Getenv("POL_TEST_LIVE_ROUTER") != "1" {
		t.Skip("requires configured OpenWrt accounts")
	}
	settings, err := load()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"worker", "warp", "masque"} {
		t.Run(mode, func(t *testing.T) {
			s := settings
			s.Mode, s.Variant = mode, "normal"
			s.Chain, s.ExitChains = "", nil
			candidate := s.EndpointByMode[mode]
			if mode == "worker" {
				if s.Host == "" || s.UUID == "" {
					t.Skip("no Worker profile")
				}
				candidate = s.Address
				if candidate == "" {
					candidate = s.Host
				}
			} else if candidate == "" {
				t.Skip("no saved endpoint")
			}
			candidates := []string{candidate}
			if mode == "worker" {
				candidates = append(candidates, s.Host)
				candidates = append(candidates, s.CleanIPs...)
			}
			seen := map[string]bool{}
			var lastError error
			for _, address := range candidates {
				if seen[address] {
					continue
				}
				seen[address] = true
				latency, err := probeCandidate(s, address)
				if err == nil {
					t.Logf("HTTPS probe succeeded in %d ms", latency)
					return
				}
				lastError = err
				if len(seen) >= 4 {
					break
				}
			}
			t.Fatalf("%d candidates failed: %v", len(seen), lastError)
		})
	}
}
