# Pol

[فارسی](README.md)

An OpenWrt VPN client with a bilingual LuCI interface, based on [BPB-VPN](https://github.com/bia-pain-bache/BPB-VPN)'s Worker protocol and configuration design.

- Persian and English, RTL and LTR, dark and light themes
- Main button controlling the service and LAN routing
- Immediate settings with startup validation and rollback; no Save & Apply
- Cloudflare Worker deployment, WARP and MASQUE registration, and serverless fragmentation
- VLESS, Trojan, SOCKS and HTTP URI import; Xray JSON outbound import
- IPv4 and IPv6 LAN routing, DNS interception, direct and blocked domains, and a chain proxy
- Live application and core reports with filtering and download
- Independent `pol` and `pol-core` packages in IPK and APK formats
- Separate application and core updates with release asset SHA-256 verification

This is a native OpenWrt adaptation, not the Android interface. Android-specific features, embedded Browser Dialer, MITM Domain Fronting and Cloudflare Pages deployment are not included. Tor and Psiphon modes connect to separately installed local SOCKS services; their binaries and bridge management are not bundled.

## Install

Download matching packages from [Releases](https://github.com/dreamboxone/pol/releases).

```sh
# APK-based OpenWrt
apk add --allow-untrusted ./pol-core_26.10.21-r1_arm_cortex-a7_neon-vfpv4.apk ./pol_1.0.0-r1_arm_cortex-a7_neon-vfpv4.apk

# OPKG-based OpenWrt
opkg install ./pol-core_26.10.21-1_arm_cortex-a7_neon-vfpv4.ipk ./pol_1.0.0-1_arm_cortex-a7_neon-vfpv4.ipk
```

Open **Services → Pol**. The service starts disabled. Dependencies are `luci-base`, `rpcd`, `ca-bundle`, `ip-full`, `nftables` and `kmod-nft-tproxy`.

## Connect

In Cloudflare, create an API token scoped to the intended account with Workers Scripts Edit or equivalent Workers Admin permissions. Set up your account's workers.dev subdomain. Enter the token, Account ID and a new Worker name in Pol's Cloudflare tab. The app deploys the bundled Worker and generates the UUID. The token is used in memory and is never saved in settings, command arguments or reports.

Deployment creates a new remote Worker. It does not replace an existing Worker. If local settings fail after remote deployment, manage the created Worker from your Cloudflare dashboard.

For WARP or MASQUE, use the matching registration button in Connection. The router needs access to Cloudflare's registration API. Turn on the main button after configuring a connection.

## Networking

Transparent routing captures traffic arriving from the configured LAN interface. Router-originated traffic and router administration remain outside capture. Private networks bypass the tunnel. DNS queries are intercepted before private-network bypass.

Local SOCKS: `127.0.0.1:24800`. Local HTTP: `127.0.0.1:24801`. Internal TPROXY and DNS ports: 24810–24813. Routing table: 120. Mark bit: `0x40000000`. The program refuses to take over an occupied routing table or known competing transparent proxy tables. Disable LAN routing for proxy-only use.

Serverless fragmentation does not change your public IP and cannot guarantee connectivity on every network. Tor requires a local SOCKS listener on 9050; Psiphon requires one on 10829. Install and configure those services separately.

Reports come from OpenWrt's bounded logd ring with `pol-app` and `pol-core` tags. They are not persistent across reboots. Exported settings contain private connection keys.

## Updates

Core updates download the architecture-matched official XTLS/Xray-core asset, verify its SHA-256 digest, validate the executable and current config, and retain a rollback binary. App updates install only the matching Pol package from `dreamboxone/pol` using APK or OPKG. Refresh the page after installation and inspect Reports for the result. Long operations run as background jobs rather than blocking rpcd.

## Build

Requires Linux or WSL, Go 1.26+, Python 3 and GNU tar. APK builds also require apk-tools 3 from an OpenWrt SDK.

```sh
python3 build/fetch-core.py arm_cortex-a7_neon-vfpv4
export APK_BIN=/opt/openwrt-sdk/staging_dir/host/bin/apk
sh build/build.sh arm_cortex-a7_neon-vfpv4 both
```

Packages are written to `dist/`. The second argument accepts `ipk`, `apk` or `both`. Supported build targets: `arm_cortex-a7_neon-vfpv4`, `aarch64_generic`, `aarch64_cortex-a53`, `x86_64`, `mips_24kc`, `mipsel_24kc`. Hardware validation is currently limited to ARMv7. The SDK Makefiles in `package/` consume previously built binaries; the primary tested packaging path is `build/build.sh`.

```sh
go test -race ./...
go vet ./...
node --check package/pol/files/www/luci-static/resources/view/pol/main.js
```

## License

Pol and the bundled Worker: GPL-3.0-only. Xray-core: MPL-2.0. Vazirmatn: OFL-1.1. See [NOTICE](NOTICE) and the included licenses for provenance.
