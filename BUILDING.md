# ساخت پل از سورس

## 🏗️ ساخت از سورس

ساخت در Linux یا WSL انجام می‌شود. هستهٔ پیش‌فرض ساخت، Xray 26.11.9 است. Go 1.26 یا جدیدتر، Python 3 و برای APK ابزار `apk` نسخهٔ ۳ از SDK رسمی OpenWrt لازم است.

```sh
python3 build/fetch-core.py arm_cortex-a7_neon-vfpv4
export APK_BIN=/opt/openwrt-sdk/staging_dir/host/bin/apk
sh build/build.sh arm_cortex-a7_neon-vfpv4 both
```

خروجی در `dist/` است. قالب دوم `ipk`، `apk` یا `both` است. معماری‌های قابل ساخت: `arm_cortex-a7_neon-vfpv4`، `aarch64_generic`، `aarch64_cortex-a53`، `x86_64`، `mips_24kc` و `mipsel_24kc`. وجود مسیر ساخت به معنی آزمایش روی همهٔ سخت‌افزارها نیست؛ آزمایش قبلی سخت‌افزاری پروژه فقط برای ARMv7 گزارش شده است.

```sh
go test -race ./...
go vet ./...
node --check package/pol/files/www/luci-static/resources/view/pol/main.js
```

Makefileهای SDK در `package/` موجودند؛ ابتدا باینری‌ها را با اسکریپت ساخت آماده کنید. روش اصلی ساخت `build/build.sh` است. ساخت محلی یا CI بسته‌ها را خودکار به‌عنوان Release منتشر نمی‌کند.


فهرست معماری‌ها در `cmd/pol/architectures.json` است. برنامه، LuCI و هسته داخل یک بسته قرار دارند.
