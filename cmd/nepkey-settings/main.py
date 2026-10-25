#!/usr/bin/env python3
"""
NepKey Settings - Native GTK4 + Libadwaita Preferences Window
"""

import os
import sys
import yaml
import subprocess
import socket
import gi

gi.require_version("Gtk", "4.0")
gi.require_version("Adw", "1")
from gi.repository import Adw, Gtk, Gio, GLib

CONFIG_DIR = os.path.expanduser("~/.config/nepkey")
CONFIG_FILE = os.path.join(CONFIG_DIR, "config.yaml")

APP_VERSION = "__NEPKEY_VERSION__"
if APP_VERSION.startswith("__NEPKEY"):
    try:
        _vf = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "VERSION")
        with open(_vf, encoding="utf-8") as _f:
            APP_VERSION = _f.read().strip() or "dev"
    except Exception:
        APP_VERSION = "dev"
MACRO_FILE = os.path.join(CONFIG_DIR, "macros.yaml")

DEFAULT_CONFIG = {
    "input_method": "Telex 2",
    "charset": "Unicode",
    "output_method": "clipboard",
    "device_path": "auto",
    "uinput_path": "/dev/uinput",
    "toggle_key": "Ctrl+Space",
    "enabled": True,
    "free_tone_marking": True,
    "modern_tone_style": True,
    "auto_correct": True,
    "macro_enabled": True,
    "macro_file": MACRO_FILE,
}

DEFAULT_MACROS = {
    "vn": "Việt Nam",
    "hn": "Hà Nội",
    "hcm": "Hồ Chí Minh",
    "dc": "được",
    "ko": "không",
    "ntn": "như thế nào",
    "bt": "bình thường",
    "ok": "đồng ý",
}


def ensure_configs():
    os.makedirs(CONFIG_DIR, exist_ok=True)
    if not os.path.exists(CONFIG_FILE):
        with open(CONFIG_FILE, "w", encoding="utf-8") as f:
            yaml.dump(DEFAULT_CONFIG, f, allow_unicode=True, default_flow_style=False)
    if not os.path.exists(MACRO_FILE):
        with open(MACRO_FILE, "w", encoding="utf-8") as f:
            yaml.dump(DEFAULT_MACROS, f, allow_unicode=True, default_flow_style=False)


def load_config():
    ensure_configs()
    try:
        with open(CONFIG_FILE, "r", encoding="utf-8") as f:
            cfg = yaml.safe_load(f) or {}
            merged = DEFAULT_CONFIG.copy()
            merged.update(cfg)
            return merged
    except Exception as e:
        print(f"Error loading config: {e}", file=sys.stderr)
        return DEFAULT_CONFIG.copy()


def save_config(cfg):
    ensure_configs()
    try:
        with open(CONFIG_FILE, "w", encoding="utf-8") as f:
            yaml.dump(cfg, f, allow_unicode=True, default_flow_style=False)
    except Exception as e:
        print(f"Error saving config: {e}", file=sys.stderr)


def load_macros():
    ensure_configs()
    try:
        with open(MACRO_FILE, "r", encoding="utf-8") as f:
            data = yaml.safe_load(f) or {}
            return data
    except Exception as e:
        print(f"Error loading macros: {e}", file=sys.stderr)
        return DEFAULT_MACROS.copy()


def save_macros(macros):
    ensure_configs()
    try:
        with open(MACRO_FILE, "w", encoding="utf-8") as f:
            yaml.dump(macros, f, allow_unicode=True, default_flow_style=False)
    except Exception as e:
        print(f"Error saving macros: {e}", file=sys.stderr)


