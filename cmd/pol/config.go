package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// OpenWrt adaptation of BPB-VPN's Xray configuration builder.
func fragment(s Settings) object {
	return object{"tcp": []any{object{"type": "fragment", "settings": object{"packets": "tlshello", "lengths": []string{s.FragmentLength}, "delays": []string{s.FragmentDelay}}}}}
}

func noise() object {
	return object{"udp": []any{object{"type": "noise", "settings": object{"reset": "5-10", "noise": []any{object{"rand": "10-30", "randRange": "0-255", "delay": "1-3"}}}}}}
}

func proxyOutbound(s Settings) (object, error) {
	switch s.Mode {
	case "worker":
		address := s.Address
		if address == "" {
			address = s.Host
		}
		stream := object{"network": "ws", "security": "tls", "tlsSettings": object{"serverName": s.Host, "fingerprint": "chrome", "alpn": []string{"http/1.1"}}, "wsSettings": object{"host": s.Host, "path": s.Path}}
		if s.Variant == "fragment" {
			stream["finalmask"] = fragment(s)
		}
		return object{"tag": "proxy", "protocol": "vless", "settings": object{"vnext": []any{object{"address": address, "port": s.Port, "users": []any{object{"id": s.UUID, "encryption": "none"}}}}}, "streamSettings": stream}, nil
	case "warp":
		a := s.Warp
		if a == nil {
			return nil, errors.New("WARP account is missing")
		}
		out := object{"tag": "proxy", "protocol": "wireguard", "settings": object{"secretKey": a.PrivateKey, "address": []string{"172.16.0.2/32", a.IPv6}, "mtu": 1280, "reserved": a.Reserved, "peers": []any{object{"endpoint": s.Endpoint, "publicKey": a.PublicKey, "keepAlive": 30}}}}
		if s.Variant == "obfuscated" {
			out["streamSettings"] = object{"finalmask": noise()}
		}
		return out, nil
	case "masque":
		a := s.Masque
		if a == nil {
			return nil, errors.New("MASQUE account is missing")
		}
		host, port, err := net.SplitHostPort(s.Endpoint)
		if err != nil {
			return nil, errors.New("endpoint must be host:port")
		}
		p, err := strconv.Atoi(port)
		if err != nil || p < 1 || p > 65535 {
			return nil, errors.New("invalid endpoint port")
		}
		tls := object{"serverName": s.SNI, "alpn": []string{s.Transport}}
		if s.Transport == "h2" {
			tls["fingerprint"] = "chrome"
		}
		stream := object{"network": "masque", "security": "tls", "tlsSettings": tls, "masqueSettings": object{"warp": object{"privateKey": a.PrivateKey, "publicKey": a.PublicKey, "address": []string{a.IPv4, a.IPv6}}}}
		if s.Variant != "normal" {
			if s.Transport == "h2" {
				stream["finalmask"] = fragment(s)
			} else {
				stream["finalmask"] = noise()
			}
		}
		return object{"tag": "proxy", "protocol": "masque", "settings": object{"address": host, "port": p}, "streamSettings": stream}, nil
	case "serverless":
		return object{"tag": "proxy", "protocol": "freedom", "settings": object{}, "streamSettings": object{"sockopt": object{"domainStrategy": "UseIP"}, "finalmask": fragment(s)}}, nil
	case "tor":
		return object{"tag": "proxy", "protocol": "socks", "settings": object{"servers": []any{object{"address": "127.0.0.1", "port": 9050}}}}, nil
	case "psiphon":
		return object{"tag": "proxy", "protocol": "socks", "settings": object{"servers": []any{object{"address": "127.0.0.1", "port": 10829}}}}, nil
	}
	return nil, errors.New("unsupported mode")
}

