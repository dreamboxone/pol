package main

import "testing"

func TestWorkerSnapshotKeepsDeploymentIdentity(t *testing.T) {
	s := defaults()
	s.Host = "example.pages.dev"
	s.UUID = "12345678-1234-1234-1234-123456789abc"
	s.WorkerProfiles = []WorkerProfile{{Host: s.Host, Name: "home", Kind: "pages", AccountID: "account", Project: "example"}}
	s.Address = "104.16.1.1"
	rememberWorker(&s)
	if len(s.WorkerProfiles) != 1 || s.WorkerProfiles[0].Kind != "pages" || s.WorkerProfiles[0].Address != s.Address {
		t.Fatal("snapshot lost its deployment identity")
	}
	s.Host = ""
	s.UUID = ""
	s.WorkerProfiles = nil
	rememberWorker(&s)
	if len(s.WorkerProfiles) != 0 {
		t.Fatal("deleted Worker restored unexpectedly")
	}
}
