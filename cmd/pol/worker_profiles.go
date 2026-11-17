package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"time"
)

func rememberWorker(s *Settings) {
	if s.Host == "" || s.UUID == "" {
		return
	}
	p := WorkerProfile{Name: s.Host, Kind: "worker", Host: s.Host, UUID: s.UUID, Address: s.Address, Port: s.Port, Path: s.Path, WorkerSNI: s.WorkerSNI}
	for i, old := range s.WorkerProfiles {
		if old.Host == p.Host {
			p.Name = old.Name
			p.Kind = old.Kind
			p.AccountID = old.AccountID
			p.Project = old.Project
			s.WorkerProfiles[i] = p
			return
		}
	}
	s.WorkerProfiles = append(s.WorkerProfiles, p)
}

func deployPages(s Settings, input object) (any, error) {
	token, _ := input["token"].(string)
	account, _ := input["account"].(string)
	name, _ := input["name"].(string)
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{20,256}$`).MatchString(token) {
		return nil, fmt.Errorf("invalid API token")
	}
	if !regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(account) {
		return nil, fmt.Errorf("invalid Account ID")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9-]{2,40}$`).MatchString(name) {
		return nil, fmt.Errorf("invalid Pages project name")
	}
	uuid, err := newUUID()
	if err != nil {
		return nil, err
	}
	script, err := os.ReadFile("/usr/share/pol/worker.js")
	if err != nil {
		return nil, err
	}
	env, _ := json.Marshal(object{"v": 1, "id": uuid, "pips": s.WorkerProxyIPs, "fb": s.WorkerFallback})
	headers := map[string]string{"Authorization": "Bearer " + token}
	base := "https://api.cloudflare.com/client/v4/accounts/" + account + "/pages/projects"
	production := object{"compatibility_date": time.Now().UTC().Format("2006-01-02"), "env_vars": object{"ENV": object{"type": "plain_text", "value": base64.StdEncoding.EncodeToString(env)}}}
	project, err := requestJSON("POST", base, headers, object{"name": name, "production_branch": "main", "deployment_configs": object{"production": production}})
	if err != nil {
		return nil, err
	}
	host := strAt(mapAt(project, "result"), "subdomain")
	if host == "" {
		host = name + ".pages.dev"
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("manifest", "{}")
	_ = writer.WriteField("branch", "main")
	part, err := writer.CreateFormFile("_worker.js", "_worker.js")
	if err != nil {
		return nil, err
	}
	if _, err = part.Write(script); err != nil {
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", base+"/"+name+"/deployments", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", headers["Authorization"])
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Pages project created but upload failed: %w", err)
	}
	defer resp.Body.Close()
	var result object
	if err = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 || result["success"] == false {
		return nil, fmt.Errorf("Pages project created but upload failed: HTTP %d", resp.StatusCode)
	}
	s.Mode = "worker"
	s.Host = host
	s.UUID = uuid
	s.Address = ""
	s.Port = 443
	s.Path = "/?ed=2560"
	rememberWorker(&s)
	for i := range s.WorkerProfiles {
		if s.WorkerProfiles[i].Host == host {
			s.WorkerProfiles[i].Kind = "pages"
			s.WorkerProfiles[i].AccountID = account
			s.WorkerProfiles[i].Project = name
		}
	}
	if err = change(s); err != nil {
		return nil, fmt.Errorf("Pages created at %s, but local settings failed: %w", host, err)
	}
	return object{"ok": true, "host": host}, nil
}

func deleteDeployment(s Settings, input object) (any, error) {
	token, _ := input["token"].(string)
	account, _ := input["account"].(string)
	name, _ := input["name"].(string)
	kind, _ := input["kind"].(string)
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{20,256}$`).MatchString(token) {
		return nil, fmt.Errorf("invalid API token")
	}
	if !regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(account) || !regexp.MustCompile(`^[a-z][a-z0-9-]{2,40}$`).MatchString(name) {
		return nil, fmt.Errorf("invalid deployment identity")
	}
	if kind != "worker" && kind != "pages" {
		return nil, fmt.Errorf("invalid deployment type")
	}
	matched := false
	for _, p := range s.WorkerProfiles {
		if p.AccountID == account && p.Project == name && p.Kind == kind {
			matched = true
			if s.Enabled && s.Mode == "worker" && p.Host == s.Host {
				return nil, fmt.Errorf("turn off the Worker connection first")
			}
		}
	}
	if !matched {
		return nil, fmt.Errorf("select a saved deployment before deleting it")
	}
	base := "https://api.cloudflare.com/client/v4/accounts/" + account
	if kind == "pages" {
		base += "/pages/projects/" + name
	} else {
		base += "/workers/scripts/" + name
	}
	if _, err := requestJSON("DELETE", base, map[string]string{"Authorization": "Bearer " + token}, nil); err != nil {
		return nil, err
	}
	var keep []WorkerProfile
	for _, p := range s.WorkerProfiles {
		if p.AccountID == account && p.Project == name && p.Kind == kind {
			if p.Host == s.Host {
				s.Host = ""
				s.UUID = ""
				s.Address = ""
				s.WorkerSNI = ""
			}
		} else {
			keep = append(keep, p)
		}
	}
	s.WorkerProfiles = keep
	if err := change(s); err != nil {
		return nil, fmt.Errorf("remote deployment deleted but local settings failed: %w", err)
	}
	return object{"ok": true}, nil
}
