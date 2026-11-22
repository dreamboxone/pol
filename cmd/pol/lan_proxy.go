package main

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

func validateSharedProxy(s Settings) error {
	if !s.LANSharing {
		return nil
	}
	if strings.TrimSpace(s.LANUsername) == "" || s.LANPassword == "" ||
		len(s.LANUsername) > 255 || len(s.LANPassword) > 255 || strings.ContainsAny(s.LANUsername+s.LANPassword, "\r\n\x00") {
		return errors.New("LAN proxy requires a username and password")
	}
	return nil
}

func sharedProxyInbounds(s Settings) ([]any, error) {
	if !s.LANSharing {
		return nil, nil
	}
	if err := validateSharedProxy(s); err != nil {
		return nil, err
	}
	link, err := net.InterfaceByName(s.LAN)
	if err != nil {
		return nil, fmt.Errorf("cannot find LAN interface %q", s.LAN)
	}
	addrs, err := link.Addrs()
	if err != nil {
		return nil, err
	}
	var host string
	for _, addr := range addrs {
		ip, _, _ := net.ParseCIDR(addr.String())
		if ip.To4() != nil && !ip.IsLoopback() {
			host = ip.String()
			break
		}
	}
	if host == "" {
		return nil, errors.New("LAN interface requires an IPv4 address for proxy sharing")
	}
	accounts := []any{object{"user": s.LANUsername, "pass": s.LANPassword}}
	return []any{
		object{"tag": "lan-socks-in", "listen": host, "port": 24802, "protocol": "socks", "settings": object{"auth": "password", "accounts": accounts, "udp": true, "ip": host}},
		object{"tag": "lan-http-in", "listen": host, "port": 24803, "protocol": "http", "settings": object{"accounts": accounts}},
	}, nil
}
