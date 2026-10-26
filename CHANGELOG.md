# Changelog

## [1.3.0] - 2026-10-26

- Sua loi nhan doi tien to trong Zalo: app bao surrounding-text nhung bo qua lenh xoa. Fast-path tu tat theo WM_CLASS X11 qua `fast_path_blocklist` (mac dinh `zalo`, `wine`); cac app khac giu nguyen.
- Them `debug_log` (mac dinh tat) ghi `/tmp/nepkey-debug.log` de chan doan theo tung app.
- Cai 1 dong lenh `curl ... install.sh | bash`, tu tai .deb moi nhat tu Releases.
- Trang gioi thieu: https://todo-laterx.github.io/nepkey/ (bo cuc bat doi xung, anh chup that, trang thai distro trung thuc, dark/light).
- Settings gon hon: bo Bang ma va Co che xuat khoi UI, van giu trong config.
- 42 test xanh, `go vet` sach.

## [1.2.0] - 2026-10-26

- Phim tat chuyen V/E (mac dinh Ctrl+Space, tuy chinh `toggle_key`): doi engine sang NepKey (Tieng Anh), top bar hien E, bam lai ve V.
- Dang ky 2 engine `nepkey` + `nepkey-en` voi icon rieng.
- Version mot nguon qua file `VERSION`; build, install, GUI va engine doc cung mot gia tri.
- Settings: sua hien thi `&`, gon trang Gioi thieu, them cong tac ASCII nhanh.
- Trong tam: tuong thich go tu nhien tren trinh duyet, app, terminal; phim `/` chay chuan.

## [1.1.0] - 2026-10-26

- Them `ascii_fast_path` (mac dinh tat): chu ASCII thuan forward ngay, Enter/Tab an prediction Omnibox 1 lan bam; chuyen sang preedit va xoa vung khi co dau.
- Khoa che do theo tung tu, chi fast-path khi client bao surrounding-text (terminal giu duong preedit cu).
- Sua huy dau Telex tu phuc hoi tien to (`terminal` khong con ra `rminal`), sua commit nhan doi o nhanh so va Escape (`gg6`), keycode BackSpace XKB (22).
- Test xanh, `go vet` sach.

## [1.0.0] - 2026-10-25

- Engine IBus Go + Bamboo core: Telex 2, Telex, VNI, VIQR; bo dau tu do; dat dau kieu moi; macro YAML.
- Raw token passthrough cho slash command, URL, path, CLI flag; smart commit khi dieu huong; bypass password/PIN.
- Settings GTK4/Libadwaita, script cai dat va dong goi .deb.
- Repo public: GPL-3.0 + TRADEMARK.md, bang so sanh, CONTRIBUTING, CI vet + test + build.
