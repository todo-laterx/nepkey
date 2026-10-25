#!/usr/bin/env bash
# NepKey one-line installer (Ubuntu/Debian) - mirror of Ngó Sen style:
#   curl -fsSL https://raw.githubusercontent.com/todo-laterx/nepkey/main/install.sh | bash
set -e

REPO="todo-laterx/nepkey"

echo "=== NepKey IBus - one-line installer (Ubuntu/Debian) ==="

if ! command -v curl >/dev/null 2>&1; then
    echo "--> Installing curl..."
    sudo apt-get update -qq
    sudo apt-get install -y -qq curl ca-certificates
fi

echo "--> Finding latest release..."
TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -n 1 | cut -d'"' -f4)"
if [ -z "$TAG" ]; then
    echo "ERROR: cannot determine latest release. Check your network." >&2
    exit 1
fi
DEB="nepkey_${TAG#v}_amd64.deb"
URL="https://github.com/${REPO}/releases/download/${TAG}/${DEB}"
echo "--> Downloading ${DEB} (${TAG})..."

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
curl -fsSL -o "$TMPDIR/$DEB" "$URL"

echo "--> Installing (needs sudo)..."
sudo dpkg -i "$TMPDIR/$DEB" || sudo apt-get install -f -y -qq

echo ""
echo "Done! NepKey ${TAG} installed."
echo "Next steps:"
echo "  1. Settings -> Keyboard -> Input Sources -> + -> Vietnamese -> add 'NepKey' (+ 'NepKey (Tiếng Anh)' for V/E toggle)"
echo "  2. Press Super + Space to switch input, Ctrl+Space to toggle V/E in place"
echo "  3. Optional power knobs in ~/.config/nepkey/config.yaml (ascii_fast_path, toggle_key)"
