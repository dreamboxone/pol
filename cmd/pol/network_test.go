package main

import (
	"bytes"
	"encoding/binary"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testDNSStub(t *testing.T, address string, ip []byte) chan bool {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	seen := make(chan bool, 10)
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, peer, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			if n < 12 {
				continue
			}
			end := 12
			for end < n && buffer[end] != 0 {
				end += int(buffer[end]) + 1
			}
			end += 5
			if end > n {
				continue
			}
			reply := append([]byte(nil), buffer[:end]...)
			reply[2], reply[3] = 0x81, 0x80
			binary.BigEndian.PutUint16(reply[10:12], 0)
			binary.BigEndian.PutUint16(reply[6:8], 1)
			reply = append(reply, 0xc0, 0x0c, 0, 1, 0, 1, 0, 0, 0, 60, 0, 4)
			reply = append(reply, ip...)
			_, _ = conn.WriteToUDP(reply, peer)
			select {
			case seen <- true:
			default:
			}
		}
	}()
	return seen
}

func testDNSQuery(name string) ([]byte, error) {
	query := []byte{0, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	for _, part := range strings.Split(name, ".") {
		query = append(query, byte(len(part)))
		query = append(query, part...)
	}
	query = append(query, 0, 0, 1, 0, 1)
	conn, err := net.Dial("udp", "127.0.0.1:24814")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	if _, err = conn.Write(query); err != nil {
		return nil, err
	}
	response := make([]byte, 4096)
	n, err := conn.Read(response)
	return response[:n], err
}

func TestSplitDNSAndNftsetInNetworkNamespace(t *testing.T) {
	if os.Getenv("POL_TEST_NETNS") != "1" {
		t.Skip("run with POL_TEST_NETNS=1 inside an isolated network namespace")
	}
	oldGeo, oldRun := geoDir, runtimeDir
	geoDir = t.TempDir()
	runtimeDir = t.TempDir()
	defer func() { geoDir = oldGeo; runtimeDir = oldRun }()
	_ = os.WriteFile(filepath.Join(geoDir, "geoip.dat"), testGeoFile("geoip"), 0600)
	_ = os.WriteFile(filepath.Join(geoDir, "geosite.dat"), testGeoFile("geosite"), 0600)
	for _, args := range [][]string{{"link", "set", "lo", "up"}, {"link", "add", "br-lan", "type", "dummy"}, {"addr", "add", "192.0.2.1/24", "dev", "br-lan"}, {"link", "set", "br-lan", "up"}} {
		if out, err := exec.Command("ip", args...).CombinedOutput(); err != nil {
			t.Fatalf("network namespace setup: %v %s", err, out)
		}
	}
	s := defaults()
	s.RouteIran = true
	s.IranDNS = "127.0.0.2"
	rules, err := networkRules(s)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("nft", "-f", "-")
	cmd.Stdin = strings.NewReader(rules)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("nft rules: %v\n%s", err, out)
	}
	direct := testDNSStub(t, "127.0.0.2:53", []byte{203, 0, 113, 7})
	tunnel := testDNSStub(t, "127.0.0.1:24812", []byte{203, 0, 113, 8})
	path := filepath.Join(runtimeDir, "dnsmasq.conf")
	_ = os.WriteFile(path, []byte(iranResolverConfig(s, []string{"iran.example"})), 0600)
	resolver := exec.Command("dnsmasq", "--keep-in-foreground", "--conf-file="+path)
	logPath := filepath.Join(runtimeDir, "resolver.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()
	logs := func() string { b, _ := os.ReadFile(logPath); return string(b) }
	resolver.Stderr = logFile
	if err = resolver.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resolver.Process.Kill(); _ = resolver.Wait() }()
	var response []byte
	for i := 0; i < 40; i++ {
		response, err = testDNSQuery("iran.example")
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("split resolver did not answer: %v %s", err, logs())
	}
	if !bytes.HasSuffix(response, []byte{203, 0, 113, 7}) {
		t.Fatal("Iran domain did not use direct DNS")
	}
	response, err = testDNSQuery("foreign.example")
	if err != nil || !bytes.HasSuffix(response, []byte{203, 0, 113, 8}) {
		t.Fatalf("foreign domain did not use tunnel DNS: %v", err)
	}
	if len(direct) == 0 || len(tunnel) == 0 {
		t.Fatal("split DNS upstream selection missing")
	}
	list := func() string {
		out, err := exec.Command("nft", "list", "set", "inet", "pol", "iran_domains4").CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
	time.Sleep(100 * time.Millisecond)
	if text := list(); !strings.Contains(text, "203.0.113.7") || strings.Contains(text, "203.0.113.8") {
		t.Fatalf("wrong Iran nftset contents: %s\n%s", text, logs())
	}
	snapshotIranDomains()
	if out, err := exec.Command("nft", "flush", "set", "inet", "pol", "iran_domains4").CombinedOutput(); err != nil {
		t.Fatalf("flush test set: %v %s", err, out)
	}
	if err = restoreIranDomains(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(list(), "203.0.113.7") {
		t.Fatal("restart lost live Iran domain mappings")
	}
	t.Log("Iran DNS stayed direct; foreign DNS stayed in the tunnel; foreign-hosted Iranian domain entered the pre-TProxy nftset; live mappings survived restart")
}
