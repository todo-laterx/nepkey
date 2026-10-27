package ibus

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/godbus/dbus"
	"github.com/sarim/goibus/ibus"
	"github.com/todo-laterx/nepkey/pkg/config"
	"github.com/todo-laterx/nepkey/pkg/engine"
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
	IBusKeyShiftL    uint32 = 0xffe1
	IBusKeyShiftR    uint32 = 0xffe2

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

	// Real BackSpace key events carry the XKB keycode (evdev + 8) on both X11
	// and Wayland, so synthetic corrections must use 22, not 14.
	XKBKeyBackSpace uint32 = 22
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

	asciiFastPath        bool         // forward pure-ASCII keystrokes without preedit (Chrome Omnibox Enter/Tab)
	forwardedLen         int          // runes of the current buffer already forwarded to the client
	correctionBackspaces int          // synthetic BackSpaces requested (observable in tests)
	bsKeycode            uint32       // keycode for synthetic BackSpace, learned from real events
	clientCaps           uint32       // IBus capabilities reported by the focused client
	hasSurrounding       bool         // client proved surrounding-text support by pushing it (terminals never do)
	engineName           string       // "nepkey" (Vietnamese) or "nepkey-en" (English passthrough)
	wordFastPath         bool         // latched at word start: fast-path for the whole word or not
	restoreCommits       int          // prefix restores after preedit abandon (observable in tests)
	switchRequests       []string     // SetGlobalEngine targets requested (observable in tests)
	commitLog            []string     // every non-empty commit request (observable in tests)
	preeditHides         int          // HidePreeditText requests (observable in tests)
	dbg                  *os.File     // opt-in debug sink (config debug_log or NEPKEY_DEBUG)
	focusClass           atomic.Value // WM_CLASS of the focused X11 window ("" when unknown)
	focusSeq             atomic.Uint64
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

	e := &NepKeyEngine{
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
		asciiFastPath:  cfg.AsciiFastPath,
		bsKeycode:      XKBKeyBackSpace,
	}
	e.focusClass.Store("")
	if cfg.DebugLog || os.Getenv("NEPKEY_DEBUG") != "" {
		e.setDebug(true)
	}
	return e
}

// debugPath is where opt-in diagnostics are written.
func debugPath() string {
	if p := os.Getenv("NEPKEY_DEBUG_FILE"); p != "" {
		return p
	}
	return "/tmp/nepkey-debug.log"
}

// setDebug opens or closes the diagnostics sink. Off by default so a normal
// install never writes logs; config debug_log or NEPKEY_DEBUG enables it.
func (e *NepKeyEngine) setDebug(enable bool) {
	if !enable {
		if e.dbg != nil {
			_ = e.dbg.Close()
			e.dbg = nil
		}
		return
	}
	if e.dbg != nil {
		return
	}
	if f, err := os.OpenFile(debugPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		e.dbg = f
	}
}

// dbgf writes one timestamped diagnostic line when debug is enabled.
func (e *NepKeyEngine) dbgf(format string, args ...interface{}) {
	if e.dbg == nil {
		return
	}
	line := fmt.Sprintf("%s "+format+"\n", append([]interface{}{time.Now().Format("15:04:05.000")}, args...)...)
	_, _ = e.dbg.WriteString(line)
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
				debugChanged := newCfg.DebugLog != e.cfg.DebugLog
				e.cfg = newCfg
				e.asciiFastPath = newCfg.AsciiFastPath
				if debugChanged {
					e.setDebug(newCfg.DebugLog || os.Getenv("NEPKEY_DEBUG") != "")
				}
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
	if text == "" {
		return
	}
	// Bounded log: only the tail is ever inspected by tests.
	e.commitLog = append(e.commitLog, text)
	if len(e.commitLog) > 32 {
		e.commitLog = e.commitLog[len(e.commitLog)-32:]
	}
	if e.conn != nil {
		e.CommitText(ibus.NewText(text))
	}
}

