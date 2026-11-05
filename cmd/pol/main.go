package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const version = "1.0.0"
const settingsPath = "/etc/pol/settings.json"
const corePath = "/usr/lib/pol/xray"
const runtimeDir = "/var/run/pol"

type object = map[string]any

type Settings struct {
	Enabled           bool            `json:"enabled"`
	Mode              string          `json:"mode"`
	Variant           string          `json:"variant"`
	Host              string          `json:"host"`
	UUID              string          `json:"uuid"`
	Address           string          `json:"address"`
	Port              int             `json:"port"`
	Path              string          `json:"path"`
	DNS               string          `json:"dns"`
	LAN               string          `json:"lan"`
	Transparent       bool            `json:"transparent"`
	FragmentLength    string          `json:"fragmentLength"`
	FragmentDelay     string          `json:"fragmentDelay"`
	Endpoint          string          `json:"endpoint"`
	Transport         string          `json:"transport"`
	SNI               string          `json:"sni"`
	Warp              *WarpAccount    `json:"warp,omitempty"`
	Masque            *MasqueAccount  `json:"masque,omitempty"`
	BypassDomains     []string        `json:"bypassDomains"`
	BlockDomains      []string        `json:"blockDomains"`
	Chain             string          `json:"chain"`
	LogLevel          string          `json:"logLevel"`
	ReleaseRepository string          `json:"releaseRepository"`
	Custom            json.RawMessage `json:"custom,omitempty"`
}

type WarpAccount struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	IPv6       string `json:"ipv6"`
	Reserved   []int  `json:"reserved"`
}

type MasqueAccount struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	IPv4       string `json:"ipv4"`
	IPv6       string `json:"ipv6"`
}

func defaults() Settings {
	return Settings{Mode: "worker", Variant: "normal", Port: 443, Path: "/?ed=2560", DNS: "https://1.1.1.1/dns-query", LAN: "br-lan", Transparent: true, FragmentLength: "100-200", FragmentDelay: "1-1", Endpoint: "162.159.192.1:2408", Transport: "h2", SNI: "engage.cloudflareclient.com", LogLevel: "warning", BypassDomains: []string{}, BlockDomains: []string{}}
}

func load() (Settings, error) {
	s := defaults()
	b, err := os.ReadFile(settingsPath)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(b, &s)
	return s, err
}

func atomicWrite(path string, b []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".pol-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func persist(s Settings) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(settingsPath, b, 0600)
}

func run(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	b, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return string(b), fmt.Errorf("%s: %w: %s", filepath.Base(name), err, strings.TrimSpace(string(b)))
	}
	return string(b), nil
}

func event(msg string) { _, _ = run(2*time.Second, "logger", "-t", "pol-app", "--", msg) }

func running() bool {
	b, err := run(3*time.Second, "ubus", "call", "service", "list", `{"name":"pol"}`)
	if err != nil {
		return false
	}
	var v struct {
		BPB struct {
			Instances map[string]struct {
				Running bool `json:"running"`
			} `json:"instances"`
		} `json:"pol"`
	}
	if json.Unmarshal([]byte(b), &v) != nil {
		return false
	}
	for _, i := range v.BPB.Instances {
		if i.Running {
			return true
		}
	}
	return false
}

func coreVersion() string {
	b, err := run(4*time.Second, corePath, "version")
	if err != nil {
		return "unavailable"
	}
	return strings.Split(strings.TrimSpace(b), "\n")[0]
}

