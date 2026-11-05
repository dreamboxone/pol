include $(TOPDIR)/rules.mk
POL_SOURCE_DIR?=$(abspath $(CURDIR)/../..)

PKG_NAME:=pol
PKG_VERSION:=1.0.1
PKG_RELEASE:=1
PKG_LICENSE:=GPL-3.0-only
PKG_BUILD_DIR:=$(BUILD_DIR)/$(PKG_NAME)

include $(INCLUDE_DIR)/package.mk

define Package/pol
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Pol VPN for OpenWrt
  DEPENDS:=+luci-base +rpcd +ca-bundle +ip-full +nftables +kmod-nft-tproxy +pol-core
endef

define Package/pol/conffiles
/etc/pol/settings.json
endef

define Build/Prepare
endef

define Build/Compile
endef

define Package/pol/install
	$(CP) $(POL_SOURCE_DIR)/package/pol/files/* $(1)/
	$(INSTALL_DIR) $(1)/usr/libexec/rpcd
	$(INSTALL_BIN) $(POL_SOURCE_DIR)/build/bin/$(ARCH)/pol $(1)/usr/libexec/rpcd/pol
	$(INSTALL_DIR) $(1)/usr/share/licenses/pol
	$(CP) $(POL_SOURCE_DIR)/LICENSE $(POL_SOURCE_DIR)/NOTICE $(1)/usr/share/licenses/pol/
endef

define Package/pol/postinst
#!/bin/sh
[ -n "$$IPKG_INSTROOT" ] || { /etc/init.d/rpcd restart; rm -f /tmp/luci-indexcache; if /usr/libexec/rpcd/pol is-enabled; then /etc/init.d/pol start; fi; }
exit 0
endef

define Package/pol/prerm
#!/bin/sh
[ -n "$$IPKG_INSTROOT" ] || /etc/init.d/pol stop
exit 0
endef

$(eval $(call BuildPackage,pol))
