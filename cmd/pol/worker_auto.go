package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"
)

const workerTokenPath = "/etc/pol/worker-update-tokens.json"
const workerStampPath = "/etc/pol/worker-update-stamps.json"

func workerUpdateAccount(kind, account string) string { return kind + "/" + strings.ToLower(account) }

func workerUpdateTokens() map[string]string {
	return readWorkerUpdateMap(workerTokenPath)
}

func readWorkerUpdateMap(path string) map[string]string {
	values := map[string]string{}
	if data, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(data, &values) != nil || values == nil {
			return map[string]string{}
		}
	}
	return values
}

func configureWorkerUpdates(input object) (any, error) {
	enabled, ok := input["enabled"].(bool)
	if !ok {
		return nil, errors.New("automatic Worker update preference is required")
	}
	kind := strAt(input, "kind")
	if !contains([]string{"worker", "pages"}, kind) {
		return nil, errors.New("invalid Cloudflare product")
	}
	token, account := strAt(input, "token"), strings.ToLower(strings.TrimSpace(strAt(input, "account")))
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(account) {
		return nil, errors.New("invalid Account ID")
	}
	if enabled {
		if _, err := signInCloudflare(object{"token": token, "account": account, "kind": kind}); err != nil {
			return nil, err
		}
	}
	values := workerUpdateTokens()
	if enabled {
		values[workerUpdateAccount(kind, account)] = strings.TrimSpace(token)
	} else {
		delete(values, workerUpdateAccount(kind, account))
	}
	data, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	if err := atomicWrite(workerTokenPath, data, 0600); err != nil {
		return nil, err
	}
	return object{"ok": true, "enabled": enabled}, nil
}

func autoUpdateWorkers() error {
	lock, err := os.OpenFile("/var/lock/pol-worker-update.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if len(workerUpdateTokens()) == 0 {
		return nil
	}
	settings, err := load()
	if err != nil {
		return err
	}
	script, err := os.ReadFile("/usr/share/pol/worker.js")
	if err != nil {
		return err
	}
	stamp := fmt.Sprintf("%x", sha256.Sum256(script))
	stamps := readWorkerUpdateMap(workerStampPath)
	for _, profile := range settings.WorkerProfiles {
		if !contains([]string{"worker", "pages"}, profile.Kind) {
			continue
		}
		account := strings.ToLower(profile.AccountID)
		token := workerUpdateTokens()[workerUpdateAccount(profile.Kind, account)]
		key := workerUpdateAccount(profile.Kind, account) + "/" + profile.Project
		if token == "" || stamps[key] == stamp {
			continue
		}
		if _, err := updateWorkerScript(settings, object{"token": token, "account": account, "name": profile.Project, "kind": profile.Kind}); err != nil {
			event("Automatic Worker update failed for " + profile.Project)
			continue
		}
		stamps[key] = stamp
		data, _ := json.Marshal(stamps)
		if err := atomicWrite(workerStampPath, data, 0600); err != nil {
			return err
		}
		event("Automatically updated Worker " + profile.Project)
	}
	return nil
}
