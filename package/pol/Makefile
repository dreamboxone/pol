include $(TOPDIR)/rules.mk
POL_SOURCE_DIR?=$(abspath $(CURDIR)/../..)

PKG_NAME:=pol
PKG_VERSION:=1.3.0
PKG_RELEASE:=1
POL_VIEW_VERSION:=$(subst .,_,$(PKG_VERSION))
PKG_LICENSE:=GPL-3.0-only AGPL-3.0-or-later
PKG_BUILD_DIR:=$(BUILD_DIR)/$(PKG_NAME)

include $(INCLUDE_DIR)/package.mk

define Package/pol
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Pol VPN for OpenWrt
  CONFLICTS:=pol-core
  DEPENDS:=+luci-base +rpcd +ca-bundle +ip-full +nftables +kmod-nft-tproxy
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
	mv $(1)/www/luci-static/resources/view/pol/main.js $(1)/www/luci-static/resources/view/pol/main-$(POL_VIEW_VERSION).js
	$(SED) 's@pol/main"@pol/main-$(POL_VIEW_VERSION)"@' $(1)/usr/share/luci/menu.d/luci-app-pol.json
	$(INSTALL_DIR) $(1)/usr/libexec/rpcd
	$(INSTALL_BIN) $(POL_SOURCE_DIR)/build/bin/$(ARCH)/pol $(1)/usr/libexec/rpcd/pol
	$(INSTALL_DIR) $(1)/usr/lib/pol
	$(INSTALL_BIN) $(POL_SOURCE_DIR)/prebuilt/$(ARCH)/xray $(1)/usr/lib/pol/xray
	$(INSTALL_DIR) $(1)/usr/share/licenses/pol
	$(CP) $(POL_SOURCE_DIR)/LICENSE $(POL_SOURCE_DIR)/NOTICE $(POL_SOURCE_DIR)/licenses/Zirgozar-AGPL-3.0.txt $(POL_SOURCE_DIR)/licenses/Xray-MPL-2.0.txt $(1)/usr/share/licenses/pol/
endef

define Package/pol/postinst
#!/bin/sh
[ -n "$$IPKG_INSTROOT" ] || { /etc/init.d/rpcd restart; rm -f /tmp/luci-indexcache /tmp/luci-indexcache.*.json; if /usr/libexec/rpcd/pol is-enabled; then /etc/init.d/pol start; fi; }
exit 0
endef

define Package/pol/prerm
#!/bin/sh
[ -n "$$IPKG_INSTROOT" ] || /etc/init.d/pol stop
exit 0
endef

$(eval $(call BuildPackage,pol))
