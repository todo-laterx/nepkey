# 🌾 NepKey IBus - Vietnamese Input Method Engine for Linux

> **NepKey IBus** is a modern, responsive Vietnamese Input Method Engine for Linux built on top of the **Bamboo Engine** and integrated directly into the Linux desktop's **IBus subsystem**.
>
> **NepKey IBus** là bộ gõ tiếng Việt thế hệ mới cho Linux phát triển dựa trên lõi **Bamboo Engine**, tích hợp nguyên bản vào hệ thống **IBus** của Ubuntu/GNOME. Được thiết kế tối ưu chuyên biệt cho lập trình viên, dòng lệnh terminal, chat AI và duyệt web mượt mà trên cả Wayland lẫn X11.

---

## 🚀 Các cải tiến thực chiến vượt trội

### 1. Tối ưu Chat AI & Terminal (Slash Commands)
- **Gõ lệnh `/` tự nhiên**: Hỗ trợ nhập các lệnh gạch chéo (`/image`, `/task`, `/clear`, `/help`) trên **Claude**, **ChatGPT**, **Slack**, **Discord**, **Notion**, **VS Code** mà không bị nuốt ký tự hay kẹt bảng gợi ý tiếng Việt.
- **Phản hồi Enter 0ms**: Phím `Enter` và `Tab` gửi tin nhắn hoặc thực thi lệnh ngay lập tức chỉ với 1 lần bấm, không bị trễ nhịp hay phải gõ 2 lần như các bộ gõ cũ.

### 2. Nhập URL & Link Web chuẩn xác
- **Không biến dạng đường dẫn**: Tự động nhận diện và phân tách các ký tự cấu trúc URL (`http://`, `https://`, dấu chấm `.`, gạch chéo `/`, gạch ngang `-`) trên thanh địa chỉ trình duyệt (Chrome, Firefox) và terminal.
- Không bị bộ gõ hiểu nhầm thành các nguyên âm/phụ âm tiếng Việt khi gõ tên miền hay đường dẫn file.

### 3. Điều hướng con trỏ thông minh (Smart Navigation)
- **Tự động chốt buffer**: Khi sử dụng các phím mũi tên `←` `→`, `Home`, `End`, `Page Up`, `Page Down`, bộ gõ tự động commit từ đang gõ tức thì.
- **Triệt tiêu lỗi lặp chữ**: Khắc phục triệt để lỗi kinh niên `chayạy`, `loôiỗi` và kẹt con trỏ trên các ứng dụng Electron và Wayland.

### 4. Lõi xử lý Bamboo Core & Gõ tắt Macro
- Hỗ trợ đầy đủ các kiểu gõ phổ biến: **Telex 2**, **Telex**, **VNI**, **VIQR**.
- Hỗ trợ chuẩn Unicode dựng sẵn, bỏ dấu tự do, đặt dấu kiểu mới (`òa`, `òe`, `ùy`) và tự động kiểm tra chính tả.
- Tính năng gõ tắt (Macro) nạp động từ file YAML, chỉnh sửa và áp dụng tức thì không cần khởi động lại.

### 5. Giao diện cấu hình GTK4 & Libadwaita
- Giao diện cài đặt hiện đại, đồng bộ giao diện Dark/Light mode tự nhiên với hệ điều hành GNOME.

---

## 🏗️ Cấu trúc thư mục dự án

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
├── LICENSE                     # GNU General Public License v3.0
└── README.md
```

---

## 📦 Hướng dẫn cài đặt nhanh

### Cách 1: Cài đặt tự động từ mã nguồn (Khuyên dùng, không cần sudo)

```bash
git clone https://github.com/todo-laterx/nepkey.git
cd nepkey
bash scripts/install.sh
```

Sau khi chạy xong:
1. Mở **Cài đặt hệ thống (Settings)** -> **Bàn phím (Keyboard)** -> **Nguồn nhập (Input Sources)**.
2. Bấm dấu **`+`** -> Chọn **Tiếng Việt** -> Thêm **NepKey**.
3. Dùng tổ hợp phím **`Super + Space`** để chuyển đổi và bắt đầu gõ tiếng Việt.

---

### Cách 2: Cài đặt gói `.deb`

```bash
bash scripts/build-deb.sh
sudo dpkg -i nepkey_1.0.0_amd64.deb
```

---

## ⌨️ Phím tắt & Thao tác

- **`Super + Space`**: Chuyển đổi giữa bàn phím tiếng Anh và tiếng Việt qua hệ thống IBus.
- **`nepkey-settings`**: Mở cửa sổ cấu hình giao diện GTK4 / Libadwaita.

---

## 📄 Giấy phép

Dự án được phát hành theo giấy phép mã nguồn mở **GNU General Public License v3.0 (GPL-3.0)**.
Mã nguồn: [https://github.com/todo-laterx/nepkey](https://github.com/todo-laterx/nepkey)

Tên và logo `NepKey` là thương hiệu của dự án. Mọi fork/redistribute phải giữ nguyên giấy phép GPL-3.0 và ghi công `bamboo-core (MIT)`. Không được dùng tên `NepKey` cho bản fork gây nhầm lẫn mà không được phép.
