package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func exitMode(s Settings) string {
	if s.Mode == "custom" {
		return ""
	}
	if s.Mode == "tor" || s.Mode == "psiphon" {
		return s.Mode
	}
	return s.ExitMode
}

func prepareExit(s Settings) error {
	mode := exitMode(s)
	if mode != "tor" && mode != "psiphon" {
		return nil
	}
	if s.ExitRegion != "" && !regexp.MustCompile(`^[A-Z]{2}$`).MatchString(s.ExitRegion) {
		return fmt.Errorf("invalid exit country")
	}
	if mode == "psiphon" {
		if _, err := os.Stat("/usr/lib/pol/psiphon"); err != nil {
			return fmt.Errorf("Psiphon engine is not installed: %w", err)
		}
		b, err := os.ReadFile("/usr/share/pol/psiphon-config.json")
		if err != nil {
			return err
		}
		var cfg object
		if err = json.Unmarshal(b, &cfg); err != nil {
			return err
		}
		cfg["DataRootDirectory"] = "/etc/pol/psiphon"
		cfg["ClientPlatform"] = "OpenWrt"
		cfg["EgressRegion"] = s.ExitRegion
		if s.Mode != "psiphon" {
			cfg["UpstreamProxyURL"] = "socks5://127.0.0.1:24828"
		}
		cfg["BpbSocksListen"] = "127.0.0.1:24806"
		cfg["BpbDnsListen"] = "127.0.0.1:24819"
		b, err = json.Marshal(cfg)
		if err != nil {
			return err
		}
		return atomicWrite(runtimeDir+"/psiphon.json", b, 0600)
	}
	if _, err := exec.LookPath("tor"); err != nil {
		return fmt.Errorf("Tor engine is not installed: %w", err)
	}
	if err := os.MkdirAll("/etc/pol/tor", 0700); err != nil {
		return err
	}
	text := "DataDirectory /etc/pol/tor\nSocksPort 127.0.0.1:24805\nDNSPort 127.0.0.1:24818\nControlPort 127.0.0.1:24817\nCookieAuthentication 1\nCookieAuthFile /var/run/pol/tor-control.cookie\nClientOnly 1\nLog notice stdout\nAvoidDiskWrites 1\n"
	if s.Mode != "tor" {
		text += "Socks5Proxy 127.0.0.1:24828\n"
	}
	if s.ExitRegion != "" {
		text += "ExitNodes {" + strings.ToLower(s.ExitRegion) + "}\nStrictNodes 1\n"
	}
	if len(s.TorBridges) > 128 {
		return fmt.Errorf("too many Tor bridges")
	}
	if len(s.TorBridges) > 0 {
		text += "UseBridges 1\n"
		transports := map[string]bool{}
		for _, bridge := range s.TorBridges {
			if strings.ContainsAny(bridge, "\r\n\x00") || len(bridge) > 2048 {
				return fmt.Errorf("invalid Tor bridge")
			}
			bridge = strings.TrimSpace(regexp.MustCompile(`(?i)^bridge\s+`).ReplaceAllString(bridge, ""))
			fields := strings.Fields(bridge)
			if len(fields) < 2 {
				return fmt.Errorf("invalid Tor bridge")
			}
			if !contains([]string{"obfs4", "snowflake", "webtunnel", "meek_lite"}, fields[0]) {
				return fmt.Errorf("unsupported Tor bridge transport")
			}
			transports[fields[0]] = true
			text += "Bridge " + bridge + "\n"
		}
		for transport := range transports {
			path := "/usr/lib/pol/lyrebird"
			if _, err := os.Stat(path); err != nil {
				binary := ""
				switch transport {
				case "obfs4":
					binary = "obfs4proxy"
				case "snowflake":
					binary = "snowflake-client"
				default:
					return fmt.Errorf("Tor bridge transport %s requires Lyrebird", transport)
				}
				var e error
				path, e = exec.LookPath(binary)
				if e != nil {
					return fmt.Errorf("Tor bridge transport %s is not installed", transport)
				}
			}

			text += "ClientTransportPlugin " + transport + " exec " + path + "\n"
		}
	}
	return atomicWrite(runtimeDir+"/torrc", []byte(text), 0600)
}

