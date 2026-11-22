package main

import (
	"fmt"
	"strings"
)

type MaskRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func (r MaskRange) text() string {
	if r.Min == r.Max {
		return fmt.Sprint(r.Min)
	}
	return fmt.Sprintf("%d-%d", r.Min, r.Max)
}
func (r MaskRange) valid() bool { return r.Min >= 0 && r.Max >= r.Min && r.Max <= 65535 }

type FragmentRule struct {
	Packets  string      `json:"packets"`
	Lengths  []MaskRange `json:"lengths"`
	Delays   []MaskRange `json:"delays"`
	MaxSplit *MaskRange  `json:"maxSplit,omitempty"`
}
type NoiseRule struct {
	Type   string    `json:"type"`
	Size   MaskRange `json:"size"`
	Bytes  []int     `json:"bytes,omitempty"`
	Packet string    `json:"packet,omitempty"`
	Count  int       `json:"count"`
	Delay  MaskRange `json:"delay"`
}

func selectedFragments(s Settings) []FragmentRule {
	if s.Mode == "masque" && len(s.MasqueFragments) > 0 {
		return s.MasqueFragments
	}
	if s.Mode == "serverless" && len(s.ServerlessFragments) > 0 {
		return s.ServerlessFragments
	}
	return s.Fragments
}
func selectedNoises(s Settings) []NoiseRule {
	if s.Mode == "masque" && len(s.MasqueNoises) > 0 {
		return s.MasqueNoises
	}
	if s.Mode == "serverless" && len(s.ServerlessNoises) > 0 {
		return s.ServerlessNoises
	}
	return s.UDPNoises
}
func fragmentRules(s Settings) []any {
	var rules []any
	for _, f := range selectedFragments(s) {
		packets := f.Packets
		if strings.EqualFold(packets, "tlshello") || packets == "" {
			packets = "tlshello"
		}
		var lengths, delays []string
		for _, r := range f.Lengths {
			lengths = append(lengths, r.text())
		}
		for _, r := range f.Delays {
			delays = append(delays, r.text())
		}
		settings := object{"packets": packets, "lengths": lengths, "delays": delays}
		if f.MaxSplit != nil {
			settings["maxSplit"] = f.MaxSplit.text()
		}
		rules = append(rules, object{"type": "fragment", "settings": settings})
	}
	return rules
}
func noiseRules(s Settings) []any {
	var rules []any
	noises := selectedNoises(s)
	if len(noises) == 0 && s.NoiseSize == "40-70" && s.NoiseDelay == "0" {
		noises = []NoiseRule{
			{Type: "exp", Packet: "<b 0x123401000001000000000000076578616d706c6503636f6d0000010001>", Count: 1},
			{Type: "rand", Size: MaskRange{Min: 40, Max: 70}, Count: 4},
		}
	}
	for _, n := range noises {
		count := n.Count
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			entry := object{"delay": n.Delay.text()}
			switch n.Type {
			case "rand":
				entry["rand"] = n.Size.text()
				entry["randRange"] = "0-255"
			case "array":
				entry["type"] = "array"
				entry["packet"] = n.Bytes
			default:
				entry["type"] = n.Type
				entry["packet"] = n.Packet
			}
			rules = append(rules, entry)
		}
	}
	return rules
}
func validateMasks(s Settings) error {
	for _, list := range [][]FragmentRule{s.Fragments, s.MasqueFragments, s.ServerlessFragments} {
		if len(list) > 32 {
			return fmt.Errorf("too many fragment rules")
		}
		for _, f := range list {
			if !contains([]string{"", "tlsHello", "tlshello", "1-1", "1-2", "1-3", "1-5"}, f.Packets) {
				return fmt.Errorf("invalid fragment packet selection")
			}
			if len(f.Lengths) == 0 || len(f.Lengths) > 32 || len(f.Delays) == 0 || len(f.Delays) > 32 {
				return fmt.Errorf("fragment length and delay ranges are required")
			}
			for _, r := range append(append([]MaskRange{}, f.Lengths...), f.Delays...) {
				if !r.valid() {
					return fmt.Errorf("invalid fragment range")
				}
			}
			if f.MaxSplit != nil && !f.MaxSplit.valid() {
				return fmt.Errorf("invalid fragment maxSplit")
			}
		}
	}
	for _, list := range [][]NoiseRule{s.UDPNoises, s.MasqueNoises, s.ServerlessNoises} {
		total := 0
		for _, n := range list {
			count := n.Count
			if count == 0 {
				count = 1
			}
			total += count
			if count < 1 || count > 100 || total > 256 {
				return fmt.Errorf("too many UDP noise packets")
			}
			if !contains([]string{"rand", "str", "hex", "base64", "array", "exp"}, n.Type) || !n.Delay.valid() || !n.Size.valid() {
				return fmt.Errorf("invalid UDP noise")
			}
			if len(n.Packet) > 4096 || len(n.Bytes) > 4096 {
				return fmt.Errorf("UDP noise payload too large")
			}
			for _, b := range n.Bytes {
				if b < 0 || b > 255 {
					return fmt.Errorf("invalid UDP noise byte")
				}
			}
		}
	}
	return nil
}
