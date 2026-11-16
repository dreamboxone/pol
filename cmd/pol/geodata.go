package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

var geoDir = "/etc/pol/geo"

const geoBase = "https://raw.githubusercontent.com/Chocolate4U/Iran-v2ray-rules/release/"

type protoField struct {
	number int
	value  uint64
	data   []byte
}

// Only the IR entry is materialized; other protobuf entries are skipped.
func walkProto(data []byte, visit func(protoField) error) error {
	for len(data) > 0 {
		key, n := binary.Uvarint(data)
		if n <= 0 || key>>3 == 0 {
			return errors.New("invalid geodata protobuf")
		}
		data = data[n:]
		f := protoField{number: int(key >> 3)}
		switch key & 7 {
		case 0:
			f.value, n = binary.Uvarint(data)
			if n <= 0 {
				return errors.New("invalid geodata integer")
			}
			data = data[n:]
		case 1:
			if len(data) < 8 {
				return io.ErrUnexpectedEOF
			}
			data = data[8:]
		case 2:
			size, k := binary.Uvarint(data)
			if k <= 0 || size > uint64(len(data)-k) {
				return io.ErrUnexpectedEOF
			}
			f.data = data[k : k+int(size)]
			data = data[k+int(size):]
		case 5:
			if len(data) < 4 {
				return io.ErrUnexpectedEOF
			}
			data = data[4:]
		default:
			return errors.New("unsupported geodata protobuf field")
		}
		if err := visit(f); err != nil {
			return err
		}
	}
	return nil
}

