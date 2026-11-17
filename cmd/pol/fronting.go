package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

const frontingCert = "/etc/pol/ca/fronting.crt"
const frontingKey = "/etc/pol/ca/fronting.key"

func ensureFrontingCA() error {
	if _, err := tls.LoadX509KeyPair(frontingCert, frontingKey); err == nil {
		return nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	cert := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Pol local fronting CA"}, NotBefore: time.Now().Add(-24 * time.Hour), NotAfter: time.Now().AddDate(10, 0, 0), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		return err
	}
	private, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	if err = atomicWrite(frontingKey, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: private}), 0600); err != nil {
		return err
	}
	return atomicWrite(frontingCert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600)
}
func buildDomainFronting(s Settings) (object, error) {
	b, err := os.ReadFile("/usr/share/pol/domain-fronting.json")
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(string(b), "bpb-cert.crt", frontingCert)
	text = strings.ReplaceAll(text, "bpb-cert.key", frontingKey)
	text = strings.ReplaceAll(text, "127.0.0.1:11666", "127.0.0.1:24866")
	text = strings.ReplaceAll(text, "127.0.0.1:11777", "127.0.0.1:24877")
	var cfg object
	if err = json.Unmarshal([]byte(text), &cfg); err != nil {
		return nil, err
	}
	var ins []any
	for _, raw := range asArray(cfg["inbounds"]) {
		in := raw.(map[string]any)
		if in["tag"] == "mixed-in" {
			continue
		}
		in["listen"] = "127.0.0.1"
		if in["port"] == float64(11666) {
			in["port"] = 24866
		}
		if in["port"] == float64(11777) {
			in["port"] = 24877
		}
		ins = append(ins, in)
	}
	cfg["inbounds"] = ins
	routing := cfg["routing"].(map[string]any)
	var rules []any
	withExit := (s.ExitMode == "tor" || s.ExitMode == "psiphon" || s.Chain != "")
	for _, raw := range asArray(routing["rules"]) {
		r := raw.(map[string]any)
		if tag, _ := r["ruleTag"].(string); tag == "block-domain" || tag == "bypass-domain" || tag == "block-ip" || tag == "bypass-ip" {
			continue
		}
		if tags, ok := r["inboundTag"].([]any); ok && len(tags) == 0 {
			if withExit {
				r["inboundTag"] = []string{"fronting-upstream"}
			} else {
				delete(r, "inboundTag")
			}
		}
		rules = append(rules, r)
	}
	if withExit {
		var exit object
		if s.Chain != "" {
			exit, err = outboundFromURI(s.Chain)
			if err != nil {
				return nil, err
			}
			exit["tag"] = "fronting-exit"
			stream, _ := exit["streamSettings"].(map[string]any)
			if stream == nil {
				stream = object{}
				exit["streamSettings"] = stream
			}
			stream["sockopt"] = object{"dialerProxy": "fronting-dialer"}
			cfg["outbounds"] = append(asArray(cfg["outbounds"]), exit, object{"tag": "fronting-dialer", "protocol": "loopback", "settings": object{"inboundTag": "fronting-upstream"}})
		} else {
			port := 24805
			if s.ExitMode == "psiphon" {
				port = 24806
			}
			cfg["outbounds"] = append(asArray(cfg["outbounds"]), object{"tag": "fronting-exit", "protocol": "socks", "settings": object{"servers": []any{object{"address": "127.0.0.1", "port": port}}}})
		}
		rules = append([]any{object{"type": "field", "inboundTag": []string{"lan-in", "lan6-in", "socks-in", "http-in"}, "outboundTag": "fronting-exit"}}, rules...)
	}
	for _, r := range rules {
		r.(map[string]any)["type"] = "field"
	}
	routing["rules"] = rules
	cfg["log"] = object{"loglevel": s.LogLevel}
	data, _ := json.Marshal(cfg)
	copy := s
	copy.Mode = "custom"
	copy.Custom = data
	result, err := buildImportedConfig(copy)
	if err != nil {
		return nil, err
	}
	if withExit {
		result["inbounds"] = append(result["inbounds"].([]any), object{"tag": "fronting-upstream", "listen": "127.0.0.1", "port": 24828, "protocol": "socks", "settings": object{"auth": "noauth", "udp": true}})
	}
	for _, raw := range result["inbounds"].([]any) {
		in := raw.(map[string]any)
		if sniff, ok := in["sniffing"].(map[string]any); ok {
			sniff["destOverride"] = []string{"http", "tls", "fakedns"}
		}
	}
	if len(asArray(result["outbounds"])) == 0 {
		return nil, fmt.Errorf("fronting template has no outbounds")
	}
	return result, nil
}
