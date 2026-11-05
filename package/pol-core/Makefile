include $(TOPDIR)/rules.mk
POL_SOURCE_DIR?=$(abspath $(CURDIR)/../..)

PKG_NAME:=pol-core
PKG_VERSION:=26.10.21
PKG_RELEASE:=1
PKG_LICENSE:=MPL-2.0
include $(INCLUDE_DIR)/package.mk

define Package/pol-core
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Xray core for Pol
  DEPENDS:=+ca-bundle
endef

define Build/Prepare
endef
define Build/Compile
endef

define Package/pol-core/install
	$(INSTALL_DIR) $(1)/usr/lib/pol
	$(INSTALL_BIN) $(POL_SOURCE_DIR)/prebuilt/$(ARCH)/xray $(1)/usr/lib/pol/xray
	ln -sf xray $(1)/usr/lib/pol/pol-core
	$(INSTALL_DIR) $(1)/usr/share/licenses/pol-core
	$(CP) $(POL_SOURCE_DIR)/licenses/Xray-MPL-2.0.txt $(1)/usr/share/licenses/pol-core/LICENSE
endef

$(eval $(call BuildPackage,pol-core))
