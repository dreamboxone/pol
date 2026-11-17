#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
ARCH=${1:-arm_cortex-a7_neon-vfpv4}
FORMAT=${2:-both}
VERSION=1.8.0
CORE_VERSION=${CORE_VERSION:-26.11.9}
GO=${GO_BIN:-go}
APK=${APK_BIN:-}
set -- $(python3 "$ROOT/build/architectures.py" "$ARCH")
GOARCH=$1
GOARM=$2
[ "$GOARM" != - ] || GOARM=
case "$FORMAT" in ipk|apk|both) ;; *) echo 'Format must be ipk, apk or both' >&2; exit 1 ;; esac

if [ "$FORMAT" != ipk ] && [ -z "$APK" ]; then
  APK=$(find /opt "$HOME" -maxdepth 6 -path '*/staging_dir/host/bin/apk' -type f 2>/dev/null | head -1)
fi
if [ "$FORMAT" != ipk ] && { [ -z "$APK" ] || [ ! -x "$APK" ]; }; then
  echo 'Set APK_BIN to the OpenWrt SDK apk-tools 3 binary' >&2; exit 1
fi
[ -x "$ROOT/prebuilt/$ARCH/xray" ] || { echo "Run build/fetch-core.py $ARCH first" >&2; exit 1; }
mkdir -p "$ROOT/build/bin/$ARCH" "$ROOT/dist"
(cd "$ROOT/third_party/bpb/psiphon" && GOTOOLCHAIN=go1.26.9 CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" GOARM="$GOARM" GOMIPS=softfloat GOMIPS64=softfloat "$GO" build -trimpath -buildvcs=false -ldflags="-s -w -checklinkname=0" -o "$ROOT/build/bin/$ARCH/psiphon" .)
(cd "$ROOT/third_party/lyrebird" && CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" GOARM="$GOARM" GOMIPS=softfloat GOMIPS64=softfloat "$GO" build -trimpath -buildvcs=false -ldflags="-s -w -X main.lyrebirdVersion=0.10.0" -o "$ROOT/build/bin/$ARCH/lyrebird" ./cmd/lyrebird)
(cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" GOARM="$GOARM" GOMIPS=softfloat GOMIPS64=softfloat "$GO" build -trimpath -ldflags="-s -w -X main.buildPackageArch=$ARCH" -o "build/bin/$ARCH/pol" ./cmd/pol)

WORK=$(mktemp -d /tmp/pol-package.XXXXXX)
trap 'rm -rf "$WORK"' EXIT HUP INT TERM
mkdir -p "$WORK/pol" "$WORK/pol/usr/lib/pol" "$WORK/pol/usr/share/licenses/pol-core"
cp -R "$ROOT/package/pol/files/." "$WORK/pol/"
# Versioned view names avoid reusing a cached interface after upgrades.
viewdir="$WORK/pol/www/luci-static/resources/view/pol"
view_version=$(printf '%s' "$VERSION" | tr '.' '_')
mv "$viewdir/main.js" "$viewdir/main-$view_version.js"
sed -i "s@pol/main\"@pol/main-$view_version\"@" "$WORK/pol/usr/share/luci/menu.d/luci-app-pol.json"

mkdir -p "$WORK/pol/usr/libexec/rpcd" "$WORK/pol/usr/share/licenses/pol"
cp "$ROOT/build/bin/$ARCH/pol" "$WORK/pol/usr/libexec/rpcd/pol"
cp "$ROOT/build/bin/$ARCH/psiphon" "$WORK/pol/usr/lib/pol/psiphon"
cp "$ROOT/build/bin/$ARCH/lyrebird" "$WORK/pol/usr/lib/pol/lyrebird"
mkdir -p "$WORK/pol/usr/share/licenses/lyrebird"
cp "$ROOT/third_party/lyrebird/LICENSE" "$ROOT/third_party/lyrebird/LICENSE-GPL3.txt" "$WORK/pol/usr/share/licenses/lyrebird/"
mkdir -p "$WORK/pol/usr/share/licenses/bpb"
cp "$ROOT/third_party/bpb/LICENSE" "$WORK/pol/usr/share/licenses/bpb/LICENSE"
cp "$ROOT/LICENSE" "$ROOT/NOTICE" "$ROOT/licenses/Zirgozar-AGPL-3.0.txt" "$WORK/pol/usr/share/licenses/pol/"
cp "$ROOT/prebuilt/$ARCH/xray" "$WORK/pol/usr/lib/pol/xray"
ln -s xray "$WORK/pol/usr/lib/pol/pol-core"
cp "$ROOT/licenses/Xray-MPL-2.0.txt" "$WORK/pol/usr/share/licenses/pol-core/LICENSE"

for name in pol; do
  stage="$WORK/$name"
  find "$stage" -type d -exec chmod 0755 {} +
  find "$stage" -type f -exec chmod 0644 {} +
  if [ "$name" = pol ]; then
    ver=$VERSION; license='GPL-3.0-only AND AGPL-3.0-or-later'
    deps='luci-base rpcd ca-bundle ip-full nftables kmod-nft-tproxy tor tor-geoip'
    desc='Pol VPN with LuCI and Xray for OpenWrt'
    chmod 0755 "$stage/usr/libexec/rpcd/pol" "$stage/etc/init.d/pol" "$stage/usr/lib/pol/xray" "$stage/usr/lib/pol/psiphon" "$stage/usr/lib/pol/lyrebird"
    chmod 0600 "$stage/etc/pol/settings.json"
  fi
  admin="$WORK/admin-$name"; mkdir -p "$admin"
  cat > "$admin/postinst" <<'EOF'
#!/bin/sh
[ -n "${IPKG_INSTROOT:-}" ] && exit 0
if [ -x /usr/libexec/rpcd/pol ]; then
  chmod 0600 /etc/pol/settings.json
  rm -f /tmp/luci-indexcache /tmp/luci-indexcache.*.json
  /etc/init.d/rpcd restart
  if /usr/libexec/rpcd/pol is-enabled; then /etc/init.d/pol start; fi
fi
exit 0
EOF
  cat > "$admin/prerm" <<'EOF'
#!/bin/sh
[ -n "${IPKG_INSTROOT:-}" ] || /etc/init.d/pol stop
exit 0
EOF
  chmod 0755 "$admin/postinst" "$admin/prerm"
  if [ "$FORMAT" != apk ]; then
    control="$WORK/control-$name"; mkdir -p "$control"
    {
      echo "Package: $name"
      echo "Version: $ver-1"
      echo "Architecture: $ARCH"
      echo "License: $license"
      echo 'Section: net'
      echo "Depends: $(echo "$deps" | sed 's/ /, /g')"
      echo "Conflicts: pol-core"
      echo "Replaces: pol-core"
      echo "Installed-Size: $(du -sk "$stage" | cut -f1)"
      echo "Description: $desc"
    } > "$control/control"
    if [ "$name" = pol ]; then echo /etc/pol/settings.json > "$control/conffiles"; cp "$admin/postinst" "$control/"; cp "$admin/prerm" "$control/"; fi
    tar --numeric-owner --owner=0 --group=0 -C "$stage" -czf "$WORK/data.tar.gz" .
    tar --numeric-owner --owner=0 --group=0 -C "$control" -czf "$WORK/control.tar.gz" .
    printf '2.0\n' > "$WORK/debian-binary"
    tar --numeric-owner --owner=0 --group=0 -C "$WORK" -czf "$ROOT/dist/${name}_${ver}-1_${ARCH}.ipk" debian-binary control.tar.gz data.tar.gz
  fi
  if [ "$FORMAT" != ipk ]; then
    mkdir -p "$stage/lib/apk/packages"
    (cd "$stage" && find . -type f | sed 's|^\.||' | sort) > "$WORK/$name.list"
    cp "$WORK/$name.list" "$stage/lib/apk/packages/$name.list"
    set --
    if [ "$name" = pol ]; then
      echo /etc/pol/settings.json > "$stage/lib/apk/packages/pol.conffiles"
      echo "/etc/pol/settings.json $(sha256sum "$stage/etc/pol/settings.json" | cut -d' ' -f1)" > "$stage/lib/apk/packages/pol.conffiles_static"
      set -- --script "post-install:$admin/postinst" --script "post-upgrade:$admin/postinst" --script "pre-deinstall:$admin/prerm" --script "pre-upgrade:$admin/prerm"
    fi
    "$APK" mkpkg --info "name:$name" --info "version:$ver-r1" --info "description:$desc" --info "arch:$ARCH" --info "license:$license" --info "origin:$name" --info "depends:$deps !pol-core" --info "replaces:pol-core" "$@" --files "$stage" --output "$ROOT/dist/${name}_${ver}-r1_${ARCH}.apk"
  fi
done
(cd "$ROOT/dist" && sha256sum ./*.ipk ./*.apk 2>/dev/null > SHA256SUMS || true)
echo "Packages built in $ROOT/dist"
