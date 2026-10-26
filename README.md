# NepKey IBus - Bộ gõ tiếng Việt cho Linux

> **NepKey IBus** is a modern, responsive Vietnamese Input Method Engine for Linux built on top of the **Bamboo Engine** and integrated directly into the **IBus subsystem**.
>
> **NepKey IBus** là bộ gõ tiếng Việt thế hệ mới cho Linux, phát triển dựa trên lõi **Bamboo Engine**, tích hợp thẳng vào hệ thống **IBus** của Ubuntu/GNOME. Được tối ưu cho lập trình viên, terminal, chat AI và duyệt web mượt mà trên cả Wayland lẫn X11.

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.27-blue.svg)](go.mod)
[![Platform](https://img.shields.io/badge/Platform-Ubuntu%20GNOME%20Wayland%2FX11-lightgrey.svg)](scripts/install.sh)

> **Cam kết tương thích:** gõ tiếng Việt tự nhiên như bàn phím thường trên trình duyệt (Chrome/Firefox Omnibox, Chat AI), mọi app (VS Code, Slack, Discord, Electron) và terminal (Wayland/X11) — phím `/` và lệnh gạch chéo chạy chuẩn, không nuốt ký tự, không kẹt gợi ý.

---

## Vì sao khác biệt so với ibus-bamboo / ibus-unikey

| Tình huống | ibus-bamboo / unikey cũ | NepKey |
|---|---|---|
| Chat AI / Slack / Discord / VS Code: gõ `/image`, `/task`, `/clear` | Hay bị nuốt ký tự `/`, kẹt suggestion tiếng Việt, Enter phải bấm 2 lần | Nhận diện raw token, passthrough, Enter 1 lần ăn ngay |
| Thanh địa chỉ Chrome / Firefox, terminal: gõ `https://...`, `/home/user/file.go` | Dấu `.` `/` `-` bị hiểu nhầm thành dấu tiếng Việt, đứt URL | Tách cấu trúc URL/path trước khi xử lý dấu, giữ nguyên domain và path |
| Điều hướng con trỏ `Left/Right/Home/End/PageUp/PageDown` | Dễ lặp chữ, kẹt buffer trên Electron/Wayland | Auto-commit buffer khi điều hướng, triệt tiêu lặp chữ |
| Electron (VS Code, Slack, Discord), Wayland native | Trễ, mất chữ, focus-out duplicate | Clipboard injection + xử lý `input-purpose` / password field, test trên Wayland |
| Gõ tắt / macro | Cấu hình tĩnh, phải khởi động lại | Nạp động từ YAML, sửa là áp dụng ngay, không restart |

Cơ chế chính: `pkg/ibus/engine.go` — `rawTokenMode` / `slashCommandMode` + `isRawTokenContinuation()`, bypass khi `inputPurpose == password/pin`, auto-commit khi gặp navigation keys.

---

## Các cải tiến thực chiến

### 1. Tối ưu Chat AI & Terminal (Slash Commands)
- Gõ lệnh `/` tự nhiên: hỗ trợ nhập các lệnh gạch chéo (`/image`, `/task`, `/clear`, `/help`) trên **Claude**, **ChatGPT**, **Slack**, **Discord**, **Notion**, **VS Code** mà không bị nuốt ký tự hay kẹt bảng gợi ý tiếng Việt.
- Phản hồi Enter 0ms: phím `Enter` và `Tab` gửi tin nhắn hoặc thực thi lệnh ngay lập tức chỉ với 1 lần bấm.

### 2. Nhập URL & Link Web chuẩn xác
- Tự động nhận diện và phân tách các ký tự cấu trúc URL (`http://`, `https://`, dấu chấm `.`, gạch chéo `/`, gạch ngang `-`) trên thanh địa chỉ trình duyệt (Chrome, Firefox) và terminal.
- Không bị bộ gõ hiểu nhầm thành nguyên âm / phụ âm tiếng Việt khi gõ tên miền hay đường dẫn file.

### 3. Điều hướng con trỏ thông minh (Smart Navigation)
- Tự động chốt buffer khi dùng `Left/Right`, `Home`, `End`, `Page Up`, `Page Down`.
- Khắc phục lỗi kinh niên lặp chữ và kẹt con trỏ trên Electron và Wayland.

### 4. Lõi Bamboo Core & Gõ tắt Macro
- Hỗ trợ đầy đủ: **Telex 2**, **Telex**, **VNI**, **VIQR**.
- Chuẩn Unicode dựng sẵn, bỏ dấu tự do, đặt dấu kiểu mới (`òa`, `òe`, `ùy`) và tự động kiểm tra chính tả.
- Gõ tắt (Macro) nạp động từ YAML, chỉnh sửa và áp dụng tức thì không cần khởi động lại.

### 5. Giao diện cấu hình GTK4 & Libadwaita
- Giao diện cài đặt hiện đại, đồng bộ Dark/Light mode với GNOME.

---

## Môi trường đã test

- Ubuntu 24.04 GNOME Wayland + X11, IBus 1.5.x
- App: GNOME Terminal, VS Code (Electron), Chrome/Firefox address bar, Slack/Discord, Claude/ChatGPT web
- Go 1.27.1, `go test ./...` pass (xem `pkg/engine/*_test.go`, `pkg/ibus/*_test.go`)

## Chế độ ASCII nhanh (thử nghiệm)

`ascii_fast_path: true` trong `~/.config/nepkey/config.yaml` (mặc định `false`): chữ ASCII thuần được chuyển thẳng tới app, không giữ preedit, nên Enter/Tab ăn prediction trên thanh địa chỉ Chrome trong 1 lần bấm. Khi xuất hiện dấu tiếng Việt, engine tự xóa phần đã chuyển và vẽ lại preedit như cũ. Chỉ bật với app báo surrounding-text (Chrome, ô nhập GTK); terminal và app không báo dùng đường preedit cũ nên hành vi giữ nguyên. App báo surrounding-text nhưng bỏ qua lệnh xóa (Zalo bản X11, app Wine) được tự động tắt fast-path theo WM_CLASS trong `fast_path_blocklist` (mặc định `zalo`, `wine`). Khi cần gỡ lỗi, bật `debug_log: true` để ghi `/tmp/nepkey-debug.log`.

## Known limitations

- `output_method: clipboard` là mặc định khuyến nghị cho Chrome/Zalo/Terminal. Chế độ `hex` (Ctrl+Shift+U) chỉ phù hợp GTK thuần.
- Chưa hỗ trợ KDE/Fcitx5, mới tập trung IBus/GNOME.
- Macro hiện là map 1:1 đơn giản trong `configs/macros.yaml`, chưa có macro nhiều con trỏ.

---

## Cấu trúc dự án

```text
nepkey/
├── assets/
│   ├── icons/                  # Biểu tượng Vector SVG (nepkey, nepkey-v, nepkey-e)
│   ├── nepkey-settings.desktop # Menu ứng dụng cài đặt desktop
│   └── nepkey.xml              # Định nghĩa IBus Component XML
├── cmd/
│   ├── nepkey-ibus/            # Engine bộ gõ IBus chính (main.go)
│   └── nepkey-settings/        # Giao diện cấu hình GTK4 + Libadwaita (main.py)
├── pkg/
│   ├── config/                 # Quản lý cấu hình YAML & bảng từ gõ tắt macros
│   ├── engine/                 # Lõi Bamboo state machine & xử lý từ vựng
│   └── ibus/                   # Server IBus daemon, xử lý phím tắt & DBus protocol
├── configs/
│   ├── config.yaml             # Cấu hình kiểu gõ mặc định
│   └── macros.yaml             # Danh sách từ gõ tắt mặc định
├── scripts/
│   ├── install.sh              # Script biên dịch và cài đặt tự động vào máy
│   └── build-deb.sh            # Script đóng gói file cài đặt Ubuntu/Debian (.deb)
├── .github/workflows/          # CI: vet + test + build
├── CONTRIBUTING.md
├── CHANGELOG.md
├── TRADEMARK.md
├── LICENSE                     # GNU General Public License v3.0
└── README.md
```

---

## Hướng dẫn cài đặt nhanh

### Cách 0: Một dòng lệnh (Ubuntu/Debian, khuyên dùng)

```bash
curl -fsSL https://raw.githubusercontent.com/todo-laterx/nepkey/main/install.sh | bash
```

Tự tải `.deb` mới nhất từ Releases và cài đặt. Trang giới thiệu: https://todo-laterx.github.io/nepkey/

### Cách 1: Cài đặt tự động từ mã nguồn (khuyên dùng, không cần sudo)

```bash
git clone https://github.com/todo-laterx/nepkey.git
cd nepkey
bash scripts/install.sh
```

Sau khi chạy xong:
1. Mở **Settings** -> **Keyboard** -> **Input Sources**.
2. Bấm dấu **`+`** -> Chọn **Vietnamese** -> Thêm **NepKey**.
3. Dùng tổ hợp phím **`Super + Space`** để chuyển đổi và bắt đầu gõ tiếng Việt.

### Cách 2: Cài đặt gói `.deb`

```bash
bash scripts/build-deb.sh
sudo dpkg -i dist/nepkey_1.3.0_amd64.deb
```

File `.deb` release cũng được đính kèm trong GitHub Releases trong GitHub Releases (bản mới nhất).

---

## Phím tắt & Thao tác

- **`Ctrl+Space`** (tùy chỉnh `Alt+Shift` / `Ctrl+Shift` qua `toggle_key`): chuyển giữa chế độ gõ Việt (top bar hiện V) và chế độ tiếng Anh (top bar hiện E). Chế độ tiếng Anh cho mọi phím đi thẳng, không xử lý dấu.
- **`Super + Space`**: chuyển nguồn nhập của GNOME (NepKey Việt / NepKey Anh / English).

- **`Super + Space`**: Chuyển đổi giữa bàn phím tiếng Anh và tiếng Việt qua hệ thống IBus.
- **`nepkey-settings`**: Mở cửa sổ cấu hình giao diện GTK4 / Libadwaita.

---

## Đóng góp

Xem [CONTRIBUTING.md](CONTRIBUTING.md). Mọi MR đều chạy CI `go vet + go test + go build`. Fork thoải mái theo GPL-3.0.

---

## Giấy phép

Dự án phát hành theo **GNU General Public License v3.0 (GPL-3.0)**. Xem [LICENSE](LICENSE).

- Được clone/fork/sửa/phân phối tự do theo GPL-3.0, fork phân phối lại phải giữ GPL-3.0 và công bố source.
- Lõi xử lý tiếng Việt kế thừa `bamboo-core (MIT)` của BambooEngine — xem `go.mod`, giữ attribution trong mọi bản phân phối.
- Tên và logo `NepKey` là thương hiệu riêng, không đi kèm license code. Xem [TRADEMARK.md](TRADEMARK.md): fork phân phối lại đổi tên để tránh nhầm lẫn (ví dụ `MyIME (based on NepKey)`).
