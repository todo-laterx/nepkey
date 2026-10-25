# Changelog

## [Unreleased]

- Settings: rut gon tab Kieu go (bo Bang ma, Co che xuat khoi UI, giu trong file cau hinh); sua mac dinh output_method khop engine.

## [1.2.0] - 2026-10-26

Trong tam: tuong thich go tu nhien tren trinh duyet / app / terminal, phim `/` chay chuan.

- Phim tat chuyen V/E cua NepKey (mac dinh Ctrl+Space, tuy chinh Alt+Shift / Ctrl+Shift trong `toggle_key`): doi engine sang NepKey (Tieng Anh) de thanh top-bar hien E, bam lai ve V.
- Dang ky 2 engine `nepkey` (V) + `nepkey-en` (E) voi icon nepkey-v.svg / nepkey-e.svg rieng.
- Settings: sua hien thi `&`, gon trang Gioi thieu, them cong tac ASCII nhanh.

## [1.1.0] - 2026-10-26

- Them `ascii_fast_path` (mac dinh tat) trong `config.yaml`: chuyen tiep phim ASCII thuan ngay lap tuc, khong giu preedit, giup Enter/Tab an prediction tren Chrome Omnibox trong 1 lan bam; tu dong chuyen sang preedit + xoa vung (surrounding-text, fallback BackSpace) khi xuat hien dau tieng Viet.
- Khoa che do theo tung tu (chiu duoc capability den muon sau focus tren Electron), chi fast-path khi client bao surrounding-text (terminal VTE giu duong preedit cu).
- Sua huy dau Telex tu phuc hoi tien to da xoa ("terminal" khong con ra "rminal"), sua commit nhan doi o nhanh so va Escape ("gg6").
- 25 test xanh (17 ibus + 8 engine), `go vet` sach.

- Them `ascii_fast_path` (mac dinh tat) trong `config.yaml`: chuyen tiep phim ASCII thuan ngay lap tuc, khong giu preedit, giup Enter/Tab an prediction tren Chrome Omnibox trong 1 lan bam; tu dong chuyen sang preedit + backspace-correct khi xuat hien dau tieng Viet. Backspace bu dung keycode XKB (22) hoc tu phim that thay vi evdev (14) de client khong drop event.

- Tach `TRADEMARK.md` rieng khoi `LICENSE`, giu GPL-3.0 cho code.
- Bo sung bang so sanh vs ibus-bamboo, moi truong test, known limitations.
- Them CI `vet + test + build`, `CONTRIBUTING.md`.

## [1.0.0] - 2026-10-25

- Engine IBus Go + Bamboo core: Telex 2 / Telex / VNI / VIQR, free tone marking, modern tone style.
- Raw token passthrough cho slash command, URL, path, CLI flag (`rawTokenMode`).
- Smart commit khi navigation keys, fix lap chu Electron/Wayland.
- Bypass password/pin field theo `input-purpose`.
- Settings GTK4/Libadwaita, macro YAML hot-reload.
- Script `install.sh` (user-local, khong sudo) va `build-deb.sh` (.deb Ubuntu/Debian).
