package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"regexp"
	"syscall"
	"time"
)

type Job struct {
	ID      string `json:"id"`
	Method  string `json:"method"`
	State   string `json:"state"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
	Started int64  `json:"started"`
}

func readJob() Job {
	var j Job
	b, _ := os.ReadFile(runtimeDir + "/job.json")
	_ = json.Unmarshal(b, &j)
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
	j := Job{ID: id, Method: method, State: "pending", Started: time.Now().Unix()}
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
	// Secrets travel over a pipe and are never written to disk or command arguments.
	err = json.NewEncoder(pipe).Encode(input)
	pipe.Close()
	_ = cmd.Process.Release()
	if err != nil {
		return nil, err
	}
	return object{"job": id}, nil
}

func executeJob(method, id string) {
	if !contains([]string{"deploy", "register", "update"}, method) || !regexp.MustCompile(`^[a-f0-9-]{36}$`).MatchString(id) {
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
