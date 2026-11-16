package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type Asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}
type Release struct {
	Tag        string  `json:"tag_name"`
	Assets     []Asset `json:"assets"`
	Prerelease bool    `json:"prerelease"`
}

func getRelease(repo, tag string) (Release, error) {
	var release Release
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`).MatchString(repo) {
		return release, errors.New("set the release repository first")
	}
	path := "latest"
	if tag != "" {
		if !regexp.MustCompile(`^[A-Za-z0-9_.-]{1,80}$`).MatchString(tag) {
			return release, errors.New("invalid release tag")
		}
		path = "tags/" + tag
	}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repo+"/releases/"+path, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Pol-OpenWrt/"+version)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return release, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return release, fmt.Errorf("release lookup: HTTP %d", resp.StatusCode)
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&release)
	return release, err
}

func installedVersion(component string) string {
	if component == "app" {
		return version
	}
	fields := strings.Fields(cachedCoreVersion())
	if len(fields) > 1 && fields[0] == "Xray" {
		return fields[1]
	}
	return ""
}

func versionOrder(a, b string) (int, bool) {
	pattern := regexp.MustCompile(`^[vV]?([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)
	x, y := pattern.FindStringSubmatch(a), pattern.FindStringSubmatch(b)
	if x == nil || y == nil {
		return 0, false
	}
	for i := 1; i <= 3; i++ {
		u, v := strings.TrimLeft(x[i], "0"), strings.TrimLeft(y[i], "0")
		if len(u) > len(v) {
			return 1, true
		}
		if len(u) < len(v) {
			return -1, true
		}
		if u > v {
			return 1, true
		}
		if u < v {
			return -1, true
		}
	}
	if x[4] == y[4] {
		return 0, true
	}
	if x[4] == "" {
		return 1, true
	}
	if y[4] == "" {
		return -1, true
	}
	u, v := strings.Split(x[4], "."), strings.Split(y[4], ".")
	numeric := regexp.MustCompile(`^[0-9]+$`)
	for i := 0; i < len(u) && i < len(v); i++ {
		if u[i] == v[i] {
			continue
		}
		un, vn := numeric.MatchString(u[i]), numeric.MatchString(v[i])
		if un && vn {
			aa, bb := strings.TrimLeft(u[i], "0"), strings.TrimLeft(v[i], "0")
			if len(aa) > len(bb) {
				return 1, true
			}
			if len(aa) < len(bb) {
				return -1, true
			}
			if aa > bb {
				return 1, true
			}
			if aa < bb {
				return -1, true
			}
			continue
		}
		if un && !vn {
			return -1, true
		}
		if !un && vn {
			return 1, true
		}
		if u[i] > v[i] {
			return 1, true
		}
		return -1, true
	}
	if len(u) > len(v) {
		return 1, true
	}
	if len(u) < len(v) {
		return -1, true
	}
	return 0, true
}

var lookupRelease = getRelease

func releases(s Settings, component string) (any, error) {
	if component != "" && component != "app" && component != "core" {
		return nil, errors.New("invalid update component")
	}
	out := object{}
	for _, kind := range []string{"core", "app"} {
		if component != "" && component != kind {
			continue
		}
		repo := "dreamboxone/pol"
		if kind == "core" {
			repo = "XTLS/Xray-core"
		}
		current := installedVersion(kind)
		out[kind+"Current"] = current
		out[kind+"Repository"] = repo
		rel, err := lookupRelease(repo, "")
		if err != nil {
			out[kind+"Error"] = err.Error()
			continue
		}
		out[kind] = rel.Tag
		order, known := versionOrder(rel.Tag, current)
		out[kind+"Comparable"] = known
		out[kind+"UpToDate"] = known && order <= 0
		out[kind+"Newer"] = current == "" || known && order > 0
	}
	return out, nil
}

func download(asset Asset, path string, max int64) error {
	if !strings.HasPrefix(asset.URL, "https://github.com/") {
		return errors.New("release asset must be hosted on GitHub")
	}
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Get(asset.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	h := sha256.New()
	progress := &downloadProgress{total: resp.ContentLength}
	reportUpdateProgress("downloading", 0, resp.ContentLength)
	n, err := io.Copy(io.MultiWriter(f, h, progress), io.LimitReader(resp.Body, max+1))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n > max {
		return errors.New("download exceeds size limit")
	}
	reportUpdateProgress("verifying", n, resp.ContentLength)
	digest := strings.TrimPrefix(asset.Digest, "sha256:")
	if len(digest) != 64 || !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), digest) {
		return errors.New("release asset SHA-256 verification failed or digest is missing")
	}
	return nil
}