class NepKeySettingsWindow(Adw.PreferencesWindow):
    def __init__(self, app):
        super().__init__(application=app, title="Cài đặt NepKey")
        self.set_default_size(720, 640)
        self.set_size_request(600, 520)
        self.cfg = load_config()
        self.macros = load_macros()

        # Build pages
        self.build_input_page()
        self.build_tone_page()
        self.build_macro_page()
        self.build_about_page()

    def build_input_page(self):
        page = Adw.PreferencesPage(title="Kiểu gõ", icon_name="input-keyboard-symbolic")

        # Group 1: Bàn phím & Phương thức gõ
        grp_methods = Adw.PreferencesGroup(title="Phương thức gõ tiếng Việt", description="Chọn kiểu gõ và phím chuyển đổi V/E")

        # Combo: Input Method
        self.method_names = ["Telex 2", "Telex", "VNI", "VIQR"]
        model_methods = Gtk.StringList.new(self.method_names)
        self.row_method = Adw.ComboRow(title="Kiểu gõ chính", subtitle="Kiểu gõ thông dụng trên bàn phím chuẩn", model=model_methods)
        curr_m = self.cfg.get("input_method", "Telex 2")
        if curr_m in self.method_names:
            self.row_method.set_selected(self.method_names.index(curr_m))
        self.row_method.connect("notify::selected", self.on_setting_changed)
        grp_methods.add(self.row_method)

        # Combo: Toggle Key
        self.toggle_keys = ["Ctrl+Space", "Alt+Shift", "Ctrl+Shift"]
        model_toggles = Gtk.StringList.new(self.toggle_keys)
        self.row_toggle = Adw.ComboRow(title="Phím tắt chuyển đổi V/E", subtitle="Chuyển nhanh giữa chế độ Tiếng Việt và Tiếng Anh", model=model_toggles)
        curr_tog = self.cfg.get("toggle_key", "Ctrl+Space")
        if curr_tog in self.toggle_keys:
            self.row_toggle.set_selected(self.toggle_keys.index(curr_tog))
        self.row_toggle.connect("notify::selected", self.on_setting_changed)
        grp_methods.add(self.row_toggle)

        page.add(grp_methods)

        # Group 2: Khởi động
        grp_startup = Adw.PreferencesGroup(title="Trạng thái", description="Cấu hình chế độ hoạt động ban đầu")
        self.row_enabled = Adw.SwitchRow(title="Mặc định bật Tiếng Việt", subtitle="Kích hoạt sẵn bộ gõ khi NepKey vừa khởi chạy")
        self.row_enabled.set_active(self.cfg.get("enabled", True))
        self.row_enabled.connect("notify::active", self.on_setting_changed)
        grp_startup.add(self.row_enabled)

        page.add(grp_startup)

        # Group 3: Experimental
        grp_exp = Adw.PreferencesGroup(title="Tính năng thử nghiệm", description="Áp dụng ngay không cần khởi động lại")
        self.row_ascii_fast = Adw.SwitchRow(
            title="Chữ ASCII chuyển tiếp nhanh",
            subtitle="Enter/Tab ăn gợi ý thanh địa chỉ trong 1 lần bấm (Chrome, ô nhập GTK)"
        )
        self.row_ascii_fast.set_active(self.cfg.get("ascii_fast_path", False))
        self.row_ascii_fast.connect("notify::active", self.on_setting_changed)
        grp_exp.add(self.row_ascii_fast)

        page.add(grp_exp)
        self.add(page)

    def build_tone_page(self):
        page = Adw.PreferencesPage(title="Dấu & Sửa lỗi", icon_name="preferences-desktop-font-symbolic")

        # Group: Quy tắc bỏ dấu
        grp_tone = Adw.PreferencesGroup(title="Quy tắc bỏ dấu thanh", description="Tiêu chuẩn vị trí đặt dấu trong tiếng Việt")

        self.row_modern_tone = Adw.SwitchRow(
            title="Đặt dấu kiểu mới (òa, òe, ùy)",
            subtitle="Đặt dấu vào nguyên âm đầu trong cặp nguyên âm mở thay vì kiểu cũ (oà, oè, uỳ)"
        )
        self.row_modern_tone.set_active(self.cfg.get("modern_tone_style", True))
        self.row_modern_tone.connect("notify::active", self.on_setting_changed)
        grp_tone.add(self.row_modern_tone)

        self.row_free_tone = Adw.SwitchRow(
            title="Cho phép bỏ dấu tự do",
            subtitle="Có thể gõ phím dấu ở bất kỳ vị trí nào trong từ (Ví dụ: hoanf -> hoàn)"
        )
        self.row_free_tone.set_active(self.cfg.get("free_tone_marking", True))
        self.row_free_tone.connect("notify::active", self.on_setting_changed)
        grp_tone.add(self.row_free_tone)

        page.add(grp_tone)

        # Group: Chính tả
        grp_spell = Adw.PreferencesGroup(title="Kiểm tra và tự động sửa lỗi", description="Khôi phục trạng thái chuẩn khi gõ sai từ")
        self.row_auto_correct = Adw.SwitchRow(
            title="Tự động kiểm tra chính tả",
            subtitle="Tự động hủy bỏ dấu nếu từ gõ không tuân theo quy tắc chính tả tiếng Việt"
        )
        self.row_auto_correct.set_active(self.cfg.get("auto_correct", True))
        self.row_auto_correct.connect("notify::active", self.on_setting_changed)
        grp_spell.add(self.row_auto_correct)

        page.add(grp_spell)
        self.add(page)

    def build_macro_page(self):
        self.page_macro = Adw.PreferencesPage(title="Gõ tắt", icon_name="document-edit-symbolic")

        # Group: Bật tắt Macro
        grp_macro_toggle = Adw.PreferencesGroup(title="Tính năng gõ tắt", description="Tự động mở rộng từ viết tắt khi nhấn khoảng trắng hoặc dấu câu")
        self.row_macro_enabled = Adw.SwitchRow(title="Bật tính năng gõ tắt", subtitle="Áp dụng bảng từ viết tắt bên dưới")
        self.row_macro_enabled.set_active(self.cfg.get("macro_enabled", True))
        self.row_macro_enabled.connect("notify::active", self.on_setting_changed)
        grp_macro_toggle.add(self.row_macro_enabled)
        self.page_macro.add(grp_macro_toggle)

        # Group: Danh sách Macro
        self.grp_macro_list = Adw.PreferencesGroup(title="Danh sách từ gõ tắt")

        # Action bar to add new macro
        add_btn = Gtk.Button(icon_name="list-add-symbolic", tooltip_text="Thêm từ gõ tắt mới")
        add_btn.add_css_class("flat")
        add_btn.connect("clicked", self.on_add_macro_clicked)
        self.grp_macro_list.set_header_suffix(add_btn)

        self.refresh_macro_rows()
        self.page_macro.add(self.grp_macro_list)
        self.add(self.page_macro)

    def refresh_macro_rows(self):
        # Clear existing rows in grp_macro_list
        # We re-populate
        for k, v in sorted(self.macros.items()):
            row = Adw.ActionRow(title=k, subtitle=v)
            del_btn = Gtk.Button(icon_name="user-trash-symbolic", tooltip_text=f"Xóa từ gõ tắt '{k}'")
            del_btn.add_css_class("flat")
            del_btn.add_css_class("error")
            del_btn.connect("clicked", self.create_delete_macro_handler(k, row))
            row.add_suffix(del_btn)
            self.grp_macro_list.add(row)

    def create_delete_macro_handler(self, key, row):
        def handler(btn):
            if key in self.macros:
                del self.macros[key]
                save_macros(self.macros)
                self.grp_macro_list.remove(row)
        return handler

    def on_add_macro_clicked(self, btn):
        dialog = Adw.MessageDialog(
            transient_for=self,
            heading="Thêm từ gõ tắt mới",
            body="Nhập từ viết tắt và nội dung thay thế tương ứng"
        )
        dialog.add_response("cancel", "Hủy")
        dialog.add_response("add", "Thêm")
        dialog.set_response_appearance("add", Adw.ResponseAppearance.SUGGESTED)

        # Form content
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=12)
        box.set_margin_top(8)
        box.set_margin_bottom(8)

        entry_key = Gtk.Entry(placeholder_text="Từ viết tắt (Ví dụ: vn)")
        entry_val = Gtk.Entry(placeholder_text="Nội dung mở rộng (Ví dụ: Việt Nam)")

        box.append(entry_key)
        box.append(entry_val)
        dialog.set_extra_child(box)

        def on_response(d, resp):
            if resp == "add":
                k = entry_key.get_text().strip()
                v = entry_val.get_text().strip()
                if k and v:
                    self.macros[k] = v
                    save_macros(self.macros)
                    # Add row to UI
                    row = Adw.ActionRow(title=k, subtitle=v)
                    del_btn = Gtk.Button(icon_name="user-trash-symbolic", tooltip_text=f"Xóa từ gõ tắt '{k}'")
                    del_btn.add_css_class("flat")
                    del_btn.add_css_class("error")
                    del_btn.connect("clicked", self.create_delete_macro_handler(k, row))
                    row.add_suffix(del_btn)
                    self.grp_macro_list.add(row)

        dialog.connect("response", on_response)
        dialog.present()

    def build_about_page(self):
        page = Adw.PreferencesPage(title="Giới thiệu", icon_name="help-about-symbolic")

        # 1. Hero Group
        grp_hero = Adw.PreferencesGroup()
        box_hero = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=10)
        box_hero.set_margin_top(20)
        box_hero.set_margin_bottom(24)

        # Large vector icon (file fallback: user-local theme lookup can miss)
        logo_path = os.path.join(os.path.expanduser("~/.local/share/icons/nepkey"), "nepkey.svg")
        if not os.path.exists(logo_path):
            logo_path = "/usr/share/icons/nepkey/nepkey.svg"
        if os.path.exists(logo_path):
            img_logo = Gtk.Image.new_from_file(logo_path)
        else:
            img_logo = Gtk.Image.new_from_icon_name("nepkey")
        img_logo.set_pixel_size(96)
        box_hero.append(img_logo)

        # Title
        lbl_title = Gtk.Label(label="NepKey IBus")
        lbl_title.add_css_class("title-1")
        box_hero.append(lbl_title)

        # Subtitle / Version
        lbl_version = Gtk.Label(label=f"Phiên bản {APP_VERSION} • Native IBus Engine")
        lbl_version.add_css_class("title-4")
        lbl_version.add_css_class("dim-label")
        box_hero.append(lbl_version)

        # Tagline
        lbl_desc = Gtk.Label(label="Bộ gõ tiếng Việt cho lập trình viên và duyệt web")
        lbl_desc.add_css_class("body")
        box_hero.append(lbl_desc)

        grp_hero.add(box_hero)
        page.add(grp_hero)

        # 2. Features Group
        grp_features = Adw.PreferencesGroup(title="Đặc điểm nổi bật")

        row1 = Adw.ActionRow(
            title="Chat AI và Terminal",
            subtitle="Gõ lệnh / tự nhiên, Enter 1 lần gửi ngay"
        )
        row1.add_prefix(Gtk.Image.new_from_icon_name("utilities-terminal-symbolic"))
        grp_features.add(row1)

        row2 = Adw.ActionRow(
            title="URL, đường dẫn",
            subtitle="Không biến dạng link khi gõ"
        )
        row2.add_prefix(Gtk.Image.new_from_icon_name("network-wired-symbolic"))
        grp_features.add(row2)

        row3 = Adw.ActionRow(
            title="Điều hướng thông minh",
            subtitle="Hết lỗi lặp chữ khi dùng ← → Home End"
        )
        row3.add_prefix(Gtk.Image.new_from_icon_name("input-keyboard-symbolic"))
        grp_features.add(row3)

        row4 = Adw.ActionRow(
            title="Bamboo Core, Macro",
            subtitle="Telex 2 / Telex / VNI, gõ tắt YAML"
        )
        row4.add_prefix(Gtk.Image.new_from_icon_name("preferences-desktop-font-symbolic"))
        grp_features.add(row4)

        page.add(grp_features)

        # 3. Project Info Group
        grp_info = Adw.PreferencesGroup(title="Thông tin dự án")
        row_github = Adw.ActionRow(
            title="Mã nguồn GitHub",
            subtitle="https://github.com/todo-laterx/nepkey"
        )
        row_github.add_prefix(Gtk.Image.new_from_icon_name("software-update-available-symbolic"))
        grp_info.add(row_github)

        row_license = Adw.ActionRow(
            title="Giấy phép mã nguồn mở",
            subtitle="Phát hành theo giấy phép GNU General Public License v3.0 (GPL-3.0)"
        )
        row_license.add_prefix(Gtk.Image.new_from_icon_name("dialog-information-symbolic"))
        grp_info.add(row_license)

        page.add(grp_info)
        self.add(page)

    def on_setting_changed(self, *args):
        # Read from widgets and save
        m_idx = self.row_method.get_selected()
        if 0 <= m_idx < len(self.method_names):
            self.cfg["input_method"] = self.method_names[m_idx]

        t_idx = self.row_toggle.get_selected()
        if 0 <= t_idx < len(self.toggle_keys):
            self.cfg["toggle_key"] = self.toggle_keys[t_idx]

        self.cfg["enabled"] = self.row_enabled.get_active()
        self.cfg["modern_tone_style"] = self.row_modern_tone.get_active()
        self.cfg["free_tone_marking"] = self.row_free_tone.get_active()
        self.cfg["auto_correct"] = self.row_auto_correct.get_active()
        self.cfg["macro_enabled"] = self.row_macro_enabled.get_active()
        self.cfg["ascii_fast_path"] = self.row_ascii_fast.get_active()

        save_config(self.cfg)
        self.notify_daemon()

    def notify_daemon(self):
        try:
            s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            s.connect("/tmp/nepkey.sock")
            s.sendall(f"SET_METHOD:{self.cfg.get('input_method', 'Telex 2')}\n".encode("utf-8"))
            s.sendall(f"SET_CHARSET:{self.cfg.get('charset', 'Unicode')}\n".encode("utf-8"))
            s.close()
        except Exception:
            pass


class NepKeySettingsApp(Adw.Application):
    def __init__(self):
        super().__init__(
            application_id="org.nepkey.Settings",
            flags=Gio.ApplicationFlags.FLAGS_NONE
        )

    def do_activate(self):
        win = self.props.active_window
        if not win:
            win = NepKeySettingsWindow(self)
        win.present()


if __name__ == "__main__":
    app = NepKeySettingsApp()
    sys.exit(app.run(sys.argv))
