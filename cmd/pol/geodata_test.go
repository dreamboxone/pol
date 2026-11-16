package main

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func protoBytes(field int, value []byte) []byte {
	b := binary.AppendUvarint(nil, uint64(field<<3|2))
	b = binary.AppendUvarint(b, uint64(len(value)))
	return append(b, value...)
}
func protoInt(field int, value uint64) []byte {
	return binary.AppendUvarint(binary.AppendUvarint(nil, uint64(field<<3)), value)
}
func testGeoFile(kind string) []byte {
	entry := protoBytes(1, []byte("IR"))
	if kind == "geoip" {
		entry = append(entry, protoBytes(2, append(protoBytes(1, []byte{5, 160, 0, 0}), protoInt(2, 16)...))...)
	} else {
		entry = append(entry, protoBytes(2, append(protoInt(1, 2), protoBytes(2, []byte("example.ir"))...))...)
	}
	return protoBytes(1, entry)
}

func TestGeodataExtractionRejectsMalformedDownloads(t *testing.T) {
	for _, kind := range []string{"geoip", "geosite"} {
		got, err := extractIran(testGeoFile(kind), kind)
		if err != nil || len(got) != 1 {
			t.Fatalf("%s: %v %v", kind, got, err)
		}
		for _, bad := range [][]byte{[]byte("<html>error</html>"), {}, testGeoFile(kind)[:4], {0x0a, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}} {
			if _, err = extractIran(bad, kind); err == nil {
				t.Fatal("malformed download accepted")
			}
		}
	}
	if validGeoDomain("x\nserver=8.8.8.8") || validGeoDomain("-example.ir") || !validGeoDomain("example.ir") {
		t.Fatal("unsafe resolver domain validation")
	}
}