func validate(s Settings, complete bool) error {
	if !regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,15}$`).MatchString(s.LAN) {
		return errors.New("invalid LAN interface")
	}
	if s.Port < 1 || s.Port > 65535 {
		return errors.New("invalid port")
	}
	if !contains([]string{"worker", "warp", "masque", "serverless", "tor", "psiphon", "custom"}, s.Mode) {
		return errors.New("invalid connection mode")
	}
	if !contains([]string{"normal", "fragment", "obfuscated"}, s.Variant) {
		return errors.New("invalid variant")
	}
	if !contains([]string{"debug", "info", "warning", "error", "none"}, s.LogLevel) {
		return errors.New("invalid log level")
	}
	if !contains([]string{"h2", "h3"}, s.Transport) {
		return errors.New("invalid MASQUE transport")
	}
	for _, r := range []string{s.FragmentLength, s.FragmentDelay} {
		if !regexp.MustCompile(`^[0-9]{1,4}(-[0-9]{1,4})?$`).MatchString(r) {
			return errors.New("invalid fragment range")
		}
		var a, b int
		if strings.Contains(r, "-") {
			fmt.Sscanf(r, "%d-%d", &a, &b)
			if a > b {
				return errors.New("fragment range is reversed")
			}
		}
	}
	if len(s.Host) > 253 || strings.ContainsAny(s.Host, " /\r\n\t") {
		return errors.New("invalid Worker hostname")
	}
	if len(s.DNS) > 2048 || len(s.Path) > 4096 {
		return errors.New("value too long")
	}
	if s.ReleaseRepository != "" && !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(s.ReleaseRepository) {
		return errors.New("repository must be owner/name")
	}
	if complete {
		switch s.Mode {
		case "worker":
			if s.Host == "" || !regexp.MustCompile(`^[0-9a-fA-F]{8}(-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$`).MatchString(s.UUID) {
				return errors.New("Worker hostname and valid UUID are required")
			}
		case "warp":
			if s.Warp == nil {
				return errors.New("register or import a WARP account first")
			}
		case "masque":
			if s.Masque == nil {
				return errors.New("register or import a MASQUE account first")
			}
		case "custom":
			if len(s.Custom) == 0 {
				return errors.New("import an Xray configuration first")
			}
		}
	}
	return nil
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func prepare(s Settings) error {
	if !s.Enabled {
		return errors.New("service is disabled")
	}
	if err := validate(s, true); err != nil {
		return err
	}
	cfg, err := buildConfig(s)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err = atomicWrite(runtimeDir+"/xray.json", b, 0600); err != nil {
		return err
	}
	_, err = run(15*time.Second, corePath, "run", "-test", "-config", runtimeDir+"/xray.json")
	return err
}

func activate(s Settings) error {
	if !s.Enabled {
		_, err := run(20*time.Second, "/etc/init.d/pol", "stop")
		_, _ = run(3*time.Second, "/etc/init.d/pol", "disable")
		for i := 0; i < 20 && running(); i++ {
			time.Sleep(250 * time.Millisecond)
		}
		return err
	}
	if err := prepare(s); err != nil {
		return err
	}
	if _, err := run(30*time.Second, "/etc/init.d/pol", "restart"); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:24800", 200*time.Millisecond)
		if err == nil {
			conn.Close()
			if running() {
				_, err = run(3*time.Second, "/etc/init.d/pol", "enable")
				return err
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return errors.New("core did not start; see Reports")
}

func change(s Settings) error {
	old, err := load()
	if err != nil {
		return err
	}
	if err = validate(s, s.Enabled); err != nil {
		return err
	}
	if err = persist(s); err != nil {
		return err
	}
	if err = activate(s); err != nil {
		_ = persist(old)
		restoreErr := activate(old)
		event("Settings rejected; previous settings restored")
		if restoreErr != nil {
			return fmt.Errorf("%w; restore failed: %v", err, restoreErr)
		}
		return err
	}
	event("Settings applied immediately")
	return nil
}

func rpc(method string, input object) (any, error) {
	s, err := load()
	if err != nil {
		return nil, err
	}
	switch method {
	case "settings":
		return s, nil
	case "status":
		_, nerr := run(2*time.Second, "nft", "list", "table", "inet", "pol")
		manager := "ipk"
		if _, e := exec.LookPath("apk"); e == nil {
			manager = "apk"
		}
		return object{"version": version, "coreVersion": coreVersion(), "running": running(), "enabled": s.Enabled, "mode": s.Mode, "transparent": s.Transparent, "firewall": nerr == nil, "packageManager": manager, "job": readJob()}, nil
	case "save":
		b, _ := json.Marshal(input["settings"])
		next := s
		if err = json.Unmarshal(b, &next); err != nil {
			return nil, err
		}
		if next.Mode != s.Mode || next.Transport != s.Transport {
			next.Variant = "normal"
			if next.Mode == "serverless" {
				next.Variant = "fragment"
			}
		}
		if err = change(next); err != nil {
			return nil, err
		}
		return object{"ok": true}, nil
	case "toggle":
		on, ok := input["enabled"].(bool)
		if !ok {
			return nil, errors.New("enabled must be boolean")
		}
		s.Enabled = on
		if err = change(s); err != nil {
			return nil, err
		}
		event(fmt.Sprintf("Main button: enabled=%t", on))
		return object{"ok": true}, nil
	case "logs":
		text, e := run(4*time.Second, "logread")
		if e != nil {
			return nil, e
		}
		kind, _ := input["kind"].(string)
		lines := []string{}
		for _, line := range strings.Split(text, "\n") {
			app := strings.Contains(line, "pol-app")
			core := strings.Contains(line, "pol[") || strings.Contains(line, "pol-core")
			if (kind == "app" && app) || (kind == "core" && core) || (kind != "app" && kind != "core" && (app || core)) {
				lines = append(lines, line)
			}
		}
		if len(lines) > 500 {
			lines = lines[len(lines)-500:]
		}
		return object{"lines": lines}, nil
	case "import":
		text, _ := input["text"].(string)
		next, e := importConfig(s, text)
		if e != nil {
			return nil, e
		}
		if e = change(next); e != nil {
			return nil, e
		}
		return object{"ok": true}, nil
	case "register":
		kind, _ := input["kind"].(string)
		next, e := registerAccount(s, kind)
		if e != nil {
			return nil, e
		}
		if e = change(next); e != nil {
			return nil, e
		}
		return object{"ok": true}, nil
	case "deploy":
		return deployWorker(s, input)
	case "releases":
		return releases(s)
	case "update":
		return update(s, input)
	default:
		return nil, errors.New("unknown method")
	}
}

func methods() object {
	return object{"status": object{}, "settings": object{}, "logs": object{"kind": ""}, "save": object{"settings": object{}}, "toggle": object{"enabled": false}, "import": object{"text": ""}, "register": object{"kind": ""}, "deploy": object{"token": "", "account": "", "name": ""}, "releases": object{}, "update": object{"component": "", "tag": ""}}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: pol list | call METHOD | prepare | network-start | network-stop | version")
		os.Exit(1)
	}
	if os.Args[1] == "version" {
		fmt.Println(version)
		return
	}
	if os.Args[1] == "is-enabled" {
		s, err := load()
		if err != nil || !s.Enabled {
			os.Exit(1)
		}
		return
	}
	if os.Args[1] == "list" {
		_ = json.NewEncoder(os.Stdout).Encode(methods())
		return
	}
	if os.Args[1] == "job" && len(os.Args) == 4 {
		executeJob(os.Args[2], os.Args[3])
		return
	}
	if os.Args[1] == "call" {
		if len(os.Args) != 3 {
			os.Exit(1)
		}
		input := object{}
		dec := json.NewDecoder(io.LimitReader(os.Stdin, 2<<20))
		if err := dec.Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			_ = json.NewEncoder(os.Stdout).Encode(object{"error": "invalid JSON"})
			return
		}
		method := os.Args[2]
		if !contains([]string{"settings", "status", "logs", "releases"}, method) {
			f, err := os.OpenFile("/var/lock/pol.lock", os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				_ = json.NewEncoder(os.Stdout).Encode(object{"error": err.Error()})
				return
			}
			defer f.Close()
			if syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
				_ = json.NewEncoder(os.Stdout).Encode(object{"error": "another operation is in progress"})
				return
			}
			defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		}
		var result any
		var err error
		if contains([]string{"deploy", "register", "update"}, method) {
			result, err = startJob(method, input)
		} else {
			result, err = rpc(method, input)
		}
		if err != nil {
			result = object{"error": err.Error()}
		}
		_ = json.NewEncoder(os.Stdout).Encode(result)
		return
	}
	var err error
	switch os.Args[1] {
	case "prepare":
		var s Settings
		s, err = load()
		if err == nil {
			err = prepare(s)
		}
	case "network-start":
		var s Settings
		s, err = load()
		if err == nil {
			err = networkStart(s)
		}
	case "network-stop":
		err = networkStop()
	default:
		err = errors.New("unknown command")
	}
	if err != nil {
		event(err.Error())
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func inputCommand(name string, args []string, input string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewBufferString(input)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s: %w", name, string(b), err)
	}
	return nil
}