func (e *NepKeyEngine) hidePreedit() {
	e.preeditHides++
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

// unforwarded returns the suffix of the current word buffer that the client
// has not received yet (empty when everything was forwarded eagerly).
func (e *NepKeyEngine) unforwarded() string {
	buf := []rune(e.eng.GetWordBuffer())
	if e.forwardedLen >= len(buf) {
		return ""
	}
	return string(buf[e.forwardedLen:])
}

// resetBufferState clears both the composition buffer and fast-path accounting.
func (e *NepKeyEngine) resetBufferState() {
	e.eng.Reset()
	e.forwardedLen = 0
	e.wordFastPath = false
}

// latchWordMode freezes the fast-path decision for the word being started.
// Clients (Electron) may report capabilities a few keystrokes late after
// focus (e.g. caps=9 then caps=41 mid-word); re-reading caps per keystroke
// would mix preedit and forwarded state in one word and corrupt it.
//
// Terminals (VTE/Ptyxis) advertise the surrounding-text capability bit but
// never push surrounding text and ignore DeleteSurroundingText, so a
// forward-then-erase cycle leaves ghosts and visible jumping on the PTY.
// The fast-path is therefore armed only after the client proves it is a real
// text field by pushing surrounding text (see SetSurroundingText); terminals
// never do and stay on the legacy preedit path even with ascii_fast_path
// enabled. Erasure itself always uses synthetic BackSpace key events, which
// every client honors (unlike DeleteSurroundingText, which several clients
// silently ignore and cause duplicated text).
func (e *NepKeyEngine) latchWordMode(wasEmpty bool) {
	if wasEmpty {
		class, _ := e.focusClass.Load().(string)
		e.wordFastPath = e.asciiFastPath &&
			e.clientCaps&IBusCapSurroundingText != 0 &&
			e.hasSurrounding &&
			!e.blockedClass(class)
		e.dbgf("LATCH fast=%v class=%q caps=%d hasSur=%v", e.wordFastPath, class, e.clientCaps, e.hasSurrounding)
	}
}

// blockedClass reports whether the focused X11 window class is on the
// fast-path blocklist (apps that advertise surrounding-text but ignore the
// deletion, e.g. Zalo). Unknown classes (Wayland-native apps) are allowed.
func (e *NepKeyEngine) blockedClass(class string) bool {
	if class == "" {
		return false
	}
	c := strings.ToLower(class)
	for _, b := range e.cfg.FastPathBlock {
		b = strings.ToLower(strings.TrimSpace(b))
		if b != "" && strings.Contains(c, b) {
			return true
		}
	}
	return false
}

var wmClassQuoted = regexp.MustCompile(`"([^"]*)"`)

// queryFocusClass reads the WM_CLASS of the currently active X11 window.
// Wayland-native clients (the usual case for browsers and Electron on GNOME)
// expose no X11 class, so the caller treats "" as "allow".
func queryFocusClass() string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	out, err := exec.CommandContext(ctx, "xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
	if err != nil {
		return ""
	}
	idMatch := regexp.MustCompile(`0x[0-9a-fA-F]+`).FindString(string(out))
	if idMatch == "" {
		return ""
	}
	out, err = exec.CommandContext(ctx, "xprop", "-id", idMatch, "WM_CLASS").Output()
	if err != nil {
		return ""
	}
	matches := wmClassQuoted.FindAllStringSubmatch(string(out), -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1][1]
}

// captureFocusClass resolves the focused app after the window manager has
// settled, and drops the result if focus moved again meanwhile.
func (e *NepKeyEngine) captureFocusClass(seq uint64) {
	time.Sleep(250 * time.Millisecond)
	if e.focusSeq.Load() != seq {
		return
	}
	class := queryFocusClass()
	if e.focusSeq.Load() != seq {
		return
	}
	e.focusClass.Store(class)
	e.dbgf("FOCUS-CLASS class=%q blocked=%v", class, e.blockedClass(class))
}

// sendBackspaces forwards n synthetic BackSpace key events to the client.
// Forwarded keys go to the application, never back into ProcessKeyEvent.
// With nil conn (unit tests) only the observable counter advances.
func (e *NepKeyEngine) sendBackspaces(n int) {
	if n <= 0 {
		return
	}
	e.dbgf("ERASE-BS n=%d keycode=%d caps=%d", n, e.bsKeycode, e.clientCaps)
	e.correctionBackspaces += n
	if e.conn != nil {
		for i := 0; i < n; i++ {
			e.ForwardKeyEvent(IBusKeyBackSpace, e.bsKeycode, 0)
		}
	}
}

// eraseForwarded removes n already-forwarded runes before the cursor.
//
// This always uses synthetic BackSpace key events, never
// DeleteSurroundingText: several clients (Chrome/Docs, Zalo, terminals)
// advertise the surrounding-text capability yet silently ignore the delete
// request, leaving the forwarded prefix behind and duplicating text.
// BackSpace is an ordinary key event every client honors, so the correction
// always lands. Terminals never reach this path anyway: without a
// surrounding-text push they stay on the legacy preedit path.
func (e *NepKeyEngine) eraseForwarded(n int) {
	e.sendBackspaces(n)
}

// switchToComposing erases the eagerly forwarded prefix from the client so the
// whole word can be redrawn as preedit. No-op when nothing was forwarded.
func (e *NepKeyEngine) switchToComposing() {
	if e.forwardedLen > 0 {
		e.hidePreedit()
		e.eraseForwarded(e.forwardedLen)
		e.forwardedLen = 0
	}
}

func (e *NepKeyEngine) commitBuffer() {
	if e.eng.HasBuffer() {
		tail := e.unforwarded()
		e.hidePreedit()
		if tail != "" {
			e.commit(tail)
		}
		e.resetBufferState()
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
	e.dbgf("KEY keyval=%d keycode=%d state=%d caps=%d purpose=%d eng=%s buf=%q fwd=%d",
		keyval, keycode, state, e.clientCaps, e.inputPurpose, e.engineName, e.eng.GetWordBuffer(), e.forwardedLen)

	// 1. Ignore key release events
	if (state & IBusReleaseMask) != 0 {
		return false, nil
	}

	// 2. Bypass if Vietnamese input is disabled or inside a password/PIN field
	if !e.enabled || e.inputPurpose == IBusPurposePassword || e.inputPurpose == IBusPurposePin || (e.inputHints&IBusHintHiddenText) != 0 {
		return false, nil
	}

	// 3. Shortcuts (Ctrl, Alt, Super held down with normal keys): commit buffer & pass through.
	// The configured V/E toggle (default Ctrl+Space) additionally swaps the
	// active engine so the top-bar indicator flips between V and E.
	if (state & (IBusControlMask | IBusMod1Mask | IBusSuperMask)) != 0 {
		e.setRawMode(false)
		e.atWordBoundary = true
		if e.matchToggle(state, keyval) {
			e.commitBuffer()
			e.switchEngine()
			return true, nil
		}
		e.commitBuffer()
		return false, nil
	}

	// 4. Special Keysyms (>= 0xff00): BackSpace, Return, Tab, Arrows, Shift, Alt, Ctrl, F1-F12
	if keyval >= 0xff00 {
		switch keyval {
		case IBusKeyBackSpace:
			if keycode != 0 {
				e.bsKeycode = keycode
			}
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
				if e.forwardedLen > 0 {
					// Fast-path: the character is already in the client.
					// Mirror the deletion internally and let the client delete it.
					e.eng.OnBackspace()
					e.forwardedLen--
					if e.forwardedLen < 0 {
						e.forwardedLen = 0
					}
					if !e.eng.HasBuffer() {
						e.atWordBoundary = true
					}
					return false, nil
				}
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
				if act := e.eng.CheckMacro(0); act != nil {
					// Macro replacement covers the whole word: erase the
					// eagerly forwarded prefix first to avoid duplication.
					e.hidePreedit()
					e.eraseForwarded(e.forwardedLen)
					e.commit(string(act.AppendRunes))
					e.resetBufferState()
				} else {
					text := e.unforwarded()
					e.hidePreedit()
					e.commit(text)
					e.resetBufferState()
				}
				// Single-press Enter: committing text AND returning false allows
				// Web Chat, Zalo, Terminal, IDE và Chrome Omnibox search/submit
				// trong 1 lần bấm. Tới suggestion cụ thể thì Tab/Down/click
				// chọn trước rồi Enter (đúng native Chrome).
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
					e.eraseForwarded(e.forwardedLen)
					e.commit(string(runes))
					e.resetBufferState()
					return true, nil
				}
				text := e.unforwarded()
				e.hidePreedit()
				e.commit(text)
				e.resetBufferState()
				return false, nil
			}
			return false, nil

		case IBusKeyEscape:
			e.setRawMode(false)
			e.atWordBoundary = true
			if e.eng.HasBuffer() {
				text := e.unforwarded()
				e.hidePreedit()
				e.commit(text)
				e.resetBufferState()
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
			if act := e.eng.CheckMacro(' '); act != nil {
				e.hidePreedit()
				e.eraseForwarded(e.forwardedLen)
				e.commit(string(act.AppendRunes))
			} else {
				text := e.unforwarded() + " "
				e.hidePreedit()
				e.commit(text)
			}
			e.resetBufferState()
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
				e.switchToComposing()
				e.updatePreedit(buf)
				return true, nil
			}
			e.hidePreedit()
			return false, nil
		}
		e.atWordBoundary = false
		if e.eng.HasBuffer() {
			// Commit only what the client has not received yet: the
			// forwarded prefix is already on screen (e.g. "g" in "g6").
			text := e.unforwarded()
			e.hidePreedit()
			e.commit(text)
			e.resetBufferState()
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
			if act := e.eng.CheckMacro(r); act != nil {
				e.hidePreedit()
				e.eraseForwarded(e.forwardedLen)
				e.commit(string(act.AppendRunes))
			} else {
				text := e.unforwarded() + string(r)
				e.hidePreedit()
				e.commit(text)
			}
			e.resetBufferState()
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
	oldLen := len([]rune(e.eng.GetWordBuffer()))
	wasEmpty := oldLen == 0
	_ = e.eng.ProcessKey(r)
	e.latchWordMode(wasEmpty)
	buf := e.eng.GetWordBuffer()
	if buf == "" {
		// Bamboo produced nothing: buffer is empty so there is no prefix
		// left to correct later; drop stale fast-path accounting.
		e.hidePreedit()
		e.forwardedLen = 0
		return false, nil
	}
	// ASCII fast-path (Chrome Omnibox Enter/Tab): a plain ASCII append means no
	// Vietnamese composition is in progress, so forward the keystroke at once and
	// never enter IME-composing state. The buffer is still tracked for tone keys.
	// Only for clients with surrounding-text (Chrome, GTK entries): terminals
	// (VTE) neither need it nor handle the synthetic correction reliably,
	// so they stay on the legacy preedit path. The decision is latched at
	// word start because caps may arrive late after focus changes.
	newRunes := []rune(buf)
	if e.wordFastPath && isPureASCII(buf) && len(newRunes) == oldLen+1 && newRunes[len(newRunes)-1] == r {
		// Leaving composing state (e.g. Telex tone cancel "te"+"r"->"ter"):
		// the erased prefix is not in the client, so re-emit everything
		// except the current keystroke (forwarded by returning false).
		// Without this the client ends up with "rminal" instead of "terminal".
		e.hidePreedit()
		if oldLen > 0 && e.forwardedLen == 0 {
			e.restoreCommits++
			e.dbgf("RESTORE buf=%q", string(newRunes[:len(newRunes)-1]))
			e.commit(string(newRunes[:len(newRunes)-1]))
		}
		e.dbgf("FAST-FWD buf=%q fwd=%d", buf, len(newRunes))
		e.forwardedLen = len(newRunes)
		return false, nil
	}
	e.dbgf("COMPOSE buf=%q fwd=%d", buf, e.forwardedLen)
	e.switchToComposing()
	e.updatePreedit(buf)
	return true, nil
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
	// New focus may be a different client (e.g. terminal after browser):
	// drop the previous surrounding proof and ask the client to push its
	// buffer. Terminals never answer, so the first word stays legacy-safe
	// until a real text field (Chrome/GTK) proves itself.
	e.hasSurrounding = false
	if e.asciiFastPath && e.conn != nil {
		e.RequireSurroundingText()
	}
	e.dbgf("FOCUS-IN eng=%s caps=%d", e.engineName, e.clientCaps)
	// Resolve now so the first word after an app switch is already classified,
	// then re-check shortly after in case the window manager updated late.
	e.focusClass.Store(queryFocusClass())
	seq := e.focusSeq.Add(1)
	go e.captureFocusClass(seq)
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
	e.dbgf("FOCUS-OUT eng=%s buf=%q fwd=%d", e.engineName, e.eng.GetWordBuffer(), e.forwardedLen)
	e.commitBuffer()
	return nil
}

// SetContentType receives input purpose and hints from GNOME Shell / GTK / Wayland clients
// //@method(in_signature="uu")
func (e *NepKeyEngine) SetContentType(purpose uint32, hints uint32) *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.inputPurpose = purpose
	e.inputHints = hints
	e.dbgf("CONTENT purpose=%d hints=%d", purpose, hints)
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

// otherEngineName returns the sibling engine to switch to on toggle.
func (e *NepKeyEngine) otherEngineName() string {
	if e.engineName == "nepkey-en" {
		return "nepkey"
	}
	return "nepkey-en"
}

// matchToggle reports whether this shortcut is the configured V/E toggle
// ("Ctrl+Space", "Alt+Shift" or "Ctrl+Shift").
func (e *NepKeyEngine) matchToggle(state uint32, keyval uint32) bool {
	toggle := e.cfg.ToggleKey
	hasCtrl := state&IBusControlMask != 0
	hasAlt := state&IBusMod1Mask != 0
	isShift := keyval == IBusKeyShiftL || keyval == IBusKeyShiftR
	switch toggle {
	case "Alt+Shift":
		return hasAlt && isShift
	case "Ctrl+Shift":
		return hasCtrl && isShift
	default: // "Ctrl+Space"
		return hasCtrl && keyval == IBusKeySpace
	}
}

// switchEngine asks IBus to activate the sibling engine so the top-bar
// indicator flips between V (nepkey-v icon) and E (nepkey-e icon).
// With nil conn (unit tests) only the observable request is recorded.
func (e *NepKeyEngine) switchEngine() {
	target := e.otherEngineName()
	e.switchRequests = append(e.switchRequests, target)
	if e.conn != nil {
		obj := e.conn.Object("org.freedesktop.IBus", "/org/freedesktop/IBus")
		obj.Call("org.freedesktop.IBus.SetGlobalEngine", 0, target)
	}
}

// SetSurroundingText receives the client's buffer push (in response to
// RequireSurroundingText or automatic on focus). Any push — even an empty
// buffer from an empty field — proves the client is a real text field whose
// key-event corrections will be honored. Terminals (VTE/Ptyxis) return FALSE
// for retrieve-surrounding and never push, so hasSurrounding stays false
// and they keep the legacy preedit path.
func (e *NepKeyEngine) SetSurroundingText(text dbus.Variant, cursor uint32, anchor uint32) *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.hasSurrounding = true
	e.dbgf("SURROUNDING cursor=%d anchor=%d caps=%d", cursor, anchor, e.clientCaps)
	return nil
}

// SetCapabilities records what the focused client supports (surrounding
// text decides the correction mechanism) and reports engine capabilities.
func (e *NepKeyEngine) SetCapabilities(cap uint32) *dbus.Error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.clientCaps = cap
	e.dbgf("CAPS caps=%d eng=%s", cap, e.engineName)
	return nil
}