func inbound() []any {
	return []any{
		object{"tag": "lan-in", "listen": "0.0.0.0", "port": 24810, "protocol": "dokodemo-door", "settings": object{"network": "tcp,udp", "followRedirect": true}, "streamSettings": object{"sockopt": object{"tproxy": "tproxy"}}, "sniffing": object{"enabled": true, "destOverride": []string{"http", "tls", "quic"}, "routeOnly": true}},
		object{"tag": "lan6-in", "listen": "::", "port": 24811, "protocol": "dokodemo-door", "settings": object{"network": "tcp,udp", "followRedirect": true}, "streamSettings": object{"sockopt": object{"tproxy": "tproxy"}}, "sniffing": object{"enabled": true, "destOverride": []string{"http", "tls", "quic"}, "routeOnly": true}},
		object{"tag": "dns-in", "listen": "0.0.0.0", "port": 24812, "protocol": "dokodemo-door", "settings": object{"address": "1.1.1.1", "port": 53, "network": "tcp,udp"}, "streamSettings": object{"sockopt": object{"tproxy": "tproxy"}}},
		object{"tag": "dns6-in", "listen": "::", "port": 24813, "protocol": "dokodemo-door", "settings": object{"address": "2606:4700:4700::1111", "port": 53, "network": "tcp,udp"}, "streamSettings": object{"sockopt": object{"tproxy": "tproxy"}}},
		object{"tag": "socks-in", "listen": "127.0.0.1", "port": 24800, "protocol": "socks", "settings": object{"auth": "noauth", "udp": true}},
		object{"tag": "http-in", "listen": "127.0.0.1", "port": 24801, "protocol": "http", "settings": object{}},
	}
}

func domainRule(domains []string, tag string) object {
	clean := []string{}
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d != "" {
			if !strings.Contains(d, ":") {
				d = "domain:" + d
			}
			clean = append(clean, d)
		}
	}
	return object{"type": "field", "domain": clean, "outboundTag": tag}
}

func buildConfig(s Settings) (object, error) {
	var outbounds []any
	if s.Mode == "custom" {
		var cfg object
		if err := json.Unmarshal(s.Custom, &cfg); err != nil {
			return nil, err
		}
		ob, ok := cfg["outbounds"].([]any)
		if !ok || len(ob) == 0 {
			return nil, errors.New("Xray config must contain outbounds")
		}
		outbounds = ob
		first, ok := outbounds[0].(map[string]any)
		if !ok {
			return nil, errors.New("invalid outbound")
		}
		first["tag"] = "proxy"
	} else {
		p, err := proxyOutbound(s)
		if err != nil {
			return nil, err
		}
		outbounds = []any{p}
	}
	if s.Chain != "" {
		chain, err := outboundFromURI(s.Chain)
		if err != nil {
			return nil, err
		}
		chain["tag"] = "chain"
		stream, ok := chain["streamSettings"].(map[string]any)
		if !ok {
			stream = object{}
			chain["streamSettings"] = stream
		}
		stream["sockopt"] = object{"dialerProxy": "proxy"}
		outbounds = append(outbounds, chain)
	}
	outbounds = append(outbounds, object{"tag": "direct", "protocol": "freedom"}, object{"tag": "block", "protocol": "blackhole"}, object{"tag": "dns-out", "protocol": "dns"})
	rules := []any{object{"type": "field", "inboundTag": []string{"dns-in", "dns6-in"}, "outboundTag": "dns-out"}}
	if len(s.BlockDomains) > 0 {
		rules = append(rules, domainRule(s.BlockDomains, "block"))
	}
	if len(s.BypassDomains) > 0 {
		rules = append(rules, domainRule(s.BypassDomains, "direct"))
	}
	rules = append(rules, object{"type": "field", "ip": []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "169.254.0.0/16", "::1/128", "fc00::/7", "fe80::/10"}, "outboundTag": "direct"})
	if s.Mode == "tor" {
		rules = append(rules, object{"type": "field", "network": "udp", "outboundTag": "block"})
	}
	final := "proxy"
	if s.Chain != "" {
		final = "chain"
	}
	rules = append(rules, object{"type": "field", "network": "tcp,udp", "outboundTag": final})
	ins := inbound()
	if !s.Transparent {
		ins = ins[4:]
	}
	return object{"log": object{"loglevel": s.LogLevel}, "dns": object{"servers": []string{s.DNS}, "queryStrategy": "UseIPv4"}, "inbounds": ins, "outbounds": outbounds, "routing": object{"domainStrategy": "IPIfNonMatch", "rules": rules}}, nil
}

