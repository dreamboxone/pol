package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func csv(value string) []string {
	var out []string
	for _, v := range strings.Split(value, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
func decodeLoose(value string) (string, error) {
	for _, encoding := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		if b, e := encoding.DecodeString(value); e == nil {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("invalid base64 proxy credentials")
}
func extendedProxyURI(u *url.URL) (object, error) {
	q := u.Query()
	port := 443
	if u.Port() != "" {
		n, e := strconv.Atoi(u.Port())
		if e != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("invalid proxy port")
		}
		port = n
	}
	user := ""
	password := ""
	hasPassword := false
	if u.User != nil {
		user = u.User.Username()
		password, hasPassword = u.User.Password()
	}
	out := object{"tag": "proxy"}
	switch u.Scheme {
	case "ss", "shadowsocks":
		method := user
		if !hasPassword {
			creds, e := decodeLoose(user)
			if e != nil {
				return nil, e
			}
			var ok bool
			method, password, ok = strings.Cut(creds, ":")
			if !ok {
				return nil, fmt.Errorf("invalid Shadowsocks credentials")
			}
		}
		if method == "" || password == "" {
			return nil, fmt.Errorf("invalid Shadowsocks credentials")
		}
		if q.Get("plugin") != "" {
			return nil, fmt.Errorf("Shadowsocks plugin links require an exported Xray JSON configuration")
		}
		out["protocol"] = "shadowsocks"
		out["settings"] = object{"servers": []any{object{"address": u.Hostname(), "port": port, "method": method, "password": password}}}
	case "wireguard":
		if user == "" || q.Get("publickey") == "" {
			return nil, fmt.Errorf("WireGuard link is missing its keys")
		}
		peer := object{"endpoint": net.JoinHostPort(u.Hostname(), strconv.Itoa(port)), "publicKey": q.Get("publickey")}
		if v := q.Get("presharedkey"); v != "" {
			peer["preSharedKey"] = v
		}
		if v := q.Get("keepalive"); v != "" {
			n, e := strconv.Atoi(v)
			if e != nil || n < 0 {
				return nil, fmt.Errorf("invalid WireGuard keepalive")
			}
			peer["keepAlive"] = n
		}
		if v := csv(q.Get("allowedips")); len(v) > 0 {
			peer["allowedIPs"] = v
		}
		cfg := object{"secretKey": user, "address": csv(q.Get("address")), "peers": []any{peer}}
		if v := q.Get("mtu"); v != "" {
			n, e := strconv.Atoi(v)
			if e != nil || n < 576 || n > 65535 {
				return nil, fmt.Errorf("invalid WireGuard MTU")
			}
			cfg["mtu"] = n
		}
		if values := csv(q.Get("reserved")); len(values) > 0 {
			if len(values) != 3 {
				return nil, fmt.Errorf("WireGuard reserved must have three bytes")
			}
			var bytes []int
			for _, v := range values {
				n, e := strconv.Atoi(v)
				if e != nil || n < 0 || n > 255 {
					return nil, fmt.Errorf("invalid WireGuard reserved byte")
				}
				bytes = append(bytes, n)
			}
			cfg["reserved"] = bytes
		}
		if values := csv(q.Get("dns")); len(values) > 0 {
			for _, v := range values {
				if net.ParseIP(v) == nil {
					return nil, fmt.Errorf("WireGuard DNS must contain IPs")
				}
			}
			cfg["remoteDNS"] = values
		}
		out["protocol"] = "wireguard"
		out["settings"] = cfg
	case "hysteria2", "hy2":
		if user == "" {
			return nil, fmt.Errorf("Hysteria2 authentication is missing")
		}
		sni := q.Get("sni")
		if sni == "" {
			sni = u.Hostname()
		}
		tls := object{"serverName": sni}
		if pin := q.Get("pinSHA256"); pin != "" {
			tls["pinnedPeerCertSha256"] = pin
		}
		if hasPassword {
			user += ":" + password
		}
		stream := object{"network": "hysteria", "security": "tls", "tlsSettings": tls, "hysteriaSettings": object{"version": 2, "auth": user}}
		var masks []any
		if q.Get("obfs") == "salamander" {
			masks = append(masks, object{"type": "salamander", "settings": object{"password": q.Get("obfs-password")}})
		}
		if ports := q.Get("mport"); ports != "" {
			masks = append(masks, object{"type": "udphop", "settings": object{"mode": "intervalRemote,intervalLocal", "remotePorts": ports}})
		}
		if len(masks) > 0 {
			stream["finalmask"] = object{"udp": masks}
		}
		out["protocol"] = "hysteria"
		out["settings"] = object{"version": 2, "address": u.Hostname(), "port": port}
		out["streamSettings"] = stream
	}
	return out, nil
}

func vmessURI(raw string) (object, error) {
	text, err := decodeLoose(strings.TrimPrefix(raw, "vmess://"))
	if err != nil {
		return nil, err
	}
	var v object
	if err = json.Unmarshal([]byte(text), &v); err != nil {
		return nil, fmt.Errorf("invalid VMess JSON")
	}
	str := func(key string) string {
		value := v[key]
		if value == nil {
			return ""
		}
		return fmt.Sprint(value)
	}
	port, err := strconv.Atoi(str("port"))
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid VMess port")
	}
	if str("add") == "" || str("id") == "" {
		return nil, fmt.Errorf("VMess server and UUID are required")
	}
	q := url.Values{}
	q.Set("type", str("net"))
	q.Set("host", str("host"))
	q.Set("path", str("path"))
	q.Set("sni", str("sni"))
	q.Set("alpn", str("alpn"))
	q.Set("fp", str("fp"))
	if str("tls") == "tls" {
		q.Set("security", "tls")
	}
	transport, err := outboundFromURI("vless://" + url.QueryEscape(str("id")) + "@" + net.JoinHostPort(str("add"), strconv.Itoa(port)) + "?" + q.Encode())
	if err != nil {
		return nil, err
	}
	security := str("scy")
	if security == "" {
		security = "auto"
	}
	aid := 0
	if str("aid") != "" {
		aid, err = strconv.Atoi(str("aid"))
		if err != nil || aid < 0 {
			return nil, fmt.Errorf("invalid VMess alterId")
		}
	}
	transport["protocol"] = "vmess"
	transport["settings"] = object{"vnext": []any{object{"address": str("add"), "port": port, "users": []any{object{"id": str("id"), "alterId": aid, "security": security}}}}}
	return transport, nil
}
