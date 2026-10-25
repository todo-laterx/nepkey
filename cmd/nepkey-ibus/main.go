package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"nepkey/pkg/config"
	nepibus "nepkey/pkg/ibus"
)

var (
	flagIBus       = flag.Bool("ibus", false, "Chạy ở chế độ thành phần IBus (được ibus-daemon tự động gọi)")
	flagStandalone = flag.Bool("standalone", false, "Chạy độc lập (tự động đăng ký với IBus daemon hiện tại)")
	flagXML        = flag.String("xml", "", "Xuất định dạng XML của IBus component (ra stdout nếu là \"-\")")
	flagInstall    = flag.Bool("install", false, "Cài đặt component XML vào ~/.local/share/ibus/component và nạp vào IBus")
	flagConfigFile = flag.String("config", "configs/config.yaml", "Đường dẫn file cấu hình YAML")
)

func installComponent() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	targetDir := filepath.Join(home, ".local", "share", "ibus", "component")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("không thể tạo thư mục %s: %w", targetDir, err)
	}

	targetFile := filepath.Join(targetDir, "nepkey.xml")
	f, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("không thể tạo file %s: %w", targetFile, err)
	}
	defer f.Close()

	comp := nepibus.MakeComponent()
	comp.OutputXML(f)
	log.Printf("[NepKey IBus] Đã cài đặt component XML vào: %s", targetFile)

	// Refresh IBus cache & restart
	log.Println("[NepKey IBus] Đang cập nhật bộ nhớ đệm IBus...")
	_ = exec.Command("ibus", "write-cache").Run()
	_ = exec.Command("ibus", "restart").Run()
	log.Println("[NepKey IBus] Hoàn tất! Bạn có thể chọn NepKey trong cài đặt bàn phím Ubuntu.")
	return nil
}

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig(*flagConfigFile)
	if err != nil {
		log.Printf("[NepKey IBus] Cảnh báo: Không thể nạp %s (%v), dùng cấu hình mặc định", *flagConfigFile, err)
		cfg = config.DefaultConfig()
	}

	if *flagInstall {
		if err := installComponent(); err != nil {
			log.Fatalf("[NepKey IBus] Lỗi cài đặt: %v", err)
		}
		return
	}

	if *flagXML != "" {
		comp := nepibus.MakeComponent()
		if *flagXML == "-" {
			comp.OutputXML(os.Stdout)
		} else {
			f, err := os.Create(*flagXML)
			if err != nil {
				log.Fatalf("Không thể tạo file %s: %v", *flagXML, err)
			}
			defer f.Close()
			comp.OutputXML(f)
			fmt.Printf("Đã xuất component XML ra %s\n", *flagXML)
		}
		return
	}

	standalone := *flagStandalone
	if !*flagIBus && !*flagStandalone {
		// Default to standalone if run directly by user from terminal
		standalone = true
	}

	if err := nepibus.RunServer(cfg, standalone); err != nil {
		log.Fatalf("[NepKey IBus] Lỗi khởi động: %v", err)
	}
}