func TestIranRulesPrecedeTunnelAndFiltering(t *testing.T) {
	old := geoDir
	geoDir = t.TempDir()
	defer func() { geoDir = old }()
	for _, kind := range []string{"geoip", "geosite"} {
		if err := os.WriteFile(filepath.Join(geoDir, kind+".dat"), testGeoFile(kind), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s := defaults()
	s.RouteIran = true
	s.BlockDomains = []string{"example.ir"}
	rules, err := networkRules(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(rules, "ip daddr @iran4") > strings.Index(rules, "tproxy ip to :24810") || !strings.Contains(rules, "redirect to :24814") || !strings.Contains(rules, "iran_domains6") {
		t.Fatal("Iran split is not before tunnel capture")
	}
	cfg, err := buildConfig(s)
	if err != nil {
		t.Fatal(err)
	}
	routing := cfg["routing"].(object)["rules"].([]any)
	if routing[2].(object)["outboundTag"] != "direct" || routing[4].(object)["outboundTag"] != "block" {
		t.Fatal("Iran split must precede domain filtering")
	}
	resolver := iranResolverConfig(s, []string{"example.ir"})
	for _, want := range []string{"no-resolv", "server=127.0.0.1#24812", "server=/ir/example.ir/" + s.IranDNS + "#53", "4#inet#pol#iran_domains4,6#inet#pol#iran_domains6"} {
		if !strings.Contains(resolver, want) {
			t.Fatalf("resolver missing %s", want)
		}
	}
}

func TestOfficialIranGeodata(t *testing.T) {
	dir := os.Getenv("POL_TEST_GEODATA")
	if dir == "" {
		t.Skip("set POL_TEST_GEODATA for real routing data validation")
	}
	old := geoDir
	geoDir = dir
	defer func() { geoDir = old }()
	for _, kind := range []string{"geoip", "geosite"} {
		entries, err := loadIran(kind)
		if err != nil || len(entries) < 100 {
			t.Fatalf("%s: %d entries, %v", kind, len(entries), err)
		}
		t.Logf("%s: %d Iranian entries", kind, len(entries))
	}
	s := defaults()
	s.RouteIran = true
	s.Host = "test.workers.dev"
	s.UUID = "12345678-1234-1234-1234-123456789abc"
	if core := os.Getenv("POL_TEST_XRAY"); core != "" {
		cfg, err := buildConfig(s)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(cfg)
		path := filepath.Join(t.TempDir(), "xray.json")
		_ = os.WriteFile(path, b, 0600)
		cmd := exec.Command(core, "run", "-test", "-config", path)
		cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+geoDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Xray rejected real geodata: %v\n%s", err, out)
		}
	}
	if out := os.Getenv("POL_TEST_OUTPUT"); out != "" {
		_ = os.MkdirAll(out, 0700)
		rules, err := networkRules(s)
		if err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(out, "rules.nft"), []byte(rules), 0600)
		domains, _ := loadIran("geosite")
		_ = os.WriteFile(filepath.Join(out, "dnsmasq.conf"), []byte(iranResolverConfig(s, domains)), 0600)
	}
}

func TestTrafficCountersAndRestartAccounting(t *testing.T) {
	for _, data := range []string{`{"stat":[{"name":"inbound>>>lan-in>>>traffic>>>uplink","value":100},{"name":"inbound>>>lan6-in>>>traffic>>>downlink","value":200},{"name":"inbound>>>dns-in>>>traffic>>>downlink","value":9999}]}`, `{"stat":[{"name":"inbound>>>lan-in>>>traffic>>>uplink","value":"100"},{"name":"inbound>>>lan6-in>>>traffic>>>downlink","value":"200"}]}`} {
		got, err := trafficCounters([]byte(data))
		if err != nil || got.Up != 100 || got.Down != 200 {
			t.Fatalf("counter parsing: %+v %v", got, err)
		}
	}
	db := trafficDB{Days: map[string]trafficDay{}}
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	trafficAdd(&db, "first", trafficDay{100, 200}, now)
	trafficAdd(&db, "first", trafficDay{150, 300}, now)
	trafficAdd(&db, "second", trafficDay{20, 40}, now)
	if db.Days["2026-11-16"] != (trafficDay{170, 340}) {
		t.Fatal("restart double-counted counters")
	}
	db.Days["2026-11-10"] = trafficDay{10, 20}
	db.Days["2026-10-18"] = trafficDay{5, 5}
	db.Days["2026-10-17"] = trafficDay{999, 999}
	usage := usageSummary(db, now)
	if usage["today"].(trafficDay).Up != 170 || usage["week"].(trafficDay).Up != 180 || usage["month"].(trafficDay).Up != 185 {
		t.Fatal("usage date windows incorrect")
	}
}

func TestIranBypassRequiresBothValidFilesEvenWhileServiceOff(t *testing.T) {
	s := defaults()
	s.GeoDirectory = t.TempDir()
	s.RouteIran = true
	s.Enabled = false
	if requireIranData(s) == nil {
		t.Fatal("enabled bypass without files")
	}
	if err := os.WriteFile(filepath.Join(s.GeoDirectory, "geoip.dat"), testGeoFile("geoip"), 0600); err != nil {
		t.Fatal(err)
	}
	if requireIranData(s) == nil {
		t.Fatal("enabled bypass with only GeoIP")
	}
	if err := os.WriteFile(filepath.Join(s.GeoDirectory, "geosite.dat"), []byte("<html>bad download</html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if requireIranData(s) == nil {
		t.Fatal("enabled bypass with corrupt GeoSite")
	}
	if err := os.WriteFile(filepath.Join(s.GeoDirectory, "geosite.dat"), testGeoFile("geosite"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := requireIranData(s); err != nil {
		t.Fatal(err)
	}
	old := geoDir
	geoDir = s.GeoDirectory
	defer func() { geoDir = old }()
	if geoStatus()["geosite"].(map[string]any)["ready"] != true {
		t.Fatal("valid data shown as unavailable")
	}
	if err := os.WriteFile(filepath.Join(s.GeoDirectory, "geosite.dat"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if geoStatus()["geosite"].(map[string]any)["ready"] != false {
		t.Fatal("corrupt data shown as ready")
	}
	s.RouteIran = false
	if err := requireIranData(s); err != nil {
		t.Fatal("disabled bypass must allow recovery")
	}
}
