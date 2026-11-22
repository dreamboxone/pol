package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

func effectiveChain(s Settings) string {
	id, configured := s.ExitChains[s.Mode]
	if !configured {
		return s.Chain
	}
	for _, chain := range s.Chains {
		if chain.ID == id {
			link, _ := wireGuardChainLink(chain.Link)
			return link
		}
	}
	return ""
}

func validateChains(s Settings) error {
	if len(s.Chains) > 32 || len(s.ExitChains) > 8 {
		return errors.New("too many saved chains")
	}
	ids := make(map[string]bool, len(s.Chains))
	for _, chain := range s.Chains {
		if strings.TrimSpace(chain.ID) == "" || len(chain.ID) > 128 || strings.TrimSpace(chain.Name) == "" || len(chain.Name) > 128 || len(chain.Link) > 16384 || ids[chain.ID] {
			return errors.New("invalid or duplicate saved chain")
		}
		link, err := wireGuardChainLink(chain.Link)
		if err != nil {
			return err
		}
		if _, err := outboundFromURI(link); err != nil {
			return fmt.Errorf("invalid saved chain %q: %w", chain.Name, err)
		}
		ids[chain.ID] = true
	}
	for mode, id := range s.ExitChains {
		if !contains([]string{"worker", "warp", "masque", "serverless", "tor", "psiphon", "ssh"}, mode) || id != "" && !ids[id] {
			return errors.New("unknown saved chain or connection type")
		}
	}
	return nil
}

// wireGuardChainLink accepts a URI or the first peer of a WireGuard configuration file.
func wireGuardChainLink(raw string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "[interface]") {
		return raw, nil
	}
	iface, peer := map[string]string{}, map[string]string{}
	var section map[string]string
	peers := 0
	for _, line := range strings.Split(raw, "\n") {
		line, _, _ = strings.Cut(line, "#")
		line = strings.TrimSpace(line)
		switch strings.ToLower(line) {
		case "[interface]":
			section = iface
		case "[peer]":
			peers++
			if peers == 1 {
				section = peer
			} else {
				section = nil
			}
		default:
			key, value, present := strings.Cut(line, "=")
			if section != nil && present {
				section[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
			}
		}
	}
	host, port, err := net.SplitHostPort(peer["endpoint"])
	if err != nil || host == "" || iface["privatekey"] == "" || iface["address"] == "" || peer["publickey"] == "" {
		return "", errors.New("WireGuard file requires PrivateKey, Address, PublicKey and Endpoint")
	}
	query := url.Values{"publickey": {peer["publickey"]}, "address": {iface["address"]}}
	for key, value := range map[string]string{"mtu": iface["mtu"], "presharedkey": peer["presharedkey"], "keepalive": peer["persistentkeepalive"], "allowedips": peer["allowedips"]} {
		if value != "" {
			query.Set(key, value)
		}
	}
	var resolvers []string
	for _, value := range csv(iface["dns"]) {
		if net.ParseIP(value) != nil {
			resolvers = append(resolvers, value)
		}
	}
	if len(resolvers) > 0 {
		query.Set("dns", strings.Join(resolvers, ","))
	}
	link := url.URL{Scheme: "wireguard", Host: net.JoinHostPort(host, port), User: url.User(iface["privatekey"]), RawQuery: query.Encode()}
	return link.String(), nil
}
