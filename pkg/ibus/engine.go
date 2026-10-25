package ibus

import (
	"os"
	"os/exec"
	"sync"
	"time"
	"unicode"

	"github.com/godbus/dbus"
	"github.com/sarim/goibus/ibus"
	"nepkey/pkg/config"
	"nepkey/pkg/engine"
)

const (
	IBusShiftMask   uint32 = 1 << 0
	IBusLockMask    uint32 = 1 << 1
	IBusControlMask uint32 = 1 << 2
	IBusMod1Mask    uint32 = 1 << 3 // Alt
	IBusSuperMask   uint32 = 1 << 26
	IBusReleaseMask uint32 = 1 << 30

	IBusKeyBackSpace uint32 = 0xff08
	IBusKeyTab       uint32 = 0xff09
	IBusKeyReturn    uint32 = 0xff0d
	IBusKeyEscape    uint32 = 0xff1b
	IBusKeySpace     uint32 = 0x0020
	IBusKeyKPEnter   uint32 = 0xff8d

	IBusKeyLeft  uint32 = 0xff51
	IBusKeyUp    uint32 = 0xff52
	IBusKeyRight uint32 = 0xff53
	IBusKeyDown  uint32 = 0xff54
	IBusKeyHome  uint32 = 0xff50
	IBusKeyEnd   uint32 = 0xff57
	IBusKeyPgUp  uint32 = 0xff55
	IBusKeyPgDn  uint32 = 0xff56

	IBusCapPreeditText     uint32 = 1 << 0
	IBusCapAuxiliaryText   uint32 = 1 << 1
	IBusCapLookupTable     uint32 = 1 << 2
	IBusCapFocus           uint32 = 1 << 3
	IBusCapProperty        uint32 = 1 << 4
	IBusCapSurroundingText uint32 = 1 << 5

	// IBusInputPurpose constants (from IBusInputPurpose enum)
	IBusPurposeFreeForm uint32 = 0
	IBusPurposePassword uint32 = 8
	IBusPurposePin      uint32 = 9

	// IBusInputHints constants
	IBusHintHiddenText uint32 = 1 << 7

	// Backspace evdev keycode (XKB keycode 22 - 8 = 14)
	EvdevKeyBackSpace uint32 = 14
)

// NepKeyEngine implements a Zero-Underline Preedit IBus input method engine
// using NepKey's Bamboo core with universal compatibility across Wayland and X11/XWayland.
type NepKeyEngine struct {
	ibus.Engine
	mu               sync.Mutex
	conn             *dbus.Conn
	eng              *engine.Engine
	cfg              *config.Config
	enabled          bool
	configFilePath   string
	configModTime    time.Time
	macroFilePath    string
	macroModTime     time.Time
	lastSyncCheck    time.Time
	atWordBoundary   bool   // true at start, after Space, Enter, Tab, Reset
	rawTokenMode     bool   // true when typing slash commands, file paths, URLs, CLI flags, code properties
	slashCommandMode bool   // alias synced with rawTokenMode for backward compatibility in tests
	slashCommandLen  int    // number of characters typed in active raw token mode
	inputPurpose     uint32 // current field input purpose (e.g. password field)
	inputHints       uint32 // current field input hints
}

