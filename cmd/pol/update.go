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
	"runtime"
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

func releases(s Settings) (any, error) {
	core, ce := getRelease("XTLS/Xray-core", "")
	out := object{"core": core.Tag, "app": "", "repository": s.ReleaseRepository}
	if ce != nil {
		out["coreError"] = ce.Error()
	}
	if s.ReleaseRepository != "" {
		app, e := getRelease(s.ReleaseRepository, "")
		if e != nil {
			out["appError"] = e.Error()
		} else {
			out["app"] = app.Tag
		}
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
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, max+1))
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
	digest := strings.TrimPrefix(asset.Digest, "sha256:")
	if len(digest) != 64 || !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), digest) {
		return errors.New("release asset SHA-256 verification failed or digest is missing")
	}
	return nil
}

func coreAsset() (string, error) {
	switch runtime.GOARCH {
	case "arm":
		return "Xray-linux-arm32-v7a.zip", nil
	case "arm64":
		return "Xray-linux-arm64-v8a.zip", nil
	case "amd64":
		return "Xray-linux-64.zip", nil
	case "386":
		return "Xray-linux-32.zip", nil
	case "mips":
		return "Xray-linux-mips32.zip", nil
	case "mipsle":
		return "Xray-linux-mips32le.zip", nil
	case "riscv64":
		return "Xray-linux-riscv64.zip", nil
	default:
		return "", errors.New("unsupported core architecture")
	}
}

func update(s Settings, input object) (any, error) {
	component, _ := input["component"].(string)
	tag, _ := input["tag"].(string)
	if component != "core" && component != "app" {
		return nil, errors.New("invalid update component")
	}
	repo := s.ReleaseRepository
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
		if f.Name == "xray" {
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
	if err = os.Rename(corePath, corePath+".previous"); err != nil {
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

func packageArch() string {
	if runtime.GOARCH == "arm" {
		return "arm_cortex-a7_neon-vfpv4"
	}
	if runtime.GOARCH == "amd64" {
		return "x86_64"
	}
	if runtime.GOARCH == "arm64" {
		return "aarch64_generic"
	}
	return runtime.GOARCH
}

func mustRead(path string) []byte { b, _ := os.ReadFile(path); return b }
