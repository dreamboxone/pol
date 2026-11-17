package main

import (
	networking "bpb/networking"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

func probeDNS(input object) (any, error) {
	b, _ := json.Marshal(input["request"])
	var request networking.DNSRequest
	if e := json.Unmarshal(b, &request); e != nil {
		return nil, fmt.Errorf("invalid DNS probe request")
	}
	if !contains([]string{"udp", "tcp", "doh", "doq"}, request.Type) || len(request.Host) > 253 || strings.ContainsAny(request.Host, " /\r\n\t") || len(request.IPs) > 8 {
		return nil, fmt.Errorf("invalid DNS probe server")
	}
	for _, ip := range request.IPs {
		if net.ParseIP(ip) == nil {
			return nil, fmt.Errorf("invalid DNS probe IP")
		}
	}
	if request.Type == "udp" && len(request.IPs) == 0 {
		return nil, fmt.Errorf("UDP DNS probe requires an IP address")
	}
	request.Endpoint = "http://cp.cloudflare.com/generate_204"
	request.TimeoutMs = 5000
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	return networking.CheckDNS(ctx, request), nil
}