// NewNepKeyEngine creates a new instance of NepKeyEngine
func NewNepKeyEngine(conn *dbus.Conn, objectPath dbus.ObjectPath, cfg *config.Config) *NepKeyEngine {
	eng := engine.NewEngine(cfg.InputMethod)
	eng.SetCharset(cfg.Charset)

	resolvedConfig := config.ResolveConfigPath("")
	var cfgModTime time.Time
	if resolvedConfig != "" {
		if fi, err := os.Stat(resolvedConfig); err == nil {
			cfgModTime = fi.ModTime()
		}
	}

	resolvedMacro := config.ResolveMacroPath(cfg.MacroFile)
	var macroMap map[string]string
	var modTime time.Time
	if cfg.MacroEnabled {
		if m, err := config.LoadMacros(resolvedMacro); err == nil {
			macroMap = m
		}
		if fi, err := os.Stat(resolvedMacro); err == nil {
			modTime = fi.ModTime()
		}
	}
	eng.SetOptions(cfg.FreeToneMarking, cfg.ModernToneStyle, cfg.AutoCorrect, cfg.MacroEnabled, macroMap)

	return &NepKeyEngine{
		conn:           conn,
		Engine:         ibus.BaseEngine(conn, objectPath),
		eng:            eng,
		cfg:            cfg,
		enabled:        cfg.Enabled,
		configFilePath: resolvedConfig,
		configModTime:  cfgModTime,
		macroFilePath:  resolvedMacro,
		macroModTime:   modTime,
		lastSyncCheck:  time.Now(),
		atWordBoundary: true,
	}
}

// syncSettingsIfModified checks if config.yaml or macros.yaml has been updated
// from nepkey-settings GUI and hot-reloads them with zero downtime.
func (e *NepKeyEngine) syncSettingsIfModified(force bool) {
	now := time.Now()
	if !force && now.Sub(e.lastSyncCheck) < 1500*time.Millisecond {
		return
	}
	e.lastSyncCheck = now

	// 1. Check config.yaml
	if e.configFilePath != "" {
		if fi, err := os.Stat(e.configFilePath); err == nil && fi.ModTime().After(e.configModTime) {
			if newCfg, err := config.LoadConfig(e.configFilePath); err == nil {
				e.configModTime = fi.ModTime()
				methodChanged := newCfg.InputMethod != e.cfg.InputMethod
				e.cfg = newCfg
				e.enabled = newCfg.Enabled
				if methodChanged {
					e.eng.SetInputMethod(newCfg.InputMethod)
				}
				e.eng.SetCharset(newCfg.Charset)
				e.eng.SetOptions(newCfg.FreeToneMarking, newCfg.ModernToneStyle, newCfg.AutoCorrect, newCfg.MacroEnabled, nil)
				if newCfg.MacroFile != "" {
					e.macroFilePath = config.ResolveMacroPath(newCfg.MacroFile)
				}
			}
		}
	}

	// 2. Check macros.yaml
	if e.cfg.MacroEnabled && e.macroFilePath != "" {
		if fi, err := os.Stat(e.macroFilePath); err == nil && fi.ModTime().After(e.macroModTime) {
			if m, err := config.LoadMacros(e.macroFilePath); err == nil {
				e.macroModTime = fi.ModTime()
				e.eng.SetMacros(m, e.cfg.MacroEnabled)
			}
		}
	}
}

func (e *NepKeyEngine) setRawMode(active bool) {
	e.rawTokenMode = active
	e.slashCommandMode = active
	if !active {
		e.slashCommandLen = 0
	}
}

func (e *NepKeyEngine) commit(text string) {
	if e.conn != nil && text != "" {
		e.CommitText(ibus.NewText(text))
	}
}

func (e *NepKeyEngine) hidePreedit() {
	if e.conn != nil {
		e.HidePreeditText()
	}
}

// updatePreedit renders the current composed Vietnamese word with ZERO underline (IBUS_ATTR_UNDERLINE_NONE).
// Mode CLEAR avoids double-commit: COMMIT makes clients auto-commit preedit on
// Hide/FocusOut, which duplicates our explicit CommitText (focus-out duplicate,
// terminal "runrun dev").
func (e *NepKeyEngine) updatePreedit(text string) {
	if e.conn != nil {
		t := ibus.NewText(text)
		// Zero-underline: do NOT add IBUS_ATTR_UNDERLINE_SINGLE.
		e.UpdatePreeditTextWithMode(t, uint32(len([]rune(text))), true, ibus.IBUS_ENGINE_PREEDIT_CLEAR)
	}
}