func outboundFromURI(raw string) (object, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" {
		return nil, errors.New("invalid proxy URI")
	}
	p := 443
	if u.Port() != "" {
		p, err = strconv.Atoi(u.Port())
		if err != nil || p < 1 || p > 65535 {
			return nil, errors.New("invalid proxy port")
		}
	}
	q := u.Query()
	protocol := u.Scheme
	settings := object{}
	user := ""
	pass := ""
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	switch protocol {
	case "vless":
		settings = object{"vnext": []any{object{"address": u.Hostname(), "port": p, "users": []any{object{"id": user, "encryption": "none", "flow": q.Get("flow")}}}}}
	case "trojan":
		settings = object{"servers": []any{object{"address": u.Hostname(), "port": p, "password": user}}}
	case "socks", "socks5", "http", "https":
		server := object{"address": u.Hostname(), "port": p}
		if user != "" {
			server["users"] = []any{object{"user": user, "pass": pass}}
		}
		settings["servers"] = []any{server}
		if protocol == "socks5" {
			protocol = "socks"
		}
		if protocol == "https" {
			protocol = "http"
			q.Set("security", "tls")
		}
	default:
		return nil, errors.New("supported URIs: vless, trojan, socks5, http, https; use JSON for other protocols")
	}
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	stream := object{"network": network}
	security := q.Get("security")
	if security == "" && protocol == "trojan" {
		security = "tls"
	}
	if security != "" && security != "none" {
		stream["security"] = security
		sni := q.Get("sni")
		if sni == "" {
			sni = u.Hostname()
		}
		tls := object{"serverName": sni, "fingerprint": "chrome"}
		if q.Get("fp") != "" {
			tls["fingerprint"] = q.Get("fp")
		}
		if q.Get("alpn") != "" {
			tls["alpn"] = strings.Split(q.Get("alpn"), ",")
		}
		if security == "reality" {
			tls["publicKey"] = q.Get("pbk")
			tls["shortId"] = q.Get("sid")
			stream["realitySettings"] = tls
		} else {
			stream["tlsSettings"] = tls
		}
	}
	switch network {
	case "ws":
		stream["wsSettings"] = object{"host": q.Get("host"), "path": q.Get("path")}
	case "grpc":
		stream["grpcSettings"] = object{"serviceName": q.Get("serviceName")}
	case "xhttp":
		stream["xhttpSettings"] = object{"host": q.Get("host"), "path": q.Get("path"), "mode": "auto"}
	case "httpupgrade":
		stream["httpupgradeSettings"] = object{"host": q.Get("host"), "path": q.Get("path")}
	case "tcp":
	default:
		return nil, errors.New("unsupported transport")
	}
	return object{"tag": "proxy", "protocol": protocol, "settings": settings, "streamSettings": stream}, nil
}

func importConfig(s Settings, text string) (Settings, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "vless://") {
		u, err := url.Parse(text)
		if err != nil || u.User == nil {
			return s, errors.New("invalid VLESS URI")
		}
		q := u.Query()
		if q.Get("type") == "ws" && q.Get("security") == "tls" {
			s.Mode = "worker"
			s.Host = q.Get("host")
			if s.Host == "" {
				s.Host = q.Get("sni")
			}
			if s.Host == "" {
				s.Host = u.Hostname()
			}
			s.UUID = u.User.Username()
			s.Address = u.Hostname()
			s.Path = q.Get("path")
			if s.Path == "" {
				s.Path = "/"
			}
			s.Port = 443
			if u.Port() != "" {
				s.Port, err = strconv.Atoi(u.Port())
				if err != nil {
					return s, err
				}
			}
			return s, nil
		}
	}
	if strings.Contains(text, "://") && !strings.HasPrefix(text, "{") {
		out, err := outboundFromURI(text)
		if err != nil {
			return s, err
		}
		s.Custom, _ = json.Marshal(object{"outbounds": []any{out}})
		s.Mode = "custom"
		return s, nil
	}
	var data object
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return s, errors.New("paste a proxy URI, BPB settings, or Xray JSON")
	}
	if _, ok := data["outbounds"]; ok {
		s.Custom = json.RawMessage(text)
		s.Mode = "custom"
		return s, nil
	}
	if _, ok := data["mode"]; !ok {
		return s, errors.New("unknown JSON format")
	}
	on := s.Enabled
	if err := json.Unmarshal([]byte(text), &s); err != nil {
		return s, err
	}
	s.Enabled = on
	return s, nil
}

