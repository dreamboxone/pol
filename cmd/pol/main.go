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
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const version = "1.5.0"
const settingsPath = "/etc/pol/settings.json"
const corePath = "/usr/lib/pol/xray"

var runtimeDir = "/var/run/pol"

type object = map[string]any

type Settings struct {
	DirectIPs         []string        `json:"directIps"`
	RebindDomains     []string        `json:"rebindDomains"`
	GeoDirectory      string          `json:"geoDirectory"`
	GeoAuto           string          `json:"geoAuto"`
	GeoIPUpdate       bool            `json:"geoipUpdate"`
	GeoSiteUpdate     bool            `json:"geositeUpdate"`
	RouteIran         bool            `json:"routeIran"`
	IranDNS           string          `json:"iranDns"`
	GeoIPURL          string          `json:"geoipUrl"`
	GeoSiteURL        string          `json:"geositeUrl"`
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
	TLSFingerprint    string          `json:"tlsFingerprint"`
	WorkerSNI         string          `json:"workerSni"`
	ECHDNS            string          `json:"echDns"`
	TCPFastOpen       bool            `json:"tcpFastOpen"`
	DNSStrategy       string          `json:"dnsStrategy"`
	NoiseSize         string          `json:"noiseSize"`
	NoiseDelay        string          `json:"noiseDelay"`
	NoiseReset        string          `json:"noiseReset"`
	WarpMTU           int             `json:"warpMtu"`
	WarpKeepAlive     int             `json:"warpKeepAlive"`
	WarpReserved      bool            `json:"warpReserved"`
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
	return Settings{DirectIPs: []string{}, RebindDomains: []string{"banksepah.ir", "cbi.ir", "ebanksepah.ir", "esata.ir", "gov.ir", "medu.ir", "qmb.ir", "tamin.ir", "meedc.net", "ntp.faraborddi.com", "qmehr724.com"}, GeoDirectory: "/etc/pol/geo", GeoAuto: "disabled", GeoIPUpdate: true, GeoSiteUpdate: true, IranDNS: "178.22.122.100", GeoIPURL: geoBase + "geoip.dat", GeoSiteURL: geoBase + "geosite.dat", Mode: "worker", Variant: "normal", Port: 443, Path: "/?ed=2560", DNS: "https://1.1.1.1/dns-query", LAN: "br-lan", Transparent: true, FragmentLength: "100-200", FragmentDelay: "1-1", TLSFingerprint: "chrome", DNSStrategy: "UseIPv4", NoiseSize: "10-30", NoiseDelay: "1-3", NoiseReset: "5-10", WarpMTU: 1280, WarpKeepAlive: 30, WarpReserved: true, Endpoint: "162.159.192.1:2408", Transport: "h2", SNI: "engage.cloudflareclient.com", LogLevel: "warning", ReleaseRepository: "dreamboxone/pol", BypassDomains: []string{}, BlockDomains: []string{}}
}

func load() (Settings, error) {
	s := defaults()
	b, err := os.ReadFile(settingsPath)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(b, &s)
	if err == nil {
		geoDir = s.GeoDirectory
	}
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

func backupFile(path, previous string) error {
	link := previous + ".next"
	_ = os.Remove(link)
	if err := os.Link(path, link); err == nil {
		return os.Rename(link, previous)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return atomicWrite(previous, data, info.Mode().Perm())
}

func run(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	if strings.HasPrefix(name, corePath) {
		cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+geoDir)
	}
	b, err := cmd.CombinedOutput()
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
	return v.BPB.Instances["core"].Running
}

func coreVersion() string {
	b, err := run(4*time.Second, corePath, "version")
	if err != nil {
		return "unavailable"
	}
	return strings.Split(strings.TrimSpace(b), "\n")[0]
}

func routerUptime() int64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	return parseUptime(string(b))
}

func parseUptime(value string) int64 {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || !(seconds >= 0 && seconds <= 1e12) {
		return 0
	}
	return int64(seconds)
}