func (e *NepKeyEngine) commitBuffer() {
	if e.eng.HasBuffer() {
		buf := e.eng.GetWordBuffer()
		e.hidePreedit()
		if buf != "" {
			e.commit(buf)
		}
		e.eng.Reset()
	}
}

// isRawTokenContinuation returns true if r continues a slash command, file path, URL, CLI flag, or code identifier
func isRawTokenContinuation(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) ||
		r == '/' || r == '.' || r == '_' || r == '-' || r == ':' || r == '~' || r == '+' || r == '%' || r == '?' || r == '=' || r == '&'
}

// ProcessKeyEvent processes a keystroke sent by IBus
func (e *NepKeyEngine) ProcessKeyEvent(keyval uint32, keycode uint32, state uint32) (bool, *dbus.Error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. Ignore key release events
	if (state & IBusReleaseMask) != 0 {
		return false, nil
	}

	// 2. Bypass if Vietnamese input is disabled or inside a password/PIN field
	if !e.enabled || e.inputPurpose == IBusPurposePassword || e.inputPurpose == IBusPurposePin || (e.inputHints&IBusHintHiddenText) != 0 {
		return false, nil
	}

	// 3. Shortcuts (Ctrl, Alt, Super held down with normal keys): commit buffer & pass through
	if (state & (IBusControlMask | IBusMod1Mask | IBusSuperMask)) != 0 {
		e.setRawMode(false)
		e.atWordBoundary = true
		e.commitBuffer()
		return false, nil
	}

	// 4. Special Keysyms (>= 0xff00): BackSpace, Return, Tab, Arrows, Shift, Alt, Ctrl, F1-F12
	if keyval >= 0xff00 {
		switch keyval {
		case IBusKeyBackSpace:
			if e.rawTokenMode {
				if e.slashCommandLen > 0 {
					e.slashCommandLen--
				} else {
					e.setRawMode(false)
					e.atWordBoundary = true
				}
				return false, nil
			}
			if e.eng.HasBuffer() {
				e.eng.OnBackspace()
				buf := e.eng.GetWordBuffer()
				if buf == "" {
					e.hidePreedit()
					e.atWordBoundary = true
				} else {
					e.updatePreedit(buf)
				}
				return true, nil
			}
			e.atWordBoundary = true
			return false, nil

		case IBusKeyReturn, IBusKeyKPEnter:
			e.setRawMode(false)
			e.atWordBoundary = true
			if e.eng.HasBuffer() {
				e.syncSettingsIfModified(false)
				text := e.eng.GetWordBuffer()
				if act := e.eng.CheckMacro(0); act != nil {
					text = string(act.AppendRunes)
				}
				e.hidePreedit()
				e.commit(text)
				e.eng.Reset()
				// ASCII thuần (you, gmail): nuốt Enter để Chrome giữ highlight
				// suggestion (youtube), Enter lần 2 sẽ navigate. Tránh race
				// CommitText + forward khiến Enter đầu chỉ chốt thành "you".
				// Tiếng Việt có dấu giữ single-press submit như cũ.
				if isPureASCII(text) {
					return true, nil
				}
				// Single-press Enter: committing text AND returning false allows
				// Web Chat, Zalo, Terminal, and IDE to submit/execute in 1 single press!
				return false, nil
			}
			return false, nil

		case IBusKeyTab:
			e.setRawMode(false)
			e.atWordBoundary = true
			if e.eng.HasBuffer() {
				e.syncSettingsIfModified(false)
				if act := e.eng.CheckMacro('\t'); act != nil {
					runes := act.AppendRunes
					if len(runes) > 0 && runes[len(runes)-1] == '\t' {
						runes = runes[:len(runes)-1]
					}
					e.hidePreedit()
					e.commit(string(runes))
					e.eng.Reset()
					return true, nil
				}
				text := e.eng.GetWordBuffer()
				e.hidePreedit()
				e.commit(text)
				e.eng.Reset()
				return false, nil
			}
			return false, nil

		case IBusKeyEscape:
			e.setRawMode(false)
			e.atWordBoundary = true
			if e.eng.HasBuffer() {
				e.hidePreedit()
				e.commit(e.eng.GetWordBuffer())
				e.eng.Reset()
				return true, nil
			}
			return false, nil

		default:
			// Modifiers (Shift, CapsLock) should NOT reset word buffer!
			if keyval >= 0xffe1 && keyval <= 0xffee {
				return false, nil
			}
			// Navigation (Arrows, PgUp, PgDn, Home, End), Functions (F1-F12)
			e.setRawMode(false)
			e.atWordBoundary = true
			e.commitBuffer()
			return false, nil
		}
	}

	// 5. Space key (keyval = 0x0020)
	if keyval == IBusKeySpace {
		e.setRawMode(false)
		e.atWordBoundary = true
		if e.eng.HasBuffer() {
			e.syncSettingsIfModified(false)
			text := e.eng.GetWordBuffer() + " "
			if act := e.eng.CheckMacro(' '); act != nil {
				text = string(act.AppendRunes)
			}
			e.hidePreedit()
			e.commit(text)
			e.eng.Reset()
			return true, nil
		}
		return false, nil
	}

	// 6. Control characters (< 0x0020)
	if keyval < 0x0020 && keyval != 0 {
		e.setRawMode(false)
		e.atWordBoundary = true
		e.commitBuffer()
		return false, nil
	}

	// 7. Printable characters (ASCII / Unicode)
	r := rune(keyval)
	if !unicode.IsPrint(r) {
		e.setRawMode(false)
		e.atWordBoundary = true
		e.commitBuffer()
		return false, nil
	}

	// 8. Active rawTokenMode: pass through slash commands (/cmd), multi-segment paths (/home/tuannt/file.go),
	// URLs (github.com/user/repo), CLI flags (--help, -rf), and code properties (fmt.Errorf, user_id) at 0ms!
	if e.rawTokenMode {
		if isRawTokenContinuation(r) {
			e.slashCommandLen++
			return false, nil
		}
		// Delimiters exit rawTokenMode
		e.setRawMode(false)
	}

	// 9. Detect start of a new Raw Token at word boundary:
	// '/', '~', '$', '@', '#', or '-' (CLI flags like -la, -rf, --version)
	if !e.eng.HasBuffer() && e.atWordBoundary {
		if r == '/' || r == '~' || r == '$' || r == '@' || r == '#' || r == '-' {
			e.setRawMode(true)
			e.slashCommandLen = 0
			e.atWordBoundary = false
			return false, nil
		}
	}

	// 10. Digits ('0' - '9'): word-break in Telex, tone keys in VNI.
	// Bamboo returns empty for "P"+"1" (digit splits last word), so feeding
	// a digit into a non-empty Telex buffer would discard "P". Commit first.
	if r >= '0' && r <= '9' {
		if e.eng.CanProcessKey(r) {
			e.atWordBoundary = false
			_ = e.eng.ProcessKey(r)
			if buf := e.eng.GetWordBuffer(); buf != "" {
				e.updatePreedit(buf)
				return true, nil
			}
			e.hidePreedit()
			return false, nil
		}
		e.atWordBoundary = false
		if e.eng.HasBuffer() {
			text := e.eng.GetWordBuffer()
			e.hidePreedit()
			e.commit(text)
			e.eng.Reset()
		}
		return false, nil
	}

	// 11. Non-word characters (punctuation, symbols: ., /, _, :, -, (, ), etc.)
	if !engine.IsWordCharacter(r) {
		hadBuffer := e.eng.HasBuffer()
		wasAtBoundary := e.atWordBoundary

		enterRawAfterSep := false
		if !wasAtBoundary {
			if r == '_' || r == ':' {
				enterRawAfterSep = true
			} else if r == '.' && isPureASCII(e.eng.GetWordBuffer()) {
				enterRawAfterSep = true
			}
		} else if r == '.' {
			enterRawAfterSep = true
		}

		if hadBuffer {
			e.syncSettingsIfModified(false)
			text := e.eng.GetWordBuffer() + string(r)
			if act := e.eng.CheckMacro(r); act != nil {
				text = string(act.AppendRunes)
			}
			e.hidePreedit()
			e.commit(text)
			e.eng.Reset()
			e.setRawMode(enterRawAfterSep)
			e.atWordBoundary = false
			return true, nil
		}

		e.setRawMode(enterRawAfterSep)
		e.atWordBoundary = (r == ' ' || r == '\n' || r == '\t' || r == '(' || r == '[' || r == '{' || r == '"' || r == '\'')
		return false, nil
	}

	// 12. Normal Word characters (letters a-z, A-Z, etc.) - Vietnamese composition via Bamboo core
	e.atWordBoundary = false
	_ = e.eng.ProcessKey(r)
	buf := e.eng.GetWordBuffer()
	if buf != "" {
		e.updatePreedit(buf)
		return true, nil
	}
	e.hidePreedit()
	return false, nil
}

func isPureASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return len(s) > 0
}

// Reset is called when input context resets (e.g. user clicks elsewhere)
func (e *NepKeyEngine) Reset() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.setRawMode(false)
	e.atWordBoundary = true
	e.commitBuffer()
	return nil
}

// FocusIn is called when an application focuses an input context - register Top Bar properties
func (e *NepKeyEngine) FocusIn() *dbus.Error {
	e.mu.Lock()
	e.setRawMode(false)
	e.atWordBoundary = true
	e.syncSettingsIfModified(true)
	methodName := e.cfg.InputMethod
	e.mu.Unlock()

	if e.conn != nil {
		propSetup := ibus.NewProperty(
			"setup",
			0,
			"Cài đặt NepKey...",
			"preferences-system",
			"Mở bảng điều khiển cài đặt NepKey",
			true,
			true,
			0,
		)
		propMethod := ibus.NewProperty(
			"method",
			0,
			"Kiểu gõ: "+methodName,
			"",
			"Kiểu gõ hiện tại",
			true,
			true,
			0,
		)
		props := ibus.NewPropList(propSetup, propMethod)
		e.RegisterProperties(props)
	}
	return nil
}

// FocusOut is called when an application loses focus
func (e *NepKeyEngine) FocusOut() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.setRawMode(false)
	e.atWordBoundary = true
	e.inputPurpose = IBusPurposeFreeForm
	e.inputHints = 0
	e.commitBuffer()
	return nil
}

// SetContentType receives input purpose and hints from GNOME Shell / GTK / Wayland clients
////@method(in_signature="uu")
func (e *NepKeyEngine) SetContentType(purpose uint32, hints uint32) *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.inputPurpose = purpose
	e.inputHints = hints
	if purpose == IBusPurposePassword || purpose == IBusPurposePin || (hints&IBusHintHiddenText) != 0 {
		e.setRawMode(false)
		e.commitBuffer()
	}
	return nil
}

// PropertyActivate handles clicks on Top Bar menu items
func (e *NepKeyEngine) PropertyActivate(propName string, propState uint32) *dbus.Error {
	if propName == "setup" {
		go func() {
			_ = exec.Command("nepkey-settings").Run()
		}()
	}
	return nil
}

// Enable enables the engine
func (e *NepKeyEngine) Enable() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.syncSettingsIfModified(true)
	e.enabled = true
	return nil
}

// Disable disables the engine
func (e *NepKeyEngine) Disable() *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.setRawMode(false)
	e.atWordBoundary = true
	e.commitBuffer()
	e.enabled = false
	return nil
}

// SetCapabilities reports supported IBus capabilities
func (e *NepKeyEngine) SetCapabilities(cap uint32) *dbus.Error {
	return nil
}
