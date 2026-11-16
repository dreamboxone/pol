package main

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestCancellationProcessHelper(t *testing.T) {
	if os.Getenv("POL_CANCEL_TEST") != "1" {
		return
	}
	for {
		time.Sleep(time.Hour)
	}
}

func TestMainSwitchCancelsOnlyItsOwnJob(t *testing.T) {
	old := runtimeDir
	runtimeDir = t.TempDir()
	defer func() { runtimeDir = old }()
	id := "12345678-1234-1234-1234-123456789abc"
	cmd := exec.Command(os.Args[0], "-test.run=TestCancellationProcessHelper", "job", "scan", id)
	cmd.Env = append(os.Environ(), "POL_CANCEL_TEST=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	job := Job{ID: "other", Method: "scan", PID: cmd.Process.Pid, State: "running"}
	if err := writeJob(job); err != nil {
		t.Fatal(err)
	}
	cancelJob()
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal("cancelled a nonmatching process")
	}
	job.ID = id
	_ = writeJob(job)
	cancelJob()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("job survived main-switch cancellation")
	}
	if readJob().State != "failed" {
		t.Fatal("job status did not record cancellation")
	}
}

func TestUpdateProgressBelongsToCurrentJob(t *testing.T) {
	old := runtimeDir
	runtimeDir = t.TempDir()
	defer func() { runtimeDir = old }()
	if err := writeJob(Job{ID: "one", Method: "update", State: "running"}); err != nil {
		t.Fatal(err)
	}
	p := &downloadProgress{total: 1000}
	if n, err := p.Write(make([]byte, 250)); n != 250 || err != nil {
		t.Fatal("progress writer failed")
	}
	j := readJob()
	if j.Progress == nil || j.Progress.Received != 250 || j.Progress.Total != 1000 || j.Progress.Phase != "downloading" {
		t.Fatalf("wrong progress: %+v", j.Progress)
	}
	reportUpdateProgress("verifying", 1000, 1000)
	if readJob().Progress.Phase != "verifying" {
		t.Fatal("verification phase missing")
	}
	if err := writeJob(Job{ID: "two", Method: "update", State: "pending"}); err != nil {
		t.Fatal(err)
	}
	if readJob().Progress != nil {
		t.Fatal("previous download leaked into new job")
	}
}

func TestGeoDownloadProgress(t *testing.T) {
	old := runtimeDir
	runtimeDir = t.TempDir()
	defer func() { runtimeDir = old }()
	if err := writeJob(Job{ID: "geo", Method: "geoUpdate", Target: "geosite", State: "running"}); err != nil {
		t.Fatal(err)
	}
	reportUpdateProgress("connecting", 0, 0)
	p := &downloadProgress{total: 4096}
	_, _ = p.Write(make([]byte, 2048))
	j := readJob()
	if j.Target != "geosite" || j.Progress == nil || j.Progress.Received != 2048 || j.Progress.Total != 4096 {
		t.Fatalf("wrong Geo progress: %+v", j)
	}
	reportUpdateProgress("installing", 4096, 4096)
	if readJob().Progress.Phase != "installing" {
		t.Fatal("Geo install phase missing")
	}
}
