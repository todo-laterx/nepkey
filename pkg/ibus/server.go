package ibus

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/godbus/dbus"
	"github.com/sarim/goibus/ibus"
	"nepkey/pkg/config"
)

// appVersionStr is stamped at build time from the VERSION file:
// go build -ldflags "-X nepkey/pkg/ibus.appVersionStr=$(cat VERSION)"
var appVersionStr = "dev"

// appVersion returns the release version stamped at build time.
func appVersion() string {
	if appVersionStr == "" {
		return "dev"
	}
	return appVersionStr
}

var engineCounter uint64

// engineEnabledByDefault reports whether a freshly spawned engine starts
// with Vietnamese processing on: only the "nepkey" engine does, while
// "nepkey-en" is a pure English passthrough (top-bar E indicator).
func engineEnabledByDefault(name string) bool {
	return name != "nepkey-en"
}

// EnsureIBusAddress ensures IBUS_ADDRESS environment variable is set
func EnsureIBusAddress() string {
	if addr := os.Getenv("IBUS_ADDRESS"); addr != "" {
		return addr
	}

	// 1. Try running `ibus address` command
	out, err := exec.Command("ibus", "address").Output()
	if err == nil {
		addr := strings.TrimSpace(string(out))
		if addr != "" {
			_ = os.Setenv("IBUS_ADDRESS", addr)
			return addr
		}
	}

	// 2. Try scanning ~/.config/ibus/bus/*
	home := os.Getenv("HOME")
	busDir := filepath.Join(home, ".config", "ibus", "bus")
	entries, err := os.ReadDir(busDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				data, rerr := os.ReadFile(filepath.Join(busDir, entry.Name()))
				if rerr == nil {
					for _, line := range strings.Split(string(data), "\n") {
						if strings.HasPrefix(line, "IBUS_ADDRESS=") {
							addr := strings.TrimSpace(strings.TrimPrefix(line, "IBUS_ADDRESS="))
							if addr != "" {
								_ = os.Setenv("IBUS_ADDRESS", addr)
								return addr
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// MakeComponent creates the IBus component description for NepKey
func MakeComponent() *ibus.Component {
	comp := ibus.NewComponent(
		"org.freedesktop.IBus.NepKey",
		"NepKey Vietnamese Input Method",
		appVersion(),
		"GPL-3.0",
		"NepKey Contributors",
		"https://github.com/todo-laterx/nepkey",
		"/usr/bin/nepkey-ibus --ibus",
		"nepkey",
	)

	engineDesc := ibus.SmallEngineDesc(
		"nepkey",
		"NepKey",
		"Bộ gõ tiếng Việt",
		"vi",
		"GPL-3.0",
		"NepKey Contributors",
		"/usr/share/icons/nepkey/nepkey-v.svg",
		"us",
		"/usr/bin/nepkey-settings",
		appVersion(),
	)
	engineDesc.Rank = 99
	engineDesc.Symbol = "V"

	comp.AddEngine(engineDesc)

	engineDescEn := ibus.SmallEngineDesc(
		"nepkey-en",
		"NepKey",
		"NepKey English passthrough",
		"en",
		"GPL-3.0",
		"NepKey Contributors",
		"/usr/share/icons/nepkey/nepkey-e.svg",
		"us",
		"/usr/bin/nepkey-settings",
		appVersion(),
	)
	engineDescEn.Rank = 99
	engineDescEn.Symbol = "E"

	comp.AddEngine(engineDescEn)
	return comp
}

// RunServer starts the IBus engine service
func RunServer(cfg *config.Config, standalone bool) error {
	addr := EnsureIBusAddress()
	if addr == "" {
		return fmt.Errorf("không thể tìm thấy địa chỉ IBus daemon (IBus daemon chưa chạy?)")
	}

	bus := ibus.NewBus()
	conn := bus.GetDbusConn()

	creator := func(c *dbus.Conn, engineName string) dbus.ObjectPath {
		id := atomic.AddUint64(&engineCounter, 1)
		path := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/IBus/Engine/NepKey/%d", id))

		engine := NewNepKeyEngine(c, path, cfg)
		engine.engineName = engineName
		if !engineEnabledByDefault(engineName) {
			engine.enabled = false
			engine.eng.SetEnabled(false)
		}
		ibus.PublishEngine(c, path, engine)
		log.Printf("[NepKey IBus] Created engine instance #%d at %s", id, path)
		return path
	}

	ibus.NewFactory(conn, creator)

	if standalone {
		log.Println("[NepKey IBus] Registering component in standalone mode...")
		bus.RegisterComponent(MakeComponent())
		bus.CallMethod("SetGlobalEngine", 0, "nepkey")
	} else {
		log.Println("[NepKey IBus] Requesting well-known name org.freedesktop.IBus.NepKey...")
		bus.RequestName("org.freedesktop.IBus.NepKey", 0)
	}

	log.Println("[NepKey IBus] Ready. Engine is running.")
	select {}
}
