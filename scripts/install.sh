#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

echo "=== Cài đặt NepKey IBus ==="

# 1. Biên dịch ứng dụng IBus
echo "--> Đang biên dịch NepKey IBus..."
go build -ldflags="-s -w" -o nepkey-ibus ./cmd/nepkey-ibus

# 2. Cài đặt binary vào ~/.local/bin
mkdir -p "$HOME/.local/bin"
cp -f nepkey-ibus "$HOME/.local/bin/nepkey-ibus"
chmod +x "$HOME/.local/bin/nepkey-ibus"
rm -f nepkey-ibus

cp -f cmd/nepkey-settings/main.py "$HOME/.local/bin/nepkey-settings"
chmod +x "$HOME/.local/bin/nepkey-settings"
echo "✓ Đã cài đặt binary vào $HOME/.local/bin/"

# 3. Tạo thư mục cấu hình ~/.config/nepkey
mkdir -p "$HOME/.config/nepkey"
if [ ! -f "$HOME/.config/nepkey/config.yaml" ]; then
    cp configs/config.yaml "$HOME/.config/nepkey/config.yaml"
    echo "✓ Đã khởi tạo cấu hình tại $HOME/.config/nepkey/config.yaml"
fi
if [ ! -f "$HOME/.config/nepkey/macros.yaml" ]; then
    cp configs/macros.yaml "$HOME/.config/nepkey/macros.yaml"
    echo "✓ Đã khởi tạo bảng gõ tắt tại $HOME/.config/nepkey/macros.yaml"
fi

# 4. Cài đặt biểu tượng Vector SVG
mkdir -p "$HOME/.local/share/icons/nepkey"
cp assets/icons/*.svg "$HOME/.local/share/icons/nepkey/"
echo "✓ Đã cài đặt icons vào $HOME/.local/share/icons/nepkey/"

# 5. Cài đặt menu desktop
mkdir -p "$HOME/.local/share/applications"
cp assets/nepkey-settings.desktop "$HOME/.local/share/applications/"
echo "✓ Đã cài đặt launcher vào $HOME/.local/share/applications/"

# 6. Đăng ký IBus component XML
IBUS_COMP_DIR="$HOME/.local/share/ibus/component"
mkdir -p "$IBUS_COMP_DIR"
sed -e "s|/usr/bin/nepkey-ibus|$HOME/.local/bin/nepkey-ibus|g" \
    -e "s|/usr/bin/nepkey-settings|$HOME/.local/bin/nepkey-settings|g" \
    -e "s|/usr/share/icons/nepkey|$HOME/.local/share/icons/nepkey|g" \
    assets/nepkey.xml > "$IBUS_COMP_DIR/nepkey.xml"
echo "✓ Đã cài đặt IBus component: $IBUS_COMP_DIR/nepkey.xml"

# 7. Cập nhật bộ nhớ đệm IBus
echo "--> Đang cập nhật bộ nhớ đệm IBus..."
if command -v ibus > /dev/null 2>&1; then
    ibus write-cache 2>/dev/null || true
    ibus restart 2>/dev/null || true
fi

echo ""
echo "🎉 Cài đặt hoàn tất!"
echo "Cách sử dụng:"
echo "1. Vào Cài đặt Ubuntu (Settings) -> Bàn phím (Keyboard) -> Nguồn nhập (Input Sources)"
echo "2. Bấm dấu [+] -> Chọn Tiếng Việt -> Thêm 'NepKey'"
echo "3. Bấm Super + Space để chuyển đổi gõ tiếng Việt mượt mà!"
