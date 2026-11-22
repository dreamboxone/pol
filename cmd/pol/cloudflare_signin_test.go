package main

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type cloudflareTestTransport func(*http.Request) (*http.Response, error)

func (f cloudflareTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCloudflareDeploymentOperations(t *testing.T) {
	if _, err := os.Stat("/usr/share/pol/worker.js"); err != nil {
		t.Skip("requires installed Worker script")
	}
	const account = "0123456789abcdef0123456789abcdef"
	const token = "fixture-token-value-123456"
	oldClient := httpClient
	t.Cleanup(func() { httpClient = oldClient })
	settings := defaults()
	settings.WorkerProfiles = []WorkerProfile{{AccountID: account, Project: "fixture", Kind: "worker"}, {AccountID: account, Project: "fixture", Kind: "pages"}}
	var writes []string
	httpClient = &http.Client{Transport: cloudflareTestTransport(func(req *http.Request) (*http.Response, error) {
		body := `{"success":true,"result":{}}`
		switch {
		case req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/pages/projects/fixture"):
			body = `{"success":true,"result":{"production_branch":"main"}}`
		case req.Method == "GET" && strings.HasSuffix(req.URL.Path, "/workers/domains"):
			body = `{"success":true,"result":[{"hostname":"vpn.example.com","service":"other"}]}`
		case req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/graphql"):
			body = `{"data":{"viewer":{"accounts":[{"total":[{"sum":{"requests":120}}],"worker":[{"sum":{"requests":30}}]}]}}}`
		default:
			writes = append(writes, req.Method+" "+req.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	input := object{"token": token, "account": account, "name": "fixture"}
	for _, kind := range []string{"worker", "pages"} {
		input["kind"] = kind
		if _, err := updateWorkerScript(settings, input); err != nil {
			t.Fatal(err)
		}
	}
	input["kind"], input["hostname"], input["action"] = "worker", "vpn.example.com", "attach"
	if _, err := manageCloudflareDomain(settings, input); err == nil {
		t.Fatal("overwriting another Worker's domain should be rejected")
	}
	usage, err := cloudflareUsage(input)
	if err != nil || usage.(object)["deployment"] != int64(30) {
		t.Fatalf("unexpected usage result: %v, %v", usage, err)
	}
	if len(writes) != 2 || !strings.HasSuffix(writes[0], "/workers/scripts/fixture/content") ||
		!strings.HasSuffix(writes[1], "/pages/projects/fixture/deployments") {
		t.Fatalf("unexpected deployment writes: %v", writes)
	}
}

func TestCloudflareSignInChecksAccountAccess(t *testing.T) {
	const account = "0123456789abcdef0123456789abcdef"
	const token = "fixture-token-value-123456"
	oldClient := httpClient
	t.Cleanup(func() { httpClient = oldClient })
	for _, test := range []struct {
		kind, path, body string
		status           int
	}{
		{"worker", "/workers/scripts", `{"success":true,"result":[]}`, 200},
		{"pages", "/pages/projects", `{"success":true,"result":[]}`, 200},
		{"worker", "/workers/scripts", `{"success":false}`, 403},
	} {
		httpClient = &http.Client{Transport: cloudflareTestTransport(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet || req.URL.Host != "api.cloudflare.com" ||
				req.URL.Path != "/client/v4/accounts/"+account+test.path ||
				req.Header.Get("Authorization") != "Bearer "+token {
				t.Fatal("unexpected Cloudflare verification request")
			}
			return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body))}, nil
		})}
		result, err := signInCloudflare(object{"account": account, "token": token, "kind": test.kind})
		if test.status != 200 {
			if err == nil {
				t.Fatal("account access was denied but sign-in succeeded")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		data := result.(object)
		if len(data) != 3 || data["account"] != account || data["kind"] != test.kind {
			t.Fatal("sign-in should return only non-secret account metadata")
		}
	}
}
