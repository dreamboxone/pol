<p align="center"><img src="package/pol/files/www/luci-static/resources/pol/logo.svg" width="180" alt="Pol logo"></p>

# Pol

[فارسی](README.md)

An OpenWrt VPN client with a bilingual LuCI interface, based on [BPB-VPN](https://github.com/bia-pain-bache/BPB-VPN)'s Worker protocol and configuration design.

- Persian and English, RTL and LTR, dark, light and system themes
- Main Switch controlling the service and LAN routing
- Immediate settings with startup validation and rollback; no Save & Apply
- Cloudflare Worker deployment, WARP and MASQUE registration, and serverless fragmentation
- TLS fingerprint, Worker SNI, ECH (normal Worker mode), TCP Fast Open and DNS IPv4/IPv6 preference
- Tunable UDP noise, WARP MTU, keepalive and reserved bytes
- Real-core HTTPS endpoint scanning with manual candidate lists and explicit result selection
- VLESS, Trojan, SOCKS and HTTP URI import; Xray JSON outbound import
- IPv4 and IPv6 LAN routing, DNS interception, direct and blocked domains, and a chain proxy
- Live application and core reports with filtering and download
- One complete `pol` package including the application, LuCI and Xray, in IPK or APK format
- Separate application and core updates with release asset SHA-256 verification

Tor and Psiphon modes connect to separately configured local SOCKS services.

## Router resources

Plan for at least 256 MB of router RAM (512 MB or more for full routing data and more concurrent clients), and 150 MB free storage (250 MB recommended for routing data and update headroom). The installed application, LuCI and bundled core use approximately 41–45 MB depending on architecture. The Iranian Lite routing files take about 2 MB; full files vary by provider. Runtime RAM varies with traffic, active connections and routing data. Updates temporarily retain both new and previous files.

## Install

Place only the matching Pol installer in `/tmp`. Download it from [Releases](https://github.com/dreamboxone/pol/releases).

```sh
# APK-based OpenWrt
apk add --allow-untrusted /tmp/*.apk

# OPKG-based OpenWrt
opkg install /tmp/*.ipk
```

Open **Services → Pol**. The service starts disabled. Dependencies are `luci-base`, `rpcd`, `ca-bundle`, `ip-full`, `nftables` and `kmod-nft-tproxy`.

## Connect

In Cloudflare, create an API token scoped to the intended account with Workers Scripts Edit or equivalent Workers Admin permissions. Set up your account's workers.dev subdomain. Enter the token, Account ID and a new Worker name in Pol's Cloudflare tab. The app deploys the bundled Worker and generates the UUID. The token is used in memory and is never saved in settings, command arguments or reports.

Deployment creates a new remote Worker. It does not replace an existing Worker. If local settings fail after remote deployment, manage the created Worker from your Cloudflare dashboard.

For WARP or MASQUE, use the matching registration button in Connection. The router needs access to Cloudflare's registration API. Turn on the main button after configuring a connection.

## Advanced connection settings and scanning

In Connection, tune TLS fingerprints, optional Worker SNI, TCP Fast Open and the DNS address family. ECH takes an IP or HTTPS DoH URL, is limited to normal Worker connections, and must be cleared before enabling fragmentation. WARP MTU accepts 1280–1500 and keepalive accepts 0–65535 seconds. UDP noise size, delay and reset accept ascending numeric ranges. Serverless combines TCP fragmentation and UDP noise; it keeps your public IP.

The scanner accepts up to 12 unique candidates: IPs or hostnames for Worker, host:port or [IPv6]:port for WARP/MASQUE. Each candidate runs through a temporary loopback-only Xray instance and must complete a verified HTTPS request to www.gstatic.com/generate_204. Probes run sequentially to bound RAM use and leave the active service untouched. A result measures request latency, not download speed. Click Use this address to apply a successful candidate. Existing chain settings are included in probes.

Installing Pol does not modify or delete previously deployed Cloudflare Workers.

See the [complete Persian guide](README.md) for all fields, troubleshooting and removal instructions. [Telegram support](https://t.me/routekernel1).

## Dashboard and Iranian routing

The four cards follow the router dashboard layout: status/server/verified exit flag, core/version/process state, individual Telegram/YouTube/Google connectivity checks, and traffic usage with vertical today/7-day/30-day selectors. Persian reads these cards right to left. The main slider sits below the cards. Stopping it terminates tunnel processes and checks and cancels active network jobs. Usage survives page reloads, checkpoints every five minutes and on stop, and excludes traffic bypassing the core.

In Routing, choose GeoIP and GeoSite sources and update them independently. Each file shows its size and update timestamp. Files must contain usable IR entries. Enable Iranian traffic bypass after downloading both files. GeoIP bypasses TProxy in nftables; a separate split DNS resolver resolves GeoSite domains directly and adds their response IPs to nftsets. This bypass precedes filtering. dnsmasq must support nftset. Failed downloads preserve the old file. Option help is available beside each relevant title; there is no About tab.

## Networking

Transparent routing captures traffic arriving from the configured LAN interface. Router-originated traffic and router administration remain outside capture. Private networks bypass the tunnel. DNS queries are intercepted before private-network bypass.

Local SOCKS: `127.0.0.1:24800`. Local HTTP: `127.0.0.1:24801`. Internal TPROXY and DNS ports: 24810–24813. Routing table: 120. Mark bit: `0x40000000`. The program refuses to take over an occupied routing table or known competing transparent proxy tables. Disable LAN routing for proxy-only use.

Serverless fragmentation does not change your public IP and cannot guarantee connectivity on every network. Tor requires a local SOCKS listener on 9050; Psiphon requires one on 10829. Install and configure those services separately.

Reports come from OpenWrt's bounded logd ring with `pol-app` and `pol-core` tags. They are not persistent across reboots. Exported settings contain private connection keys.

## Updates

Core updates download the architecture-matched official XTLS/Xray-core asset, verify its SHA-256 digest, validate the executable and current config, and retain a rollback binary. App updates install the complete matching Pol package from `dreamboxone/pol` using APK or OPKG. Refresh the page after installation and inspect Reports for the result. Long operations run as background jobs rather than blocking rpcd.

## Remove

Export settings from Connection first, then stop and disable Pol:

```sh
/etc/init.d/pol stop
/etc/init.d/pol disable
```

Use `apk del pol` on APK-based OpenWrt, or `opkg remove pol` on OPKG-based OpenWrt. This removes the application, LuCI and bundled core together. Modified settings may remain in `/etc/pol`. Removing Pol does not delete Workers in your Cloudflare account. The [Persian guide](README.md) includes all 29 package architectures and detailed instructions.

## License

Pol and the bundled Worker: GPL-3.0-only. Dashboard styles adapted from Zirgozar: AGPL-3.0-or-later. Xray-core: MPL-2.0. Vazirmatn: OFL-1.1. See [NOTICE](NOTICE) and the included licenses for provenance.

System dependencies are declared in the package: luci-base, rpcd, ca-bundle, ip-full, nftables and kmod-nft-tproxy. The package manager keeps installed dependencies and fetches missing ones from the matching OpenWrt repositories, so repository access is required. Pol includes its own management view and Xray, while these system packages remain dependencies. Iran bypass additionally requires dnsmasq with nftset support; use a compatible dnsmasq-full package if the installed dnsmasq lacks that feature.
