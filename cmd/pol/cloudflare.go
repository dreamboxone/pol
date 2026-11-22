package main

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 35 * time.Second}

func signInCloudflare(input object) (any, error) {
	token := strings.TrimSpace(strAt(input, "token"))
	account := strings.TrimSpace(strAt(input, "account"))
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{20,256}$`).MatchString(token) {
		return nil, errors.New("invalid API token")
	}
	if !regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(account) {
		return nil, errors.New("invalid Account ID")
	}
	kind := strAt(input, "kind")
	var path string
	switch kind {
	case "worker":
		path = "/workers/scripts"
	case "pages":
		path = "/pages/projects?per_page=1"
	default:
		return nil, errors.New("invalid Cloudflare product")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	url := "https://api.cloudflare.com/client/v4/accounts/" + account + path
	result, err := requestJSONContext(ctx, http.MethodGet, url, map[string]string{"Authorization": "Bearer " + token}, nil)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("Cloudflare sign-in timed out")
		}
		return nil, errors.New("Cloudflare account access could not be verified")
	}
	if _, ok := result["result"].([]any); result["success"] != true || !ok {
		return nil, errors.New("Cloudflare account access could not be verified")
	}
	return object{"account": account, "kind": kind, "autoUpdate": workerUpdateTokens()[workerUpdateAccount(kind, account)] != ""}, nil
}

func cloudflareUsage(input object) (any, error) {
	token, account, name := strAt(input, "token"), strAt(input, "account"), strAt(input, "name")
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{20,256}$`).MatchString(token) ||
		!regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(account) ||
		!regexp.MustCompile(`^[a-z][a-z0-9-]{2,40}$`).MatchString(name) {
		return nil, errors.New("invalid deployment identity")
	}
	const query = `query GetUsage($account: String!, $script: String!, $start: String!, $end: String!) {
 viewer { accounts(filter: {accountTag: $account}) {
 total: workersInvocationsAdaptive(limit: 100, filter: {datetime_geq: $start, datetime_leq: $end}) { sum { requests } }
 worker: workersInvocationsAdaptive(limit: 100, filter: {scriptName: $script, datetime_geq: $start, datetime_leq: $end}) { sum { requests } }
 }}}`
	end := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	data, err := requestJSONContext(ctx, http.MethodPost, "https://api.cloudflare.com/client/v4/graphql", map[string]string{"Authorization": "Bearer " + token}, object{"query": query, "variables": object{"account": account, "script": name, "start": end.Add(-24 * time.Hour).Format(time.RFC3339), "end": end.Format(time.RFC3339)}})
	if err != nil || len(asArray(data["errors"])) > 0 {
		return nil, errors.New("Cloudflare usage could not be loaded")
	}
	accounts := asArray(mapAt(mapAt(data, "data"), "viewer")["accounts"])
	if len(accounts) == 0 {
		return nil, errors.New("Cloudflare usage could not be loaded")
	}
	metrics, _ := accounts[0].(object)
	sum := func(key string) int64 {
		var total int64
		for _, raw := range asArray(metrics[key]) {
			row, _ := raw.(object)
			count, _ := mapAt(row, "sum")["requests"].(float64)
			total += int64(count)
		}
		return total
	}
	return object{"total": sum("total"), "deployment": sum("worker")}, nil
}

func requestJSON(method, address string, headers map[string]string, body any) (object, error) {
	return requestJSONContext(context.Background(), method, address, headers, body)
}

func requestJSONContext(ctx context.Context, method, address string, headers map[string]string, body any) (object, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, address, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result object
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("HTTP %d: invalid API response", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API HTTP %d: %v", resp.StatusCode, result["errors"])
	}
	if result["success"] == false {
		return nil, fmt.Errorf("API: %v", result["errors"])
	}
	return result, nil
}

func mapAt(m object, key string) object {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return object{}
}
func strAt(m object, key string) string { s, _ := m[key].(string); return s }

