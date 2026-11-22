package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const cloudflareAPI = "https://api.cloudflare.com/client/v4"

func manageCloudflareDomain(s Settings, input object) (any, error) {
	token, account, name, kind := strAt(input, "token"), strAt(input, "account"), strAt(input, "name"), strAt(input, "kind")
	hostname := strings.ToLower(strings.TrimSpace(strAt(input, "hostname")))
	action := strAt(input, "action")
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{20,256}$`).MatchString(token) ||
		!regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(account) ||
		!regexp.MustCompile(`^[a-z][a-z0-9-]{2,40}$`).MatchString(name) || !validGeoDomain(hostname) ||
		!strings.Contains(hostname, ".") || !contains([]string{"attach", "detach"}, action) {
		return nil, errors.New("invalid Cloudflare domain request")
	}
	if action == "detach" && s.Enabled && s.Mode == "worker" && strings.EqualFold(s.WorkerDomain, hostname) {
		return nil, errors.New("switch away from the custom domain before detaching it")
	}
	headers := map[string]string{"Authorization": "Bearer " + token}
	for _, p := range s.WorkerProfiles {
		if strings.EqualFold(p.AccountID, account) && p.Project == name && p.Kind == kind {
			switch kind {
			case "worker":
				if err := setWorkerDomain(p, hostname, action, headers); err != nil {
					return nil, err
				}
			case "pages":
				if err := setPagesDomain(p, hostname, action, headers); err != nil {
					return nil, err
				}
			default:
				return nil, errors.New("invalid Cloudflare product")
			}
			return object{"ok": true, "hostname": hostname}, nil
		}
	}
	return nil, errors.New("select a saved deployment first")
}

func cloudflareZone(account, hostname string, headers map[string]string) (string, error) {
	for name := hostname; strings.Contains(name, "."); {
		query := url.Values{"name": {name}, "account.id": {account}}
		resp, err := requestJSON("GET", cloudflareAPI+"/zones?"+query.Encode(), headers, nil)
		if err != nil {
			return "", err
		}
		for _, raw := range asArray(resp["result"]) {
			zone, _ := raw.(object)
			if strAt(zone, "name") == name && regexp.MustCompile(`^[a-fA-F0-9]{32}$`).MatchString(strAt(zone, "id")) {
				return strAt(zone, "id"), nil
			}
		}
		_, name, _ = strings.Cut(name, ".")
	}
	return "", errors.New("no matching Cloudflare zone for this account")
}

func setWorkerDomain(p WorkerProfile, hostname, action string, headers map[string]string) error {
	base := cloudflareAPI + "/accounts/" + p.AccountID + "/workers/domains"
	query := url.Values{"hostname": {hostname}}
	resp, err := requestJSON("GET", base+"?"+query.Encode(), headers, nil)
	if err != nil {
		return err
	}
	for _, raw := range asArray(resp["result"]) {
		domain, _ := raw.(object)
		if strAt(domain, "hostname") != hostname {
			continue
		}
		if strAt(domain, "service") != p.Project {
			return errors.New("domain already belongs to another Worker")
		}
		if action == "attach" {
			return nil
		}
		id := strAt(domain, "id")
		if id == "" {
			return errors.New("missing Worker domain ID")
		}
		_, err = requestJSON("DELETE", base+"/"+url.PathEscape(id), headers, nil)
		return err
	}
	if action == "detach" {
		return nil
	}
	zone, err := cloudflareZone(p.AccountID, hostname, headers)
	if err != nil {
		return err
	}
	_, err = requestJSON("PUT", base, headers, object{"hostname": hostname, "service": p.Project, "zone_id": zone})
	return err
}

func setPagesDomain(p WorkerProfile, hostname, action string, headers map[string]string) error {
	zone, err := cloudflareZone(p.AccountID, hostname, headers)
	if err != nil {
		return err
	}
	base := cloudflareAPI + "/accounts/" + p.AccountID + "/pages/projects/" + p.Project + "/domains"
	records := cloudflareAPI + "/zones/" + zone + "/dns_records"
	query := url.Values{"name": {hostname}}
	found, err := requestJSON("GET", records+"?"+query.Encode(), headers, nil)
	if err != nil {
		return err
	}
	domains, err := requestJSON("GET", base, headers, nil)
	if err != nil {
		return err
	}
	attached := false
	for _, raw := range asArray(domains["result"]) {
		domain, _ := raw.(object)
		attached = attached || strAt(domain, "name") == hostname
	}
	comment := "Pol Pages " + p.Project
	var ownRecord string
	var alreadyPoints bool
	for _, raw := range asArray(found["result"]) {
		record, _ := raw.(object)
		if strAt(record, "name") != hostname {
			continue
		}
		switch kind := strAt(record, "type"); kind {
		case "A", "AAAA":
			if action == "attach" {
				return errors.New("custom domain already has a conflicting DNS record")
			}
		case "CNAME":
			matches := strings.EqualFold(strings.TrimSuffix(strAt(record, "content"), "."), p.Host)
			if !matches && action == "attach" {
				return errors.New("custom domain already has a conflicting DNS record")
			}
			if matches {
				alreadyPoints = true
				if strAt(record, "comment") == comment {
					ownRecord = strAt(record, "id")
				}
			}
		}
	}
	if action == "detach" {
		if attached {
			if _, err := requestJSON("DELETE", base+"/"+url.PathEscape(hostname), headers, nil); err != nil {
				return err
			}
		}
		if ownRecord != "" {
			_, err = requestJSON("DELETE", records+"/"+url.PathEscape(ownRecord), headers, nil)
			return err
		}
		return nil
	}
	if !attached {
		if _, err := requestJSON("POST", base, headers, object{"name": hostname}); err != nil {
			return err
		}
	}
	if alreadyPoints {
		return nil
	}
	_, err = requestJSON("POST", records, headers, object{"type": "CNAME", "name": hostname, "content": p.Host, "proxied": true, "ttl": 1, "comment": comment})
	if err != nil && !attached {
		if _, rollback := requestJSON("DELETE", base+"/"+url.PathEscape(hostname), headers, nil); rollback != nil {
			return fmt.Errorf("Pages DNS creation failed and domain rollback failed: %v; %v", err, rollback)
		}
	}
	return err
}
