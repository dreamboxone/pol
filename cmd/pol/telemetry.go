package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type trafficDay struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}
type trafficDB struct {
	Epoch   string                `json:"epoch"`
	Last    trafficDay            `json:"last"`
	Days    map[string]trafficDay `json:"days"`
	Updated int64                 `json:"updated"`
}

func trafficLoad() trafficDB {
	db := trafficDB{Days: map[string]trafficDay{}}
	b, err := os.ReadFile(runtimeDir + "/traffic.json")
	if err != nil {
		b, _ = os.ReadFile("/etc/pol/traffic.json")
	}
	_ = json.Unmarshal(b, &db)
	if db.Days == nil {
		db.Days = map[string]trafficDay{}
	}
	return db
}

func trafficCounters(data []byte) (trafficDay, error) {
	var result struct {
		Stat []struct {
			Name  string      `json:"name"`
			Value json.Number `json:"value"`
		} `json:"stat"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return trafficDay{}, err
	}
	total := trafficDay{}
	for _, counter := range result.Stat {
		parts := strings.Split(counter.Name, ">>>")
		if len(parts) != 4 || parts[0] != "inbound" || !contains([]string{"lan-in", "lan6-in", "socks-in", "http-in"}, parts[1]) {
			continue
		}
		value, err := counter.Value.Int64()
		if err != nil || value < 0 {
			continue
		}
		if parts[3] == "uplink" {
			total.Up += value
		}
		if parts[3] == "downlink" {
			total.Down += value
		}
	}
	return total, nil
}

func trafficAdd(db *trafficDB, epoch string, current trafficDay, now time.Time) {
	delta := current
	if db.Epoch == epoch {
		if current.Up >= db.Last.Up {
			delta.Up -= db.Last.Up
		}
		if current.Down >= db.Last.Down {
			delta.Down -= db.Last.Down
		}
	}
	day := now.Format("2006-01-02")
	value := db.Days[day]
	value.Up += delta.Up
	value.Down += delta.Down
	db.Days[day] = value
	db.Last = current
	db.Epoch = epoch
	db.Updated = now.Unix()
	cutoff := now.AddDate(0, 0, -90).Format("2006-01-02")
	for key := range db.Days {
		if key < cutoff {
			delete(db.Days, key)
		}
	}
}

func routerTime() time.Time {
	offset, err := run(time.Second, "date", "+%z")
	offset = strings.TrimSpace(offset)
	if err == nil && len(offset) == 5 {
		hours, e1 := strconv.Atoi(offset[1:3])
		minutes, e2 := strconv.Atoi(offset[3:])
		if e1 == nil && e2 == nil {
			seconds := hours*3600 + minutes*60
			if offset[0] == '-' {
				seconds = -seconds
			}
			return time.Now().In(time.FixedZone("router", seconds))
		}
	}
	return time.Now()
}

func usageSummary(db trafficDB, now time.Time) object {
	result := object{"updated": db.Updated}
	for _, period := range []struct {
		name string
		days int
	}{{"today", 1}, {"week", 7}, {"month", 30}} {
		start := now.AddDate(0, 0, -period.days+1).Format("2006-01-02")
		end := now.Format("2006-01-02")
		total := trafficDay{}
		for day, value := range db.Days {
			if day >= start && day <= end {
				total.Up += value.Up
				total.Down += value.Down
			}
		}
		result[period.name] = total
	}
	return result
}

type connectionInfo struct {
	Connected bool   `json:"connected"`
	Latency   int64  `json:"latency"`
	ExitIP    string `json:"exitIp"`
	Country   string `json:"country"`
	Checked   int64  `json:"checked"`
}

func readConnection() connectionInfo {
	var info connectionInfo
	b, _ := os.ReadFile(runtimeDir + "/connection.json")
	_ = json.Unmarshal(b, &info)
	if time.Now().Unix()-info.Checked > 90 {
		info = connectionInfo{}
	}
	return info
}

func probeConnection(ctx context.Context) connectionInfo {
	info := connectionInfo{Checked: time.Now().Unix()}
	proxy, _ := url.Parse("http://127.0.0.1:24801")
	transport := &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 7 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.gstatic.com/generate_204", nil)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return info
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		return info
	}
	info.Connected = true
	info.Latency = time.Since(start).Milliseconds()
	req, _ = http.NewRequestWithContext(ctx, "GET", "https://www.cloudflare.com/cdn-cgi/trace", nil)
	resp, err = client.Do(req)
	if err != nil {
		return info
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return info
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
	for _, line := range strings.Split(string(body), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if key == "ip" {
			info.ExitIP = value
		}
		if key == "loc" && len(value) == 2 {
			info.Country = strings.ToUpper(value)
		}
	}
	return info
}

func checkSite(site string) (any, error) {
	targets := map[string]string{"google": "https://www.google.com/generate_204", "youtube": "https://www.youtube.com/", "telegram": "https://telegram.org/"}
	address, ok := targets[site]
	if !ok {
		return nil, fmt.Errorf("unknown connectivity test")
	}
	if !running() {
		return nil, fmt.Errorf("start Pol before checking a site")
	}
	proxy, _ := url.Parse("http://127.0.0.1:24801")
	transport := &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	start := time.Now()
	response, err := client.Get(address)
	result := object{"site": site, "ok": false, "latency": int64(0)}
	if err != nil {
		return result, nil
	}
	defer response.Body.Close()
	result["ok"] = response.StatusCode >= 200 && response.StatusCode < 400
	result["latency"] = time.Since(start).Milliseconds()
	return result, nil
}

func serverName(s Settings) string {
	switch s.Mode {
	case "worker":
		if s.Address != "" {
			return s.Address
		}
		return s.Host
	case "warp", "masque":
		return s.Endpoint
	case "ssh":
		return net.JoinHostPort(s.SSH.Host, fmt.Sprint(s.SSH.Port))
	case "tor":
		return "127.0.0.1:9050"
	case "psiphon":
		return "127.0.0.1:10829"
	case "custom":
		var cfg struct {
			Outbounds []struct {
				Settings struct {
					Address string `json:"address"`
					Vnext   []struct {
						Address string `json:"address"`
					} `json:"vnext"`
					Servers []struct {
						Address string `json:"address"`
					} `json:"servers"`
				} `json:"settings"`
			} `json:"outbounds"`
		}
		if json.Unmarshal(s.Custom, &cfg) == nil && len(cfg.Outbounds) > 0 {
			out := cfg.Outbounds[0].Settings
			if out.Address != "" {
				return out.Address
			}
			if len(out.Vnext) > 0 {
				return out.Vnext[0].Address
			}
			if len(out.Servers) > 0 {
				return out.Servers[0].Address
			}
		}
	}
	return "—"
}

func monitor() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	db := trafficLoad()
	checkpoint := time.Now()
	lastProbe := time.Time{}
	lastGeo := time.Time{}
	persist := func(disk bool) {
		b, _ := json.Marshal(db)
		_ = atomicWrite(runtimeDir+"/traffic.json", b, 0600)
		if disk {
			_ = atomicWrite("/etc/pol/traffic.json", b, 0600)
		}
	}
	defer persist(true)
	sample := func() {
		if time.Since(lastGeo) > 10*time.Minute {
			scheduleGeoUpdate(ctx)
			lastGeo = time.Now()
		}
		if ctx.Err() != nil {
			return
		}
		queryCtx, queryCancel := context.WithTimeout(ctx, 3*time.Second)
		data, err := exec.CommandContext(queryCtx, corePath, "api", "statsquery", "--server=127.0.0.1:24815", "--pattern=inbound").Output()
		queryCancel()
		if err == nil {
			if current, e := trafficCounters(data); e == nil {
				epoch, _ := os.ReadFile(runtimeDir + "/traffic-epoch")
				trafficAdd(&db, string(epoch), current, routerTime())
				disk := time.Since(checkpoint) >= 5*time.Minute
				persist(disk)
				if disk {
					checkpoint = time.Now()
				}
			}
		}
		if time.Since(lastProbe) >= 45*time.Second && ctx.Err() == nil {
			info := probeConnection(ctx)
			if ctx.Err() == nil {
				b, _ := json.Marshal(info)
				_ = atomicWrite(runtimeDir+"/connection.json", b, 0600)
			}
			lastProbe = time.Now()
		}
	}
	sample()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sample()
		}
	}
}

func cachedCoreVersion() string {
	info, err := os.Stat(corePath)
	if err != nil {
		return "unavailable"
	}
	key := fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
	var cached struct {
		Key     string `json:"key"`
		Version string `json:"version"`
	}
	b, _ := os.ReadFile(runtimeDir + "/core-version.json")
	_ = json.Unmarshal(b, &cached)
	if cached.Key == key && cached.Version != "" {
		return cached.Version
	}
	value := coreVersion()
	if value != "unavailable" {
		b, _ = json.Marshal(object{"key": key, "version": value})
		_ = atomicWrite(runtimeDir+"/core-version.json", b, 0600)
	}
	return value
}
