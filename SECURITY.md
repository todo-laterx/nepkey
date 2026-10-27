# Security Policy — NepKey

## Phạm vi

NepKey là bộ gõ IBus chạy local, offline 100%. Không gửi keystroke, không telemetry, không network call (trừ `install.sh` tải `.deb` từ GitHub Releases khi bạn tự chạy).

## Cam kết

- Không log keystroke theo mặc định. `debug_log: false` mặc định.
- Khi bật `debug_log: true` (hoặc `NEPKEY_DEBUG=1`), chỉ ghi ra `/tmp/nepkey-debug.log` local: keyval/keycode/caps/purpose, không ghi toàn bộ câu, không upload.
- Tự bypass (passthrough, không xử lý dấu) khi `inputPurpose == password/pin` hoặc `hidden-text`, và commit buffer ngay khi gặp `Ctrl/Alt/Super`.
- Không đọc file ngoài `~/.config/nepkey/config.yaml`, `macros.yaml`, WM_CLASS qua `xprop` (chỉ để blocklist fast-path).

## Báo cáo lỗ hổng

Không dùng Issues công khai cho lỗ hổng keylogger/rò rỉ phím. Mở GitHub Security Advisory hoặc liên hệ maintainer qua profile `https://github.com/todo-laterx`. Ghi rõ: Ubuntu version, Wayland/X11, app bị ảnh hưởng, cách tái hiện, log `journalctl --user -u ibus*` (che thông tin nhạy cảm).

## Cập nhật

Fix bảo mật được release patch version + ghi `CHANGELOG.md`, khuyến nghị `sudo dpkg -i nepkey_<ver>_amd64.deb` mới nhất từ Releases.