func coreAsset() (string, error) {
	target, ok := architectures()[packageArch()]
	if !ok {
		return "", errors.New("unsupported core architecture")
	}
	return target.Asset, nil
}

func update(s Settings, input object) (any, error) {
	component, _ := input["component"].(string)
	tag, _ := input["tag"].(string)
	if component != "core" && component != "app" {
		return nil, errors.New("invalid update component")
	}
	repo := "dreamboxone/pol"
	name := ""
	if component == "core" {
		repo = "XTLS/Xray-core"
		var err error
		name, err = coreAsset()
		if err != nil {
			return nil, err
		}
	}
	rel, err := getRelease(repo, tag)
	if err != nil {
		return nil, err
	}
	current := installedVersion(component)
	if current != "" {
		order, known := versionOrder(rel.Tag, current)
		if !known {
			return nil, errors.New("cannot compare installed and release versions")
		}
		if order <= 0 {
			return object{"ok": true, "upToDate": true, "version": current}, nil
		}
	}
	manager := "opkg"
	ext := ".ipk"
	if _, e := os.Stat("/usr/bin/apk"); e == nil {
		manager = "apk"
		ext = ".apk"
	}
	var asset Asset
	for _, a := range rel.Assets {
		if (component == "core" && a.Name == name) || (component == "app" && strings.HasPrefix(a.Name, "pol_") && strings.Contains(a.Name, "_"+packageArch()+".") && strings.HasSuffix(a.Name, ext)) {
			asset = a
			break
		}
	}
	if asset.URL == "" {
		return nil, errors.New("no compatible asset in this release")
	}
	dir, err := os.MkdirTemp("/tmp", "pol-update-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "download"+ext)
	if component == "core" {
		path = filepath.Join(dir, "core.zip")
	}
	event("Downloading " + component + " " + rel.Tag)
	if err = download(asset, path, 100<<20); err != nil {
		return nil, err
	}
	reportUpdateProgress("installing", 0, 0)
	if component == "app" {
		// The detached installer survives rpcd's restart during the package post-install hook.
		if err = atomicWrite("/tmp/pol-update"+ext, mustRead(path), 0600); err != nil {
			return nil, err
		}
		script := "#!/bin/sh\nsleep 2\n" + manager
		if manager == "apk" {
			script += " add --allow-untrusted --force-reinstall /tmp/pol-update.apk"
		} else {
			script += " install /tmp/pol-update.ipk"
		}
		script += " > /tmp/pol-update-result 2>&1\nrc=$?\nlogger -t pol-app \"Application update completed with code $rc\"\nrm -f /tmp/pol-update" + ext + "\n"
		if err = atomicWrite("/tmp/pol-install-update", []byte(script), 0700); err != nil {
			return nil, err
		}
		installer := exec.Command("/bin/sh", "/tmp/pol-install-update")
		installer.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		err = installer.Start()
		if err == nil {
			err = installer.Process.Release()
		}
		return object{"ok": err == nil, "scheduled": true, "version": rel.Tag}, err
	}
	zipReader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zipReader.Close()
	var data []byte
	for _, f := range zipReader.File {
		if f.Name == architectures()[packageArch()].Binary {
			if f.UncompressedSize64 > 120<<20 {
				return nil, errors.New("core too large")
			}
			r, e := f.Open()
			if e != nil {
				return nil, e
			}
			data, e = io.ReadAll(io.LimitReader(r, 120<<20))
			r.Close()
			if e != nil {
				return nil, e
			}
			break
		}
	}
	if len(data) < 4 || string(data[:4]) != "\x7fELF" {
		return nil, errors.New("release has no Xray ELF binary")
	}
	next := corePath + ".next"
	if err = atomicWrite(next, data, 0755); err != nil {
		return nil, err
	}
	defer os.Remove(next)
	if _, err = run(8*time.Second, next, "version"); err != nil {
		return nil, err
	}
	if s.Enabled {
		if _, err = run(15*time.Second, next, "run", "-test", "-config", runtimeDir+"/xray.json"); err != nil {
			return nil, err
		}
	}
	if err = backupFile(corePath, corePath+".previous"); err != nil {
		return nil, err
	}
	if err = os.Rename(next, corePath); err != nil {
		_ = os.Rename(corePath+".previous", corePath)
		return nil, err
	}
	if s.Enabled {
		if err = activate(s); err != nil {
			_ = os.Rename(corePath+".previous", corePath)
			_ = activate(s)
			return nil, fmt.Errorf("core update rolled back: %w", err)
		}
	}
	event("Core updated independently to " + rel.Tag)
	return object{"ok": true, "version": rel.Tag}, nil
}

func mustRead(path string) []byte { b, _ := os.ReadFile(path); return b }
