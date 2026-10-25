# Changelog

## [Unreleased]

- Them `ascii_fast_path` (mac dinh tat) trong `config.yaml`: chuyen tiep phim ASCII thuan ngay lap tuc, khong giu preedit, giup Enter/Tab an prediction tren Chrome Omnibox trong 1 lan bam; tu dong chuyen sang preedit + backspace-correct khi xuat hien dau tieng Viet.

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
