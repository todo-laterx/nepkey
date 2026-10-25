# NepKey IBus - Bộ gõ tiếng Việt cho Linux

> **NepKey IBus** is a modern, responsive Vietnamese Input Method Engine for Linux built on top of the **Bamboo Engine** and integrated directly into the **IBus subsystem**.
>
> **NepKey IBus** là bộ gõ tiếng Việt thế hệ mới cho Linux, phát triển dựa trên lõi **Bamboo Engine**, tích hợp thẳng vào hệ thống **IBus** của Ubuntu/GNOME. Được tối ưu cho lập trình viên, terminal, chat AI và duyệt web mượt mà trên cả Wayland lẫn X11.

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.27-blue.svg)](go.mod)
[![Platform](https://img.shields.io/badge/Platform-Ubuntu%20GNOME%20Wayland%2FX11-lightgrey.svg)](scripts/install.sh)

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
sudo dpkg -i dist/nepkey_1.0.0_amd64.deb
```

File `.deb` release cũng được đính kèm trong GitHub Releases từ tag `v1.0.0` trở đi.

---

## Phím tắt & Thao tác

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