func networkStart(s Settings) error {
	if !s.Transparent {
		return networkStop()
	}
	if _, err := os.Stat(runtimeDir + "/network-owned"); err != nil {
		for _, family := range []string{"-4", "-6"} {
			b, routeErr := run(2*time.Second, "ip", family, "route", "show", "table", "120")
			if routeErr == nil && strings.TrimSpace(b) != "" {
				return errors.New("routing table 120 is already in use")
			}
		}
	}
	for _, table := range []string{"zirgozar", "passwall", "passwall2", "homeproxy", "nikki"} {
		if _, err := run(2*time.Second, "nft", "list", "table", "inet", table); err == nil {
			return fmt.Errorf("another transparent proxy is active (%s); stop it first or disable LAN routing", table)
		}
	}
	if _, err := net.InterfaceByName(s.LAN); err != nil {
		return err
	}
	if s.Mode == "tor" || s.Mode == "psiphon" {
		port := "9050"
		if s.Mode == "psiphon" {
			port = "10829"
		}
		c, e := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
		if e != nil {
			return fmt.Errorf("%s SOCKS service is not listening on %s", s.Mode, port)
		}
		c.Close()
	}
	_ = networkStop()
	if err := atomicWrite(runtimeDir+"/network-owned", []byte("pol\n"), 0600); err != nil {
		return err
	}
	for _, family := range []string{"-4", "-6"} {
		if _, err := run(4*time.Second, "ip", family, "route", "add", "local", "default", "dev", "lo", "table", "120"); err != nil {
			_ = networkStop()
			return err
		}
		if _, err := run(4*time.Second, "ip", family, "rule", "add", "priority", "120", "fwmark", "0x40000000/0x40000000", "lookup", "120"); err != nil {
			_ = networkStop()
			return err
		}
	}
	script := fmt.Sprintf(`table inet pol {
 chain prerouting {
  type filter hook prerouting priority -151; policy accept;
  iifname != %q return
  meta nfproto ipv4 meta l4proto { tcp, udp } th dport 53 tproxy ip to :24812 meta mark set meta mark | 0x40000000 accept
  meta nfproto ipv6 meta l4proto { tcp, udp } th dport 53 tproxy ip6 to :24813 meta mark set meta mark | 0x40000000 accept
  fib daddr type local return
  ip daddr { 0.0.0.0/8, 10.0.0.0/8, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16, 224.0.0.0/4, 240.0.0.0/4 } return
  ip6 daddr { ::1/128, fc00::/7, fe80::/10, ff00::/8 } return
  meta nfproto ipv4 meta l4proto { tcp, udp } tproxy ip to :24810 meta mark set meta mark | 0x40000000 accept
  meta nfproto ipv6 meta l4proto { tcp, udp } tproxy ip6 to :24811 meta mark set meta mark | 0x40000000 accept
 }
}
`, s.LAN)
	if err := inputCommand("nft", []string{"-f", "-"}, script); err != nil {
		_ = networkStop()
		return err
	}
	// fw4 owns the input chain; a dynamic rule also covers custom LAN interface names.
	if err := inputCommand("nft", []string{"-f", "-"}, fmt.Sprintf("insert rule inet fw4 input iifname %q meta mark & 0x40000000 == 0x40000000 accept comment \"pol-live\"\n", s.LAN)); err != nil {
		_ = networkStop()
		return err
	}
	event("Transparent routing started on " + s.LAN)
	return nil
}

func networkStop() error {
	if _, err := os.Stat(runtimeDir + "/network-owned"); err != nil {
		return nil
	}
	_, _ = run(3*time.Second, "nft", "delete", "table", "inet", "pol")
	// Remove only this application's rules and routes.
	for _, family := range []string{"-4", "-6"} {
		_, _ = run(3*time.Second, "ip", family, "rule", "del", "priority", "120", "fwmark", "0x40000000/0x40000000", "lookup", "120")
		_, _ = run(3*time.Second, "ip", family, "route", "del", "local", "default", "dev", "lo", "table", "120")
	}
	b, err := run(3*time.Second, "nft", "-a", "list", "chain", "inet", "fw4", "input")
	if err == nil {
		for _, line := range strings.Split(b, "\n") {
			if strings.Contains(line, `comment "pol-live"`) {
				parts := strings.Split(line, "# handle ")
				if len(parts) == 2 {
					handle := strings.TrimSpace(parts[1])
					if _, err := strconv.Atoi(handle); err == nil {
						_, _ = run(2*time.Second, "nft", "delete", "rule", "inet", "fw4", "input", "handle", handle)
					}
				}
			}
		}
	}
	_ = os.Remove(runtimeDir + "/network-owned")
	return nil
}
