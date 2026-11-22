package main

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"os/signal"
	"strings"
	"syscall"

	sshproxy "pol-openwrt/third_party/bpb/ssh"
)

func validateSSH(s Settings, complete bool) error {
	if s.Mode != "ssh" {
		return nil
	}
	scheme, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(s.Chain)), ":")
	if scheme == "wireguard" || scheme == "hysteria2" || scheme == "hy2" {
		return errors.New("SSH chains require a TCP-capable proxy")
	}
	c := s.SSH
	if c.Port < 1 || c.Port > 65535 || c.KeepAliveInterval < 0 || c.KeepAliveInterval > 3600 ||
		c.TCPKeepAliveInterval < 0 || c.TCPKeepAliveInterval > 3600 {
		return errors.New("invalid SSH port or keepalive interval")
	}
	if (net.ParseIP(c.Host) == nil && strings.ContainsAny(c.Host, " :/[]\r\n\t")) ||
		len(c.Host) > 253 || len(c.User) > 255 || len(c.PrivateKey) > 65536 {
		return errors.New("invalid SSH server, username or private key")
	}
	if c.HostKey != "" {
		hash, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(c.HostKey, "SHA256:"))
		if !strings.HasPrefix(c.HostKey, "SHA256:") || err != nil || len(hash) != 32 {
			return errors.New("SSH host fingerprint must be in SHA256 format")
		}
	}
	if complete {
		return sshproxy.Validate(c)
	}
	return nil
}

func runSSH(s Settings) error {
	if !s.Enabled || s.Mode != "ssh" {
		return errors.New("SSH is disabled")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	server, err := sshproxy.Start(s.SSH)
	if err != nil {
		return err
	}
	defer server.Stop()
	<-ctx.Done()
	return nil
}