func validate(s Settings, complete bool) error {
	if len(s.DirectIPs) > 1024 || len(s.RebindDomains) > 1024 {
		return errors.New("routing list is too large")
	}
	for _, ip := range s.DirectIPs {
		if !validDirectIP(ip) {
			return errors.New("invalid direct IP, CIDR or GeoIP reference")
		}
	}
	for _, domain := range s.RebindDomains {
		if !validGeoDomain(domain) {
			return errors.New("invalid rebind domain")
		}
	}
	if !validGeoDirectory(s.GeoDirectory) || !contains([]string{"disabled", "daily", "weekly"}, s.GeoAuto) {
		return errors.New("invalid routing data directory or schedule")
	}
	for _, address := range []string{s.GeoIPURL, s.GeoSiteURL} {
		if !strings.HasPrefix(address, "https://") || len(address) > 2048 {
			return errors.New("geodata source must be an HTTPS URL")
		}
	}
	if net.ParseIP(s.IranDNS) == nil || strings.Contains(s.IranDNS, ":") {
		return errors.New("Iran DNS must be an IPv4 address")
	}
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
	if !contains([]string{"chrome", "firefox", "safari", "ios", "android", "edge", "randomized"}, s.TLSFingerprint) {
		return errors.New("invalid TLS fingerprint")
	}
	if !contains([]string{"UseIPv4", "UseIPv6", "UseIP"}, s.DNSStrategy) {
		return errors.New("invalid DNS strategy")
	}
	if s.WarpMTU < 1280 || s.WarpMTU > 1500 || s.WarpKeepAlive < 0 || s.WarpKeepAlive > 65535 {
		return errors.New("WARP MTU must be 1280-1500; keepalive must be 0-65535")
	}
	if len(s.WorkerSNI) > 253 || strings.ContainsAny(s.WorkerSNI, " /\r\n\t") {
		return errors.New("invalid Worker SNI")
	}
	if s.ECHDNS != "" {
		if net.ParseIP(s.ECHDNS) == nil && (!strings.HasPrefix(s.ECHDNS, "https://") || len(s.ECHDNS) > 2048 || strings.ContainsAny(s.ECHDNS, "\r\n\t ")) {
			return errors.New("ECH DNS must be an IP address or HTTPS DoH URL")
		}
		if s.Mode == "worker" && s.Variant == "fragment" {
			return errors.New("disable ECH before using Worker fragmentation")
		}
	}
	for _, r := range []string{s.FragmentLength, s.FragmentDelay, s.NoiseSize, s.NoiseDelay, s.NoiseReset} {
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
	geoDir = s.GeoDirectory
	if err := prepareIran(s); err != nil {
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
	if _, err := os.Stat(runtimeDir + "/stop-request"); err == nil && s.Enabled {
		s.Enabled = false
		if err = persist(s); err != nil {
			return err
		}
	}
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
			ready := true
			if s.RouteIran && s.Transparent {
				resolver, e := net.DialTimeout("tcp", "127.0.0.1:24814", 200*time.Millisecond)
				ready = e == nil
				if resolver != nil {
					resolver.Close()
				}
			}
			if running() && ready {
				_, err = run(3*time.Second, "/etc/init.d/pol", "enable")
				return err
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return errors.New("core did not start; see Reports")
}

func change(s Settings) error {
	if _, e := os.Stat(runtimeDir + "/stop-request"); e == nil {
		s.Enabled = false
	}
	old, err := load()
	if err != nil {
		return err
	}
	if err = validate(s, s.Enabled); err != nil {
		return err
	}
	if err = requireIranData(s); err != nil {
		return err
	}
	if !reflect.DeepEqual(s.RebindDomains, old.RebindDomains) {
		if err = syncRebind(s.RebindDomains); err != nil {
			return err
		}
	}
	if err = persist(s); err != nil {
		_ = syncRebind(old.RebindDomains)
		return err
	}
	runtimeSettings := s
	if !old.RouteIran {
		runtimeSettings.RebindDomains = old.RebindDomains
	}
	runtimeSettings.GeoAuto = old.GeoAuto
	runtimeSettings.GeoIPUpdate = old.GeoIPUpdate
	runtimeSettings.GeoSiteUpdate = old.GeoSiteUpdate
	runtimeSettings.GeoIPURL = old.GeoIPURL
	runtimeSettings.GeoSiteURL = old.GeoSiteURL
	runtimeSettings.ReleaseRepository = old.ReleaseRepository
	if (!s.Enabled && !old.Enabled) || reflect.DeepEqual(runtimeSettings, old) {
		return nil
	}
	if err = activate(s); err != nil {
		_ = persist(old)
		_ = syncRebind(old.RebindDomains)
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
		return object{"version": version, "uptime": routerUptime(), "features": object{"siteChecks": true, "routingLists": true, "geoManager": true}, "coreVersion": cachedCoreVersion(), "usage": usageSummary(trafficLoad(), routerTime()), "connection": readConnection(), "server": serverName(s), "running": running(), "enabled": s.Enabled, "mode": s.Mode, "transparent": s.Transparent, "firewall": nerr == nil, "packageManager": manager, "job": readJob()}, nil
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
		if on {
			_ = os.Remove(runtimeDir + "/stop-request")
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
	case "check":
		site, _ := input["site"].(string)
		return checkSite(site)
	case "geo":
		return geoStatus(), nil
	case "geoUpdate":
		kind, _ := input["kind"].(string)
		return updateGeo(s, kind)
	case "scan":
		text, _ := input["text"].(string)
		return scan(s, text)
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
		component, _ := input["component"].(string)
		return releases(s, component)
	case "update":
		return update(s, input)
	default:
		return nil, errors.New("unknown method")
	}
}

func methods() object {
	return object{"check": object{"site": ""}, "geo": object{}, "geoUpdate": object{"kind": ""}, "scan": object{"text": ""}, "status": object{}, "settings": object{}, "logs": object{"kind": ""}, "save": object{"settings": object{}}, "toggle": object{"enabled": false}, "import": object{"text": ""}, "register": object{"kind": ""}, "deploy": object{"token": "", "account": "", "name": ""}, "releases": object{"component": ""}, "update": object{"component": "", "tag": ""}}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: pol list | call METHOD | prepare | network-start | network-stop | version")
		os.Exit(1)
	}
	if os.Args[1] == "geo-directory" {
		s, e := load()
		if e != nil || !validGeoDirectory(s.GeoDirectory) {
			os.Exit(1)
		}
		fmt.Println(s.GeoDirectory)
		return
	}
	if os.Args[1] == "monitor" {
		monitor()
		return
	}
	if os.Args[1] == "start-epoch" {
		id, e := newUUID()
		if e != nil {
			os.Exit(1)
		}
		if atomicWrite(runtimeDir+"/traffic-epoch", []byte(id), 0600) != nil {
			os.Exit(1)
		}
		_ = os.Remove(runtimeDir + "/connection.json")
		return
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
		if method == "toggle" && input["enabled"] == false {
			_ = atomicWrite(runtimeDir+"/stop-request", []byte("off\n"), 0600)
			cancelJob()
			_, _ = run(12*time.Second, "/etc/init.d/pol", "stop")
		}
		if !contains([]string{"settings", "status", "logs", "releases", "geo"}, method) {
			f, err := os.OpenFile("/var/lock/pol.lock", os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				_ = json.NewEncoder(os.Stdout).Encode(object{"error": err.Error()})
				return
			}
			defer f.Close()
			lockMode := syscall.LOCK_EX | syscall.LOCK_NB
			if method == "toggle" && input["enabled"] == false {
				lockMode = syscall.LOCK_EX
			}
			if syscall.Flock(int(f.Fd()), lockMode) != nil {
				_ = json.NewEncoder(os.Stdout).Encode(object{"error": "another operation is in progress"})
				return
			}
			defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		}
		var result any
		var err error
		if contains([]string{"deploy", "register", "update", "scan", "geoUpdate", "check"}, method) {
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
			err = syncRebind(s.RebindDomains)
		}
		if err == nil {
			err = prepare(s)
		}
	case "network-start":
		var s Settings
		s, err = load()
		if err == nil {
			err = networkStart(s)
		}
	case "network-pause":
		snapshotIranDomains()
		err = networkStop()
	case "network-stop":
		_ = os.Remove(runtimeDir + "/iran-cache.json")
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
