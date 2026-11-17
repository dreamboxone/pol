package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

type Job struct {
	Target   string          `json:"target,omitempty"`
	PID      int             `json:"pid,omitempty"`
	ID       string          `json:"id"`
	Method   string          `json:"method"`
	State    string          `json:"state"`
	Result   any             `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
	Started  int64           `json:"started"`
	Progress *UpdateProgress `json:"progress,omitempty"`
}

func readJob() Job {
	var j Job
	b, _ := os.ReadFile(runtimeDir + "/job.json")
	_ = json.Unmarshal(b, &j)
	var p UpdateProgress
	if b, err := os.ReadFile(runtimeDir + "/update-progress.json"); err == nil && json.Unmarshal(b, &p) == nil && p.ID == j.ID {
		j.Progress = &p
	}
	return j
}

func writeJob(j Job) error {
	b, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return atomicWrite(runtimeDir+"/job.json", b, 0600)
}

func startJob(method string, input object) (any, error) {
	old := readJob()
	if (old.State == "running" || old.State == "pending") && time.Now().Unix()-old.Started < 600 {
		return nil, errors.New("another background operation is in progress")
	}
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	target, _ := input["kind"].(string)
	if method == "update" {
		target, _ = input["component"].(string)
	}
	j := Job{ID: id, Method: method, Target: target, State: "pending", Started: time.Now().Unix()}
	if err = writeJob(j); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(executable, "job", method, id)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		pipe.Close()
		return nil, err
	}
	j.PID = cmd.Process.Pid
	if err = writeJob(j); err != nil {
		_ = syscall.Kill(-j.PID, syscall.SIGTERM)
		pipe.Close()
		return nil, err
	}
	// Secrets travel over a pipe and are never written to disk or command arguments.
	err = json.NewEncoder(pipe).Encode(input)
	pipe.Close()
	_ = cmd.Process.Release()
	if err != nil {
		return nil, err
	}
	return object{"job": id}, nil
}

func cancelJob() {
	j := readJob()
	if j.PID <= 1 || (j.State != "pending" && j.State != "running") {
		return
	}
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", j.PID))
	if err != nil || !strings.Contains(string(cmdline), "\x00job\x00"+j.Method+"\x00"+j.ID) {
		return
	}
	_ = syscall.Kill(-j.PID, syscall.SIGTERM)
	j.State = "failed"
	j.Error = "Cancelled by the main switch"
	_ = writeJob(j)
}

func executeJob(method, id string) {
	if !contains([]string{"deploy", "register", "update", "scan", "geoUpdate", "check"}, method) || !regexp.MustCompile(`^[a-f0-9-]{36}$`).MatchString(id) {
		os.Exit(1)
	}
	input := object{}
	if json.NewDecoder(io.LimitReader(os.Stdin, 2<<20)).Decode(&input) != nil {
		os.Exit(1)
	}
	f, err := os.OpenFile("/var/lock/pol.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		os.Exit(1)
	}
	defer f.Close()
	if syscall.Flock(int(f.Fd()), syscall.LOCK_EX) != nil {
		os.Exit(1)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	j := readJob()
	if j.ID != id {
		os.Exit(1)
	}
	j.State = "running"
	_ = writeJob(j)
	j.Result, err = rpc(method, input)
	if err != nil {
		j.State = "failed"
		j.Error = err.Error()
		event(method + " failed: " + j.Error)
	} else {
		j.State = "done"
	}
	_ = writeJob(j)
}

type UpdateProgress struct {
	ID       string `json:"id"`
	Phase    string `json:"phase"`
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
}

func reportUpdateProgress(phase string, received, total int64) {
	j := readJob()
	if (j.Method != "update" && j.Method != "geoUpdate" && j.Method != "scan") || j.State != "running" {
		return
	}
	p := UpdateProgress{ID: j.ID, Phase: phase, Received: received, Total: total}
	b, _ := json.Marshal(p)
	_ = atomicWrite(runtimeDir+"/update-progress.json", b, 0600)
}

type downloadProgress struct {
	received, total int64
	last            time.Time
}

func (p *downloadProgress) Write(b []byte) (int, error) {
	p.received += int64(len(b))
	if time.Since(p.last) >= 500*time.Millisecond {
		reportUpdateProgress("downloading", p.received, p.total)
		p.last = time.Now()
	}
	return len(b), nil
}
