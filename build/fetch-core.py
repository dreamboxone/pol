#!/usr/bin/env python3
"""Fetch the matching core binary and verify its release digest."""
import hashlib
import io
import json
import os
import pathlib
import sys
import time
import urllib.request
import zipfile

ROOT = pathlib.Path(__file__).resolve().parent.parent
TARGETS = json.loads((ROOT / "cmd/pol/architectures.json").read_text())


def fetch(url, limit, api=False):
    headers = {"User-Agent": "Pol-OpenWrt"}
    if api:
        headers["Accept"] = "application/vnd.github+json"
        token = os.environ.get("GITHUB_TOKEN")
        if token:
            headers["Authorization"] = "Bearer " + token
    for attempt in range(3):
        try:
            request = urllib.request.Request(url, headers=headers)
            with urllib.request.urlopen(request, timeout=180) as response:
                data = response.read(limit + 1)
            if len(data) > limit:
                raise ValueError("Download exceeds size limit")
            return data
        except (OSError, TimeoutError):
            if attempt == 2:
                raise
            time.sleep(attempt + 1)


def main():
    arch = sys.argv[1] if len(sys.argv) > 1 else "arm_cortex-a7_neon-vfpv4"
    tag = sys.argv[2] if len(sys.argv) > 2 else "v26.11.9"
    target = TARGETS[arch]
    release = json.loads(fetch(
        f"https://api.github.com/repos/XTLS/Xray-core/releases/tags/{tag}",
        4 * 1024 * 1024, api=True))
    asset = next(a for a in release["assets"] if a["name"] == target["asset"])
    data = fetch(asset["browser_download_url"], 100 * 1024 * 1024)
    digest = asset.get("digest", "").removeprefix("sha256:")
    if len(digest) != 64 or hashlib.sha256(data).hexdigest() != digest:
        raise ValueError("Missing or invalid release SHA-256 digest")
    dest = ROOT / "prebuilt" / arch
    dest.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        binary = archive.read(target["binary"])
    if not binary.startswith(b"\x7fELF"):
        raise ValueError("The selected core is not an ELF executable")
    (dest / "xray").write_bytes(binary)
    (dest / "xray").chmod(0o755)
    (dest / "version.txt").write_text(tag + "\n")
    print(f"Verified {tag}: {dest / 'xray'}")


if __name__ == "__main__":
    main()
