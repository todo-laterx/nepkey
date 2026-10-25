#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

VERSION="$(tr -d '[:space:]' < "$DIR/VERSION")"
ARCH="amd64"
PKG_DIR="build/deb"
DEB_NAME="nepkey_${VERSION}_${ARCH}.deb"

echo "=== 📦 Đang đóng gói NepKey IBus thành file ${DEB_NAME} ==="

# 1. Dọn dẹp thư mục build cũ
rm -rf "$PKG_DIR" "$DEB_NAME"
mkdir -p "$PKG_DIR/DEBIAN"
mkdir -p "$PKG_DIR/usr/bin"
mkdir -p "$PKG_DIR/usr/share/applications"
mkdir -p "$PKG_DIR/usr/share/icons/nepkey"
mkdir -p "$PKG_DIR/usr/share/icons/hicolor/scalable/apps"
mkdir -p "$PKG_DIR/etc/nepkey"
mkdir -p "$PKG_DIR/usr/share/ibus/component"

# 2. Biên dịch nhị phân IBus
echo "Compiling optimized binary..."
go build -ldflags="-s -w -X nepkey/pkg/ibus.appVersionStr=${VERSION}" -o "$PKG_DIR/usr/bin/nepkey-ibus" ./cmd/nepkey-ibus
chmod 755 "$PKG_DIR/usr/bin/nepkey-ibus"

# Cài đặt IBus Component XML
sed -e "s|<version>[^<]*</version>|<version>${VERSION}</version>|g" assets/nepkey.xml > "$PKG_DIR/usr/share/ibus/component/nepkey.xml"
chmod 644 "$PKG_DIR/usr/share/ibus/component/nepkey.xml"

# Cài đặt giao diện cài đặt NepKey Settings (GTK4 + Libadwaita)
sed -e "s|__NEPKEY_VERSION__|${VERSION}|g" cmd/nepkey-settings/main.py > "$PKG_DIR/usr/bin/nepkey-settings"
chmod 755 "$PKG_DIR/usr/bin/nepkey-settings"

# 3. Sao chép file cấu hình mẫu & macros
cp configs/config.yaml "$PKG_DIR/etc/nepkey/config.yaml"
cp configs/macros.yaml "$PKG_DIR/etc/nepkey/macros.yaml"
chmod 644 "$PKG_DIR"/etc/nepkey/*.yaml

# 4. Sao chép biểu tượng Vector SVG (nepkey, nepkey-v, nepkey-e)
cp assets/icons/*.svg "$PKG_DIR/usr/share/icons/nepkey/"
cp assets/icons/*.svg "$PKG_DIR/usr/share/icons/hicolor/scalable/apps/"
chmod 644 "$PKG_DIR"/usr/share/icons/nepkey/*.svg
chmod 644 "$PKG_DIR"/usr/share/icons/hicolor/scalable/apps/*.svg

# 5. Cài đặt file desktop launcher
cp assets/nepkey-settings.desktop "$PKG_DIR/usr/share/applications/nepkey-settings.desktop"
chmod 644 "$PKG_DIR/usr/share/applications/nepkey-settings.desktop"

# 6. Tạo file DEBIAN/control
cat << EOF > "$PKG_DIR/DEBIAN/control"
Package: nepkey
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: NepKey Contributors <https://github.com/todo-laterx/nepkey>
Depends: python3, python3-gi, gir1.2-gtk-4.0, gir1.2-adw-1, python3-yaml, ibus
Description: Native Vietnamese Input Method Engine (IBus) for Linux
 NepKey IBus is a lightning-fast native Vietnamese input method for Linux
 integrated directly into the IBus subsystem. Optimized for developers,
 terminals, AI chats (Claude, ChatGPT), and browser web links.
 Includes a modern GTK4/Libadwaita settings app, modern tone styling,
 free tone marking, and dynamic macro expansion.
EOF

# 7. Tạo script hậu cài đặt DEBIAN/postinst
cat << 'EOF' > "$PKG_DIR/DEBIAN/postinst"
#!/bin/sh
set -e

# Cập nhật cache ứng dụng và biểu tượng
if command -v update-desktop-database > /dev/null 2>&1; then
    update-desktop-database -q /usr/share/applications || true
fi
if command -v gtk-update-icon-cache > /dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi

# Cập nhật cache IBus hệ thống
if command -v ibus > /dev/null 2>&1; then
    ibus write-cache --system 2>/dev/null || true
fi

echo "✓ NepKey IBus đã được cài đặt thành công vào hệ thống!"
echo "  - Đã đăng ký component vào: /usr/share/ibus/component/nepkey.xml"
echo "  - Hãy thêm 'NepKey' trong Cài đặt bàn phím Ubuntu (Settings -> Keyboard -> Input Sources)"
echo "  - Nhấn Super + Space để chuyển đổi."
exit 0
EOF
chmod 755 "$PKG_DIR/DEBIAN/postinst"

# 8. Đóng gói với dpkg-deb
dpkg-deb --build --root-owner-group "$PKG_DIR" "$DEB_NAME"

# Di chuyển deb vào dist/
mkdir -p dist
mv -f "$DEB_NAME" dist/

echo ""
echo "🎉 Đã đóng gói thành công: dist/$DEB_NAME"
