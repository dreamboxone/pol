package main

import (
	_ "embed"
	"encoding/json"
	"runtime"
)

//go:embed architectures.json
var architectureCatalog []byte

var buildPackageArch string

type architecture struct {
	Goarch string `json:"goarch"`
	Goarm  string `json:"goarm"`
	Asset  string `json:"asset"`
	Binary string `json:"binary"`
}

func architectures() map[string]architecture {
	var targets map[string]architecture
	if err := json.Unmarshal(architectureCatalog, &targets); err != nil {
		panic(err)
	}
	return targets
}

func packageArch() string {
	if buildPackageArch != "" {
		return buildPackageArch
	}
	return map[string]string{"arm": "arm_cortex-a7_neon-vfpv4", "arm64": "aarch64_generic", "amd64": "x86_64", "386": "i386_pentium4", "mips": "mips_24kc", "mipsle": "mipsel_24kc", "mips64": "mips64_mips64r2", "mips64le": "mips64el_mips64r2", "riscv64": "riscv64_riscv64"}[runtime.GOARCH]
}