func readExitState() object {
	var state object
	b, _ := os.ReadFile(runtimeDir + "/exit-state.json")
	_ = json.Unmarshal(b, &state)
	if ready, _ := state["ready"].(bool); ready {
		pid, _ := state["pid"].(float64)
		b, e := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", int(pid)))
		if e != nil || !strings.Contains(string(b), "\x00exit-run") {
			state["ready"] = false
		}
	}
	return state
}
func writeExitState(mode, phase string, ready bool) {
	b, _ := json.Marshal(object{"pid": os.Getpid(), "engine": mode, "phase": phase, "ready": ready, "updated": time.Now().Unix()})
	_ = atomicWrite(runtimeDir+"/exit-state.json", b, 0600)
}
func runExit(s Settings) error {
	mode := exitMode(s)
	if mode != "tor" && mode != "psiphon" {
		return fmt.Errorf("no native exit selected")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	writeExitState(mode, "starting", false)
	if mode == "tor" && s.TorBridgeMode != "direct" && s.TorBridgeMode != "custom" && len(s.TorBridges) == 0 {
		writeExitState(mode, "fetching-bridges", false)
		transport := s.TorBridgeMode
		if transport == "" {
			transport = "auto"
		}
		result, e := fetchTorBridges(ctx, transport)
		if e == nil {
			s.TorBridges, _ = result["bridges"].([]string)
			b, _ := json.Marshal(s.TorBridges)
			_ = atomicWrite("/etc/pol/tor-bridges-cache.json", b, 0600)
		} else {
			b, _ := os.ReadFile("/etc/pol/tor-bridges-cache.json")
			_ = json.Unmarshal(b, &s.TorBridges)
			if len(s.TorBridges) == 0 {
				writeExitState(mode, "bridge-fetch-failed", false)
				return e
			}
		}
	}
	if err := prepareExit(s); err != nil {
		writeExitState(mode, "failed", false)
		return err
	}
	var cmd *exec.Cmd
	if mode == "tor" {
		cmd = exec.Command("/usr/sbin/tor", "-f", runtimeDir+"/torrc")
	} else {
		cmd = exec.Command("/usr/lib/pol/psiphon", "-keepalive", "-config", runtimeDir+"/psiphon.json")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	output, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err = cmd.Start(); err != nil {
		return err
	}
	writeExitState(mode, "starting", false)
	done := make(chan struct{})
	go func() {
		select {
		case <-done:
			return
		case <-ctx.Done():
		}
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-done:
		case <-timer.C:
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}()
	if mode == "tor" {
		go monitorTorControl(ctx)
	}
	scan := bufio.NewScanner(output)
	scan.Buffer(make([]byte, 4096), 1<<20)
	for scan.Scan() {
		line := scan.Text()
		fmt.Println(line)
		if mode == "tor" {
			if strings.Contains(line, "Bootstrapped 100%") {
				writeExitState(mode, "ready", true)
			} else if strings.Contains(line, "Bootstrapped ") {
				if match := regexp.MustCompile(`Bootstrapped (\d+)%`).FindStringSubmatch(line); len(match) > 1 {
					writeExitState(mode, "bootstrap:"+match[1], false)
				}
			}
		} else {
			var notice struct {
				Event string `json:"event"`
				Type  string `json:"noticeType"`
				Data  struct {
					Count int `json:"count"`
				} `json:"data"`
			}
			if json.Unmarshal([]byte(line), &notice) == nil {
				if notice.Event == "ready" {
					writeExitState(mode, "ready", true)
				}
				if notice.Type == "Tunnels" {
					if notice.Data.Count == 0 {
						writeExitState(mode, "reconnecting", false)
					} else {
						writeExitState(mode, "ready", true)
					}
				}
			}
		}
	}
	err = cmd.Wait()
	close(done)
	writeExitState(mode, "stopped", false)
	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return err
	}
	return scan.Err()
}

func fetchTorBridges(ctx context.Context, transport string) (object, error) {
	if !contains([]string{"auto", "obfs4", "snowflake", "webtunnel", "meek_lite"}, transport) {
		return nil, fmt.Errorf("unsupported Tor bridge transport")
	}
	transports := []string{transport}
	if transport == "auto" {
		transports = []string{"obfs4", "snowflake", "webtunnel", "meek_lite"}
	}
	response, err := requestJSONContext(ctx, "POST", "https://bridges.torproject.org/moat/circumvention/settings", map[string]string{"Content-Type": "application/vnd.api+json"}, object{"country": "ir", "transports": transports})
	if err != nil {
		return nil, err
	}
	var bridges []string
	for _, raw := range asArray(response["settings"]) {
		setting, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for _, value := range asArray(mapAt(setting, "bridges")["bridge_strings"]) {
			line, ok := value.(string)
			if !ok {
				continue
			}
			line = strings.TrimSpace(strings.TrimPrefix(line, "Bridge "))
			fields := strings.Fields(line)
			if len(fields) > 1 && contains(transports, fields[0]) && !strings.ContainsAny(line, "\r\n\x00") && len(line) <= 2048 {
				bridges = append(bridges, line)
			}
		}
	}
	if len(bridges) == 0 {
		return nil, fmt.Errorf("Tor Project sent no bridges of this type")
	}
	return object{"bridges": bridges}, nil
}

func monitorTorControl(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	everReady := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		cookie, err := os.ReadFile(runtimeDir + "/tor-control.cookie")
		if err != nil || len(cookie) != 32 {
			continue
		}
		conn, err := net.DialTimeout("tcp", "127.0.0.1:24817", time.Second)
		if err != nil {
			continue
		}
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		reader := bufio.NewReaderSize(conn, 4096)
		_, err = fmt.Fprintf(conn, "AUTHENTICATE %s\r\n", hex.EncodeToString(cookie))
		if err != nil {
			conn.Close()
			continue
		}
		line, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(line, "250") {
			conn.Close()
			continue
		}
		_, err = fmt.Fprint(conn, "GETINFO status/circuit-established\r\n")
		if err != nil {
			conn.Close()
			continue
		}
		line, err = reader.ReadString('\n')
		conn.Close()
		if err != nil || ctx.Err() != nil {
			continue
		}
		if strings.Contains(line, "status/circuit-established=1") {
			everReady = true
			writeExitState("tor", "ready", true)
		} else if everReady {
			writeExitState("tor", "reconnecting", false)
		}
	}
}
