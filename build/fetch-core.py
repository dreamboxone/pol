#!/usr/bin/env python3
"""Fetch a release core and require GitHub's SHA-256 asset digest."""
import hashlib
import io
import json
import pathlib
import sys
import urllib.request
import zipfile

ARCHES = {
    "arm_cortex-a7_neon-vfpv4": "arm32-v7a",
    "aarch64_generic": "arm64-v8a",
    "aarch64_cortex-a53": "arm64-v8a",
    "x86_64": "64",
    "mips_24kc": "mips32",
    "mipsel_24kc": "mips32le",
}

def main():
    arch = sys.argv[1] if len(sys.argv) > 1 else "arm_cortex-a7_neon-vfpv4"
    tag = sys.argv[2] if len(sys.argv) > 2 else "v26.10.21"
    name = f"Xray-linux-{ARCHES[arch]}.zip"
    request = urllib.request.Request(
        f"https://api.github.com/repos/XTLS/Xray-core/releases/tags/{tag}",
        headers={"User-Agent": "Pol-OpenWrt", "Accept": "application/vnd.github+json"},
    )
    with urllib.request.urlopen(request, timeout=35) as response:
        release = json.load(response)
    asset = next(a for a in release["assets"] if a["name"] == name)
    with urllib.request.urlopen(asset["browser_download_url"], timeout=180) as response:
        data = response.read(100 * 1024 * 1024 + 1)
    if len(data) > 100 * 1024 * 1024:
        raise ValueError("Core download exceeds size limit")
    digest = asset.get("digest", "").removeprefix("sha256:")
    if len(digest) != 64 or hashlib.sha256(data).hexdigest() != digest:
        raise ValueError("Missing or invalid release SHA-256 digest")
    dest = pathlib.Path(__file__).resolve().parent.parent / "prebuilt" / arch
    dest.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        (dest / "xray").write_bytes(archive.read("xray"))
    (dest / "xray").chmod(0o755)
    (dest / "version.txt").write_text(tag + "\n")
    print(f"Verified {tag}: {dest / 'xray'}")

if __name__ == "__main__":
    main()