func extractIran(data []byte, kind string) ([]string, error) {
	var result []string
	err := walkProto(data, func(entry protoField) error {
		if entry.number != 1 {
			return nil
		}
		var code string
		if err := walkProto(entry.data, func(f protoField) error {
			if f.number == 1 {
				code = strings.ToLower(string(f.data))
			}
			if kind == "geoip" && f.number == 3 && f.value != 0 {
				return errors.New("reverse GeoIP entries are unsupported")
			}
			return nil
		}); err != nil {
			return err
		}
		if code != "ir" {
			return nil
		}
		return walkProto(entry.data, func(f protoField) error {
			if f.number != 2 {
				return nil
			}
			if kind == "geoip" {
				var ip []byte
				var prefix uint64
				if err := walkProto(f.data, func(c protoField) error {
					if c.number == 1 {
						ip = c.data
					}
					if c.number == 2 {
						prefix = c.value
					}
					return nil
				}); err != nil {
					return err
				}
				addr, ok := netip.AddrFromSlice(ip)
				if !ok || prefix > uint64(addr.BitLen()) {
					return errors.New("invalid GeoIP CIDR")
				}
				result = append(result, netip.PrefixFrom(addr, int(prefix)).Masked().String())
			} else {
				var value string
				var typ uint64
				if err := walkProto(f.data, func(d protoField) error {
					if d.number == 1 {
						typ = d.value
					}
					if d.number == 2 {
						value = strings.ToLower(string(d.data))
					}
					return nil
				}); err != nil {
					return err
				}
				if (typ == 2 || typ == 3) && validGeoDomain(value) {
					result = append(result, value)
				}
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, errors.New("file has no usable IR entries")
	}
	sort.Strings(result)
	unique := result[:0]
	for _, v := range result {
		if len(unique) == 0 || v != unique[len(unique)-1] {
			unique = append(unique, v)
		}
	}
	return unique, nil
}

func validGeoDomain(value string) bool {
	if len(value) == 0 || len(value) > 253 {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

func geoSources(kind string) []object {
	f := kind + ".dat"
	return []object{
		{"url": geoBase + f, "name": "Chocolate4U (IR)"},
		{"url": geoBase + kind + "-lite.dat", "name": "Chocolate4U Lite (IR)"},
		{"url": "https://github.com/Chocolate4U/Iran-v2ray-rules/releases/latest/download/" + f, "name": "Chocolate4U (IR, release)"},
		{"url": "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/" + f, "name": "Loyalsoldier"},
		{"url": "https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/" + f, "name": "Loyalsoldier (CDN)"},
		{"url": "https://github.com/MetaCubeX/meta-rules-dat/releases/latest/download/" + f, "name": "MetaCubeX"},
		{"url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/" + f, "name": "MetaCubeX (CDN)"},
		{"url": "https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/" + f, "name": "runetfreedom (RU)"},
	}
}

func geoStatus() object {
	result := object{"directory": geoDir}
	path := geoDir
	for {
		var st syscall.Statfs_t
		if syscall.Statfs(path, &st) == nil {
			result["free"] = uint64(st.Bavail) * uint64(st.Bsize)
			break
		}
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	for _, kind := range []string{"geoip", "geosite"} {
		entry := object{"sources": geoSources(kind), "ready": false, "size": int64(0)}
		if info, err := os.Stat(filepath.Join(geoDir, kind+".dat")); err == nil {
			if _, e := loadIran(kind); e == nil {
				entry["ready"] = true
			} else {
				entry["error"] = e.Error()
			}
			entry["size"] = info.Size()
			entry["updated"] = info.ModTime().Unix()
		}
		if b, err := os.ReadFile(filepath.Join(geoDir, kind+".json")); err == nil {
			var metadata object
			if json.Unmarshal(b, &metadata) == nil {
				entry["source"] = metadata["source"]
				entry["sha256"] = metadata["sha256"]
				entry["entries"] = metadata["entries"]
			}
		}
		result[kind] = entry
	}
	return result
}

func updateGeo(s Settings, kind string) (any, error) {
	if !validGeoDirectory(s.GeoDirectory) {
		return nil, errors.New("invalid routing data directory")
	}
	geoDir = s.GeoDirectory
	if kind != "geoip" && kind != "geosite" {
		return nil, errors.New("choose geoip or geosite")
	}
	address := s.GeoIPURL
	if kind == "geosite" {
		address = s.GeoSiteURL
	}
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return nil, errors.New("geodata source must be an HTTPS URL")
	}
	if err = os.MkdirAll(geoDir, 0700); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(geoDir, ".download-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	client := &http.Client{Timeout: 150 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || len(via) > 10 {
			return errors.New("unsafe geodata redirect")
		}
		return nil
	}}
	hash := sha256.New()
	var n int64
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
		if err = tmp.Truncate(0); err != nil {
			return nil, err
		}
		if _, err = tmp.Seek(0, 0); err != nil {
			return nil, err
		}
		hash.Reset()
		var resp *http.Response
		reportUpdateProgress("connecting", 0, 0)
		resp, err = client.Get(address)
		if err != nil {
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("geodata download HTTP %d", resp.StatusCode)
		}
		reportUpdateProgress("downloading", 0, resp.ContentLength)
		progress := &downloadProgress{total: resp.ContentLength}
		n, err = io.Copy(io.MultiWriter(tmp, hash, progress), io.LimitReader(resp.Body, 64<<20+1))
		resp.Body.Close()
		if n > 64<<20 {
			return nil, errors.New("geodata exceeds 64 MiB")
		}
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("geodata transfer failed: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, err
	}
	reportUpdateProgress("verifying", n, n)
	entries, err := extractIran(data, kind)
	if err != nil {
		return nil, fmt.Errorf("invalid %s file: %w", kind, err)
	}
	data = nil
	path := filepath.Join(geoDir, kind+".dat")
	previous := path + ".previous"
	hadOld := false
	if err = backupFile(path, previous); err == nil {
		hadOld = true
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	rollback := func() {
		if hadOld {
			_ = os.Rename(previous, path)
		} else {
			_ = os.Remove(path)
		}
	}
	reportUpdateProgress("installing", n, n)
	if err = os.Rename(tmp.Name(), path); err != nil {
		rollback()
		return nil, err
	}
	if s.Enabled && s.RouteIran {
		if kind == "geosite" {
			if err = atomicWrite(runtimeDir+"/geo-reset", []byte("reset\n"), 0600); err != nil {
				rollback()
				return nil, err
			}
			defer os.Remove(runtimeDir + "/geo-reset")
		}
		if err = activate(s); err != nil {
			rollback()
			_ = activate(s)
			return nil, fmt.Errorf("geodata restored after activation failure: %w", err)
		}
	}
	metadata, _ := json.Marshal(object{"source": address, "sha256": hex.EncodeToString(hash.Sum(nil)), "entries": len(entries)})
	if err = atomicWrite(filepath.Join(geoDir, kind+".json"), metadata, 0600); err != nil {
		rollback()
		if s.Enabled && s.RouteIran {
			_ = activate(s)
		}
		return nil, err
	}
	_ = os.Remove(previous)
	event("Updated " + kind + " routing data")
	return geoStatus(), nil
}

func loadIran(kind string) ([]string, error) { return readIranAt(geoDir, kind) }

func readIranAt(directory, kind string) ([]string, error) {
	path := filepath.Join(directory, kind+".dat")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 64<<20 {
		return nil, fmt.Errorf("%s is missing or invalid; download it before enabling Iran routing", kind)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s; download it before enabling Iran routing", kind)
	}
	entries, err := extractIran(b, kind)
	if err != nil {
		return nil, fmt.Errorf("%s is invalid or lacks usable IR entries; download it again", kind)
	}
	return entries, nil
}

func requireIranData(s Settings) error {
	if !s.RouteIran {
		return nil
	}
	for _, kind := range []string{"geoip", "geosite"} {
		if _, err := readIranAt(s.GeoDirectory, kind); err != nil {
			return err
		}
	}
	return nil
}

func iranSets(prefixes []string) string {
	var v4, v6 []string
	for _, p := range prefixes {
		if strings.Contains(p, ":") {
			v6 = append(v6, p)
		} else {
			v4 = append(v4, p)
		}
	}
	set := func(name, typ string, values []string) string {
		elements := ""
		if len(values) > 0 {
			elements = "elements = { " + strings.Join(values, ", ") + " };"
		}
		return fmt.Sprintf(" set %s { type %s; flags interval; auto-merge; %s }\n", name, typ, elements)
	}
	return set("iran4", "ipv4_addr", v4) + set("iran6", "ipv6_addr", v6) + " set iran_domains4 { type ipv4_addr; flags timeout; timeout 1h; }\n set iran_domains6 { type ipv6_addr; flags timeout; timeout 1h; }\n"
}

func iranResolverConfig(s Settings, domains []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "port=24814\npid-file=%s/dnsmasq.pid\nno-resolv\nno-poll\nbind-dynamic\ninterface=lo\ninterface=%s\nuser=root\ncache-size=1000\nmax-ttl=3600\nmax-cache-ttl=3600\nserver=127.0.0.1#24812\nlog-facility=-\n", runtimeDir, s.LAN)
	b.WriteString("stop-dns-rebind\nrebind-localhost-ok\n")
	for _, domain := range s.RebindDomains {
		if validGeoDomain(domain) {
			fmt.Fprintf(&b, "rebind-domain-ok=/%s/\n", domain)
		}
	}
	group := ""
	emit := func() {
		if group != "" {
			fmt.Fprintf(&b, "server=%s/%s#53\nnftset=%s/4#inet#pol#iran_domains4,6#inet#pol#iran_domains6\n", group, s.IranDNS, group)
			group = ""
		}
	}
	for _, d := range append([]string{"ir"}, domains...) {
		if len(group)+len(d) > 800 {
			emit()
		}
		group += "/" + d
	}
	emit()
	return b.String()
}

func prepareIran(s Settings) error {
	_ = os.Remove(runtimeDir + "/dnsmasq.conf")
	if !s.RouteIran {
		return nil
	}
	if _, err := loadIran("geoip"); err != nil {
		return err
	}
	domains, err := loadIran("geosite")
	if err != nil {
		return err
	}
	if !s.Transparent {
		return nil
	}
	version, err := run(3*time.Second, "dnsmasq", "--version")
	if err != nil || strings.Contains(version, "no-nftset") || !strings.Contains(version, "nftset") {
		return errors.New("Iran pre-routing requires dnsmasq with nftset support (dnsmasq-full)")
	}
	if net.ParseIP(s.IranDNS) == nil || strings.Contains(s.IranDNS, ":") {
		return errors.New("Iran direct DNS must be an IPv4 address")
	}
	config := iranResolverConfig(s, domains)
	zone, _ := run(time.Second, "uci", "-q", "get", "dhcp.@dnsmasq[0].domain")
	zone = strings.TrimSpace(zone)
	if !validGeoDomain(zone) {
		zone = "lan"
	}
	config += "rebind-domain-ok=/" + zone + "/\n" + "server=/" + zone + "/127.0.0.1#53\nserver=/in-addr.arpa/ip6.arpa/127.0.0.1#53\n"
	if err = atomicWrite(runtimeDir+"/dnsmasq.conf", []byte(config), 0600); err != nil {
		return err
	}
	_, err = run(5*time.Second, "dnsmasq", "--test", "--conf-file="+runtimeDir+"/dnsmasq.conf")
	return err
}

type cachedDomainIP struct {
	IP      string `json:"ip"`
	Expires int64  `json:"expires"`
}
type iranCache struct {
	SiteStamp int64            `json:"siteStamp"`
	Entries   []cachedDomainIP `json:"entries"`
}

func siteStamp() int64 {
	if info, err := os.Stat(filepath.Join(geoDir, "geosite.dat")); err == nil {
		return info.ModTime().UnixNano()
	}
	return 0
}

func snapshotIranDomains() {
	if _, err := os.Stat(runtimeDir + "/geo-reset"); err == nil {
		_ = os.Remove(runtimeDir + "/iran-cache.json")
		return
	}
	cache := iranCache{SiteStamp: siteStamp()}
	if cache.SiteStamp == 0 {
		return
	}
	for _, name := range []string{"iran_domains4", "iran_domains6"} {
		out, err := run(2*time.Second, "nft", "-j", "list", "set", "inet", "pol", name)
		if err != nil {
			continue
		}
		var data struct {
			NFTables []struct {
				Set struct {
					Elem []struct {
						Elem struct {
							Value   string `json:"val"`
							Expires int64  `json:"expires"`
						} `json:"elem"`
					} `json:"elem"`
				} `json:"set"`
			} `json:"nftables"`
		}
		if json.Unmarshal([]byte(out), &data) != nil {
			continue
		}
		for _, item := range data.NFTables {
			for _, element := range item.Set.Elem {
				ip, err := netip.ParseAddr(element.Elem.Value)
				if err == nil && element.Elem.Expires > 0 {
					remaining := min(element.Elem.Expires, 3600)
					cache.Entries = append(cache.Entries, cachedDomainIP{IP: ip.String(), Expires: time.Now().Unix() + remaining})
				}
			}
		}
	}
	data, _ := json.Marshal(cache)
	_ = atomicWrite(runtimeDir+"/iran-cache.json", data, 0600)
}

func restoreIranDomains() error {
	b, err := os.ReadFile(runtimeDir + "/iran-cache.json")
	if err != nil {
		return nil
	}
	defer os.Remove(runtimeDir + "/iran-cache.json")
	var cache iranCache
	if json.Unmarshal(b, &cache) != nil || cache.SiteStamp != siteStamp() {
		return nil
	}
	var rules strings.Builder
	for _, entry := range cache.Entries {
		ip, e := netip.ParseAddr(entry.IP)
		remaining := min(entry.Expires-time.Now().Unix(), 3600)
		if e != nil || remaining <= 0 {
			continue
		}
		name := "iran_domains6"
		if ip.Is4() {
			name = "iran_domains4"
		}
		fmt.Fprintf(&rules, "add element inet pol %s { %s timeout %ds }\n", name, ip.String(), remaining)
	}
	if rules.Len() == 0 {
		return nil
	}
	return inputCommand("nft", []string{"-f", "-"}, rules.String())
}

func validGeoDirectory(path string) bool {
	if filepath.Clean(path) != path || strings.ContainsAny(path, "\r\n\x00") {
		return false
	}
	return path == "/etc/pol/geo" || strings.HasPrefix(path, "/mnt/") && len(path) > 5 || strings.HasPrefix(path, "/opt/") && len(path) > 5
}

func scheduleGeoUpdate(ctx context.Context) {
	s, err := load()
	if err != nil || !s.Enabled || s.GeoAuto == "disabled" {
		return
	}
	interval := 24 * time.Hour
	if s.GeoAuto == "weekly" {
		interval = 7 * 24 * time.Hour
	}
	job := readJob()
	if job.State == "pending" || job.State == "running" {
		return
	}
	for _, item := range []struct {
		kind    string
		enabled bool
	}{{"geoip", s.GeoIPUpdate}, {"geosite", s.GeoSiteUpdate}} {
		if !item.enabled {
			continue
		}
		info, e := os.Stat(filepath.Join(s.GeoDirectory, item.kind+".dat"))
		if e == nil && time.Since(info.ModTime()) < interval {
			continue
		}
		executable, e := os.Executable()
		if e != nil {
			return
		}
		command := exec.CommandContext(ctx, executable, "call", "geoUpdate")
		command.Stdin = strings.NewReader(fmt.Sprintf(`{"kind":%q}`, item.kind))
		_ = command.Run()
		return
	}
}
