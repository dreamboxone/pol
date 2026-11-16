package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"regexp"
	"strings"
	"time"
)

var geoIPReference = regexp.MustCompile(`^(geoip:!?[A-Za-z0-9_-]+|ext:[A-Za-z0-9_-][A-Za-z0-9_.-]*\.dat:[A-Za-z0-9_!@-]+)$`)

func validDirectIP(value string) bool {
	if _, err := netip.ParseAddr(value); err == nil {
		return true
	}
	if _, err := netip.ParsePrefix(value); err == nil {
		return true
	}
	return len(value) <= 256 && !strings.Contains(value, "..") && geoIPReference.MatchString(value)
}

// Preserve pre-existing exceptions; remove only entries previously added by Pol.
func syncRebind(domains []string) error {
	for _, domain := range domains {
		if !validGeoDomain(domain) {
			return errors.New("invalid rebind exception domain")
		}
	}
	current, err := run(3*time.Second, "uci", "-q", "get", "dhcp.@dnsmasq[0].rebind_domain")
	if err != nil {
		if _, e := run(3*time.Second, "uci", "-q", "get", "dhcp.@dnsmasq[0]"); e != nil {
			return e
		}
	}
	have := strings.Fields(current)
	var owned []string
	b, _ := os.ReadFile("/etc/pol/rebind-owned.json")
	_ = json.Unmarshal(b, &owned)
	var keep []string
	changed := false
	for _, domain := range owned {
		if contains(domains, domain) {
			keep = append(keep, domain)
		} else if contains(have, domain) {
			if _, err = run(3*time.Second, "uci", "-q", "del_list", "dhcp.@dnsmasq[0].rebind_domain="+domain); err != nil {
				return err
			}
			changed = true
		}
	}
	for _, domain := range domains {
		if !contains(have, domain) && !contains(keep, domain) {
			if _, err = run(3*time.Second, "uci", "-q", "add_list", "dhcp.@dnsmasq[0].rebind_domain="+domain); err != nil {
				return err
			}
			keep = append(keep, domain)
			changed = true
		}
	}
	if changed {
		if _, err = run(5*time.Second, "uci", "commit", "dhcp"); err != nil {
			return err
		}
	}
	b, _ = json.Marshal(keep)
	if err = atomicWrite("/etc/pol/rebind-owned.json", b, 0600); err != nil {
		return err
	}
	if changed {
		if _, err = run(20*time.Second, "/etc/init.d/dnsmasq", "restart"); err != nil {
			return fmt.Errorf("dnsmasq restart: %w", err)
		}
	}
	return nil
}

func literalDirectRules(values []string) string {
	var v4, v6 []string
	for _, value := range values {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			addr, e := netip.ParseAddr(value)
			if e != nil {
				continue
			}
			prefix = netip.PrefixFrom(addr, addr.BitLen())
		}
		if prefix.Addr().Is4() {
			v4 = append(v4, prefix.Masked().String())
		} else {
			v6 = append(v6, prefix.Masked().String())
		}
	}
	var rules string
	if len(v4) > 0 {
		rules += "  ip daddr { " + strings.Join(v4, ", ") + " } counter return\n"
	}
	if len(v6) > 0 {
		rules += "  ip6 daddr { " + strings.Join(v6, ", ") + " } counter return\n"
	}
	return rules
}
