package main

import "testing"

func TestCoreUpdateUsesExactPackageArchitecture(t *testing.T) {
	old := buildPackageArch
	defer func() { buildPackageArch = old }()
	for _, tc := range []struct{ arch, asset, binary string }{
		{"arm_arm926ej-s", "Xray-linux-arm32-v5.zip", "xray"},
		{"arm_arm1176jzf-s_vfp", "Xray-linux-arm32-v6.zip", "xray"},
		{"arm_cortex-a9_neon", "Xray-linux-arm32-v7a.zip", "xray"},
		{"aarch64_cortex-a76", "Xray-linux-arm64-v8a.zip", "xray"},
		{"mipsel_74kc", "Xray-linux-mips32le.zip", "xray_softfloat"},
		{"mips_4kec", "Xray-linux-mips32.zip", "xray_softfloat"},
		{"mips64el_mips64r2", "Xray-linux-mips64le.zip", "xray"},
		{"loongarch64_generic", "Xray-linux-loong64.zip", "xray"},
	} {
		buildPackageArch = tc.arch
		asset, err := coreAsset()
		if err != nil || packageArch() != tc.arch || asset != tc.asset || architectures()[tc.arch].Binary != tc.binary {
			t.Fatalf("incorrect update target for %s: %s %v", tc.arch, asset, err)
		}
	}
	buildPackageArch = "unknown"
	if _, err := coreAsset(); err == nil {
		t.Fatal("unknown package architecture accepted")
	}
}
