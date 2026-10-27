# Changelog

## [1.4.0] - 2026-10-27

- Bo hoan toan `ascii_fast_path` va `fast_path_blocklist`: forward-then-erase khong the dung tren client bo qua lenh xoa (Chrome/Docs/Zalo), gay dup chu. Go luon phan code WM_CLASS/X11 va verify surrounding.
- Enter 1 lan bam khong can fast-path: khi dang go do, engine commit chu roi bom Enter that qua `/dev/uinput` sau ~45ms, nen dropdown/goi y (Omnibox, chat) duoc chap nhan ngay. Enter goc bi nuot (khong submit 2 lan), Shift+Enter giu nguyen, co cooldown 300ms chong lap; thieu quyen uinput thi tu ve hanh vi IBus cu.
- Settings: bo cong tac "Chu ASCII chuyen tiep nhanh"; `configs/config.yaml` bo 2 khoa da xoa va ghi chu `uinput_path`.
- 22 test xanh (them 5 test cho Enter injection), `go vet` sach.

## [1.3.3] - 2026-10-27

- Sua dup chu tren Chrome/Docs khi bat `ascii_fast_path`: Chrome/Docs bat co surrounding-text nhung lo lenh xoa. Co che xoa fast-path chuyen han sang phim BackSpace gia lap (key event app nao cung hieu), bo `DeleteSurroundingText`. Cua kiem tra surrounding-proof giu nguyen de terminal o lai preedit cu.
- Test hoi quy `CorrectionUsesBackspaceEvenWithSurrounding`; 46 test xanh, `go vet` sach.

## [1.3.2] - 2026-10-27

- Fix `go.mod` module path `nepkey` -> `github.com/todo-laterx/nepkey` de `go install`/`go get` tu ngoai hoat dong; dong bo import + ldflags `appVersionStr`.
- `configs/config.yaml` mau uncomment `ascii_fast_path/debug_log/fast_path_blocklist` mac dinh (hanh vi khong doi, de user nhin thay).
- `CONTRIBUTING.md`: xoa dong copy-paste DxDataGrid, thay bang quy tac dong bo config 3 noi.
- Them `SECURITY.md`: offline 100%, khong log keystroke mac dinh, bypass password/PIN, bao cao lo hong qua Advisory.
- `scripts/install.sh`: check thieu `python3-gi`/`python3-yaml`, phan biet ro source installer vs `/install.sh` (.deb).

## [1.3.1] - 2026-10-27

- Sua nhay chu tren terminal khi bat `ascii_fast_path`: VTE/Ptyxis quang cao surrounding-text nhung khong day surrounding va bo qua lenh xoa. Fast-path gio doi ca bit capability lan bang chung day surrounding-text (`RequireSurroundingText` -> `SetSurroundingText`); terminal khong bao gio day nen luon giu preedit cu, ke ca Wayland. Tu dau sau focus giu an toan cho toi khi app chung minh ho tro.
- Mo rong `fast_path_blocklist` mac dinh voi nhom terminal X11 (gnome-terminal, ptyxis, alacritty, kitty, ...).
- 4 test hoi quy moi (caps nhung khong push, tone trong preedit, FocusIn reset, blocklist terminal); 46 test xanh, `go vet` sach.

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