func registerAccount(s Settings, kind string) (Settings, error) {
	if kind != "warp" && kind != "masque" {
		return s, errors.New("invalid account type")
	}
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return s, err
	}
	api := "https://api.cloudflareclient.com/v0a4005/reg"
	headers := map[string]string{"User-Agent": "WARP for Android", "CF-Client-Version": "a-6.35-4471"}
	if kind == "masque" {
		api = "https://api.cloudflareclient.com/v0a4471/reg"
	}
	body := object{"key": base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()), "install_id": "", "fcm_token": "", "tos": time.Now().UTC().Format(time.RFC3339), "model": "PC", "type": "Android", "locale": "en_US", "warp_enabled": true}
	if kind == "masque" {
		serial := make([]byte, 8)
		if _, err = rand.Read(serial); err != nil {
			return s, err
		}
		body["serial_number"] = fmt.Sprintf("%x", serial)
		body["os_version"] = ""
		body["key_type"] = "curve25519"
		body["tunnel_type"] = "wireguard"
	}
	reg, err := requestJSON("POST", api, headers, body)
	if err != nil {
		return s, err
	}
	if kind == "masque" {
		private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return s, err
		}
		pub, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
		if err != nil {
			return s, err
		}
		priv, err := x509.MarshalECPrivateKey(private)
		if err != nil {
			return s, err
		}
		id := strAt(reg, "id")
		token := strAt(reg, "token")
		if id == "" || token == "" {
			return s, errors.New("registration response is missing account credentials")
		}
		headers["Authorization"] = "Bearer " + token
		enrolled, e := requestJSON("PATCH", api+"/"+id, headers, object{"key": base64.StdEncoding.EncodeToString(pub), "key_type": "secp256r1", "tunnel_type": "masque"})
		if e != nil {
			return s, e
		}
		cfg := mapAt(enrolled, "config")
		iface := mapAt(mapAt(cfg, "interface"), "addresses")
		peers, _ := cfg["peers"].([]any)
		if len(peers) == 0 {
			return s, errors.New("MASQUE response has no peers")
		}
		peer, _ := peers[0].(map[string]any)
		public := strAt(peer, "public_key")
		if strings.Contains(public, "-----BEGIN") {
			parts := []string{}
			for _, line := range strings.Split(public, "\n") {
				if !strings.HasPrefix(line, "-----") {
					parts = append(parts, strings.TrimSpace(line))
				}
			}
			public = strings.Join(parts, "")
		}
		s.Masque = &MasqueAccount{PrivateKey: base64.StdEncoding.EncodeToString(priv), PublicKey: public, IPv4: strAt(iface, "v4"), IPv6: strAt(iface, "v6")}
		if s.Masque.PublicKey == "" || s.Masque.IPv4 == "" || s.Masque.IPv6 == "" {
			return s, errors.New("incomplete MASQUE response")
		}
		s.Endpoint = offeredEndpoint(peer, "162.159.198.1", 443)
		s.Mode = "masque"
		if s.SNI == "" || s.SNI == "engage.cloudflareclient.com" {
			s.SNI = "speed.cloudflare.com"
		}
	} else {
		cfg := mapAt(reg, "config")
		iface := mapAt(mapAt(cfg, "interface"), "addresses")
		peers, _ := cfg["peers"].([]any)
		if len(peers) == 0 {
			return s, errors.New("WARP response has no peers")
		}
		peer, _ := peers[0].(map[string]any)
		reserved := []int{0, 0, 0}
		decoded, _ := base64.StdEncoding.DecodeString(strAt(cfg, "client_id"))
		if len(decoded) == 3 {
			for i, b := range decoded {
				reserved[i] = int(b)
			}
		}
		ip := strAt(iface, "v6")
		if ip != "" && !strings.Contains(ip, "/") {
			ip += "/128"
		}
		s.Warp = &WarpAccount{PrivateKey: base64.StdEncoding.EncodeToString(key.Bytes()), PublicKey: strAt(peer, "public_key"), IPv6: ip, Reserved: reserved}
		if s.Warp.PublicKey == "" || ip == "" {
			return s, errors.New("incomplete WARP response")
		}
		s.Mode = "warp"
		s.Endpoint = offeredEndpoint(peer, "162.159.192.1", 2408)
	}
	event("Registered " + kind + " account")
	return s, nil
}

func offeredEndpoint(peer object, fallback string, port int) string {
	ep := mapAt(peer, "endpoint")
	host, _, err := net.SplitHostPort(strAt(ep, "v4"))
	if err != nil || host == "" {
		host = fallback
	}
	if ports, ok := ep["ports"].([]any); ok && len(ports) > 0 {
		if p, ok := ports[0].(float64); ok && p > 0 && p <= 65535 {
			port = int(p)
		}
	}
	return net.JoinHostPort(host, fmt.Sprint(port))
}

func newUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func deployWorker(s Settings, input object) (any, error) {
	token, _ := input["token"].(string)
	account, _ := input["account"].(string)
	name, _ := input["name"].(string)
	token = strings.TrimSpace(token)
	account = strings.TrimSpace(account)
	name = strings.TrimSpace(name)
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{20,256}$`).MatchString(token) {
		return nil, errors.New("invalid API token")
	}
	if !regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(account) {
		return nil, errors.New("invalid Account ID")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9-]{2,40}$`).MatchString(name) {
		return nil, errors.New("Worker name must be 3-41 lowercase letters, digits or hyphens")
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	base := "https://api.cloudflare.com/client/v4/accounts/" + account + "/workers"
	sub, err := requestJSON("GET", base+"/subdomain", headers, nil)
	if err != nil {
		return nil, err
	}
	subdomain := strAt(mapAt(sub, "result"), "subdomain")
	if subdomain == "" {
		return nil, errors.New("choose a workers.dev subdomain in Cloudflare Workers & Pages first")
	}
	// A new name avoids replacing unrelated deployments.
	_, err = requestJSON("GET", base+"/scripts/"+name+"/settings", headers, nil)
	if err == nil {
		return nil, errors.New("Worker name already exists; choose a new name")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		return nil, err
	}
	uuid, err := newUUID()
	if err != nil {
		return nil, err
	}
	env, _ := json.Marshal(object{"v": 1, "id": uuid, "pips": s.WorkerProxyIPs, "fb": s.WorkerFallback})
	meta, _ := json.Marshal(object{"main_module": "worker.js", "compatibility_date": "2026-09-01", "bindings": []any{object{"type": "plain_text", "name": "ENV", "text": base64.StdEncoding.EncodeToString(env)}}})
	if err = uploadWorkerScript(base+"/scripts/"+name, token, meta); err != nil {
		return nil, err
	}
	if _, err = requestJSON("POST", base+"/scripts/"+name+"/subdomain", headers, object{"enabled": true, "previews_enabled": false}); err != nil {
		return nil, err
	}
	s.Mode = "worker"
	s.WorkerDomain = ""
	s.Host = name + "." + subdomain + ".workers.dev"
	s.Address = ""
	s.UUID = uuid
	s.Port = 443
	s.Path = "/?ed=2560"
	rememberWorker(&s)
	for i := range s.WorkerProfiles {
		if s.WorkerProfiles[i].Host == s.Host {
			s.WorkerProfiles[i].AccountID = account
			s.WorkerProfiles[i].Project = name
			s.WorkerProfiles[i].Kind = "worker"
		}
	}
	if err = change(s); err != nil {
		return nil, fmt.Errorf("Worker created at %s, but local settings failed: %w", s.Host, err)
	}
	event("Worker deployed: " + s.Host)
	return object{"ok": true, "host": s.Host}, nil
}

func uploadWorkerScript(address, token string, meta []byte) error {
	script, err := os.ReadFile("/usr/share/pol/worker.js")
	if err != nil {
		return err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, p := range []struct {
		name, typ string
		data      []byte
	}{{"metadata", "application/json", meta}, {"worker.js", "application/javascript+module", script}} {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, p.name, p.name))
		h.Set("Content-Type", p.typ)
		part, e := writer.CreatePart(h)
		if e != nil {
			return e
		}
		if _, e = part.Write(p.data); e != nil {
			return e
		}
	}
	if err = writer.Close(); err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", address, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	var result object
	decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result)
	resp.Body.Close()
	if decodeErr != nil || resp.StatusCode >= 300 || result["success"] == false {
		return fmt.Errorf("Worker upload failed (HTTP %d): %v", resp.StatusCode, result["errors"])
	}
	return nil
}

func updateWorkerScript(s Settings, input object) (any, error) {
	token, account, name := strAt(input, "token"), strAt(input, "account"), strAt(input, "name")
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{20,256}$`).MatchString(token) ||
		!regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(account) ||
		!regexp.MustCompile(`^[a-z][a-z0-9-]{2,40}$`).MatchString(name) {
		return nil, errors.New("invalid Worker token, account or name")
	}
	kind := strAt(input, "kind")
	if kind == "" {
		kind = "worker"
	}
	for _, profile := range s.WorkerProfiles {
		if profile.Kind != kind || !strings.EqualFold(profile.AccountID, account) || profile.Project != name {
			continue
		}
		base := "https://api.cloudflare.com/client/v4/accounts/" + account
		if kind == "pages" {
			address := base + "/pages/projects/" + name
			project, err := requestJSON("GET", address, map[string]string{"Authorization": "Bearer " + token}, nil)
			if err != nil {
				return nil, err
			}
			branch := strAt(mapAt(project, "result"), "production_branch")
			if branch == "" {
				return nil, errors.New("Pages project has no production branch")
			}
			if err := uploadPagesScript(address, token, branch); err != nil {
				return nil, err
			}
			return object{"ok": true}, nil
		}
		if kind != "worker" {
			return nil, errors.New("unsupported deployment")
		}
		address := base + "/workers/scripts/" + name + "/content"
		if err := uploadWorkerScript(address, token, []byte(`{"main_module":"worker.js"}`)); err != nil {
			return nil, err
		}
		return object{"ok": true}, nil
	}
	return nil, errors.New("select a Worker deployed by Pol")
}
