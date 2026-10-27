package ibus

import (
	"testing"

	"github.com/godbus/dbus"
	"github.com/sarim/goibus/ibus"
	"github.com/todo-laterx/nepkey/pkg/config"
)

// pushSurrounding simulates a supporting client (Chrome/GTK) answering the
// engine's RequireSurroundingText: any push — even an empty buffer — proves
// DeleteSurroundingText will work. Terminals (VTE/Ptyxis) never push.
func pushSurrounding(e *NepKeyEngine) {
	e.SetSurroundingText(dbus.MakeVariant(*ibus.NewText("")), 0, 0)
}

func newFastPathEngine() *NepKeyEngine {
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"
	cfg.AsciiFastPath = true
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
	// Simulate a Chrome/GTK client: capability bit + actual surrounding push
	// together arm the fast-path (terminals may set the bit but never push).
	e.SetCapabilities(IBusCapSurroundingText)
	pushSurrounding(e)
	e.eng.SetMacros(map[string]string{"vn": "Việt Nam"}, true)
	return e
}

func typeRunes(e *NepKeyEngine, s string) []bool {
	var out []bool
	for _, ch := range s {
		handled, _ := e.ProcessKeyEvent(uint32(ch), 0, 0)
		out = append(out, handled)
	}
	return out
}

func TestAsciiFastPath_ForwardPureASCII(t *testing.T) {
	e := newFastPathEngine()
	for i, h := range typeRunes(e, "you") {
		if h {
			t.Fatalf("char %d: expected forward (false), got true", i)
		}
	}
	if e.eng.GetWordBuffer() != "you" {
		t.Fatalf("expected buffer 'you', got %q", e.eng.GetWordBuffer())
	}
	if e.forwardedLen != 3 {
		t.Fatalf("expected forwardedLen 3, got %d", e.forwardedLen)
	}
	if e.correctionBackspaces != 0 {
		t.Fatalf("expected 0 correction backspaces, got %d", e.correctionBackspaces)
	}
	// Enter must reach the client so the Omnibox navigates in one press.
	if handled, _ := e.ProcessKeyEvent(IBusKeyReturn, 36, 0); handled {
		t.Fatalf("expected Return to return false")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected empty buffer after Return, got %q", e.eng.GetWordBuffer())
	}
	// Tab likewise must reach the client to accept the suggestion.
	typeRunes(e, "you")
	if handled, _ := e.ProcessKeyEvent(IBusKeyTab, 0, 0); handled {
		t.Fatalf("expected Tab to return false")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected empty buffer after Tab, got %q", e.eng.GetWordBuffer())
	}
}

func TestAsciiFastPath_TransitionToComposing(t *testing.T) {
	e := newFastPathEngine()
	for _, h := range typeRunes(e, "cach") {
		if h {
			t.Fatalf("expected 'cach' to pass through")
		}
	}
	handled, _ := e.ProcessKeyEvent(uint32('s'), 0, 0)
	if !handled {
		t.Fatalf("expected tone key 's' to engage preedit (true)")
	}
	if e.eng.GetWordBuffer() != "cách" {
		t.Fatalf("expected buffer 'cách', got %q", e.eng.GetWordBuffer())
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen reset to 0, got %d", e.forwardedLen)
	}
	if e.correctionDeletes != 4 {
		t.Fatalf("expected 4 surrounding deletions, got %d", e.correctionDeletes)
	}
	if e.correctionBackspaces != 0 {
		t.Fatalf("expected 0 synthetic backspaces on surrounding path, got %d", e.correctionBackspaces)
	}
	// Enter submits the composed Vietnamese word.
	if handled, _ := e.ProcessKeyEvent(IBusKeyReturn, 36, 0); handled {
		t.Fatalf("expected Return to return false")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected empty buffer after Return, got %q", e.eng.GetWordBuffer())
	}
}

func TestAsciiFastPath_VietnameseFlow(t *testing.T) {
	e := newFastPathEngine()
	var handled []bool
	for _, ch := range "chaof" {
		h, _ := e.ProcessKeyEvent(uint32(ch), 0, 0)
		handled = append(handled, h)
	}
	for i := 0; i < 4; i++ {
		if handled[i] {
			t.Fatalf("char %d: expected forward (false)", i)
		}
	}
	if !handled[4] {
		t.Fatalf("expected 'f' (tone) to engage preedit (true)")
	}
	if e.eng.GetWordBuffer() != "chào" {
		t.Fatalf("expected buffer 'chào', got %q", e.eng.GetWordBuffer())
	}
	if e.correctionDeletes != 4 {
		t.Fatalf("expected 4 surrounding deletions, got %d", e.correctionDeletes)
	}
	if e.correctionBackspaces != 0 {
		t.Fatalf("expected 0 synthetic backspaces on surrounding path, got %d", e.correctionBackspaces)
	}
}

func TestAsciiFastPath_BackspaceSync(t *testing.T) {
	e := newFastPathEngine()
	typeRunes(e, "yo")
	if handled, _ := e.ProcessKeyEvent(IBusKeyBackSpace, EvdevKeyBackSpace, 0); handled {
		t.Fatalf("expected Backspace to forward (false) in fast-path")
	}
	if e.eng.GetWordBuffer() != "y" || e.forwardedLen != 1 {
		t.Fatalf("expected buffer 'y'/forwardedLen 1, got %q/%d", e.eng.GetWordBuffer(), e.forwardedLen)
	}
	if handled, _ := e.ProcessKeyEvent(IBusKeyBackSpace, EvdevKeyBackSpace, 0); handled {
		t.Fatalf("expected Backspace to forward (false) in fast-path")
	}
	if e.eng.HasBuffer() || e.forwardedLen != 0 {
		t.Fatalf("expected empty buffer, got %q/%d", e.eng.GetWordBuffer(), e.forwardedLen)
	}
}

func TestAsciiFastPath_MacroForwarded(t *testing.T) {
	e := newFastPathEngine()
	typeRunes(e, "vn")
	if handled, _ := e.ProcessKeyEvent(IBusKeySpace, 0, 0); !handled {
		t.Fatalf("expected Space with macro to return true")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected empty buffer after macro, got %q", e.eng.GetWordBuffer())
	}
	if e.correctionDeletes != 2 {
		t.Fatalf("expected 2 surrounding deletions for forwarded 'vn', got %d", e.correctionDeletes)
	}
}

func TestAsciiFastPath_DisabledByDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.AsciiFastPath {
		t.Fatalf("AsciiFastPath must default to false")
	}
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
	for i, h := range typeRunes(e, "you") {
		if !h {
			t.Fatalf("char %d: legacy path must consume into preedit (true)", i)
		}
	}
}

func TestAsciiFastPath_SurroundingTextCorrection(t *testing.T) {
	e := newFastPathEngine()
	// Simulate a Chrome/GTK client advertising surrounding-text support.
	e.SetCapabilities(IBusCapSurroundingText)
	for _, h := range typeRunes(e, "cach") {
		if h {
			t.Fatalf("expected forward, got true")
		}
	}
	if handled, _ := e.ProcessKeyEvent(uint32('s'), 0, 0); !handled {
		t.Fatalf("expected tone key to engage preedit (true)")
	}
	if e.eng.GetWordBuffer() != "cách" {
		t.Fatalf("expected buffer 'cách', got %q", e.eng.GetWordBuffer())
	}
	if e.correctionDeletes != 4 {
		t.Fatalf("expected 4 surrounding deletions, got %d", e.correctionDeletes)
	}
	if e.correctionBackspaces != 0 {
		t.Fatalf("expected 0 synthetic backspaces, got %d", e.correctionBackspaces)
	}
}

func TestAsciiFastPath_TerminalStaysOnLegacyPreedit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"
	cfg.AsciiFastPath = true
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
	e.SetCapabilities(0) // VTE terminals advertise no surrounding-text
	// 'plugin' has no Telex tone keys, so the legacy buffer mirrors the keys.
	for i, h := range typeRunes(e, "plugin") {
		if !h {
			t.Fatalf("char %d: terminal must stay on consuming preedit path (true)", i)
		}
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen 0 on legacy path, got %d", e.forwardedLen)
	}
	if e.eng.GetWordBuffer() != "plugin" {
		t.Fatalf("expected buffer 'plugin', got %q", e.eng.GetWordBuffer())
	}
	if e.correctionBackspaces != 0 || e.correctionDeletes != 0 {
		t.Fatalf("expected no correction on legacy path, got bs=%d del=%d", e.correctionBackspaces, e.correctionDeletes)
	}
}

func TestAsciiFastPath_ClaimsSurroundingButNeverPushesStaysLegacy(t *testing.T) {
	// Ptyxis/VTE on Wayland: advertises the surrounding-text capability bit
	// but never pushes surrounding text and ignores DeleteSurroundingText.
	// Forward-then-erase would leave ghosts and visible jumping on the PTY,
	// so the fast-path must stay off (legacy preedit) without the proof.
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"
	cfg.AsciiFastPath = true
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
	e.SetCapabilities(IBusCapSurroundingText)
	// NOTE: no pushSurrounding — like a real terminal.
	for i, h := range typeRunes(e, "plugin") {
		if !h {
			t.Fatalf("char %d: terminal-like client must stay on consuming preedit path (true)", i)
		}
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen 0 on legacy path, got %d", e.forwardedLen)
	}
	if e.eng.GetWordBuffer() != "plugin" {
		t.Fatalf("expected buffer 'plugin', got %q", e.eng.GetWordBuffer())
	}
	if e.correctionBackspaces != 0 || e.correctionDeletes != 0 {
		t.Fatalf("expected no correction on legacy path, got bs=%d del=%d", e.correctionBackspaces, e.correctionDeletes)
	}
}

func TestAsciiFastPath_TerminalToneStaysInPreeditWithoutErasure(t *testing.T) {
	// Same terminal-like client typing Vietnamese: the tone key must compose
	// inside preedit without any eager erasure (no ghosts).
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"
	cfg.AsciiFastPath = true
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
	e.SetCapabilities(IBusCapSurroundingText)
	// NOTE: no pushSurrounding — like a real terminal.
	for i, h := range typeRunes(e, "cach") {
		if !h {
			t.Fatalf("char %d: legacy path must consume (true)", i)
		}
	}
	if handled, _ := e.ProcessKeyEvent(uint32('s'), 0, 0); !handled {
		t.Fatalf("expected tone key consumed (true)")
	}
	if e.eng.GetWordBuffer() != "cách" {
		t.Fatalf("expected buffer 'cách', got %q", e.eng.GetWordBuffer())
	}
	if e.correctionDeletes != 0 || e.correctionBackspaces != 0 {
		t.Fatalf("expected no eager erasure, got del=%d bs=%d", e.correctionDeletes, e.correctionBackspaces)
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen 0, got %d", e.forwardedLen)
	}
}

func TestAsciiFastPath_FocusInResetsSurroundingProof(t *testing.T) {
	// Switching focus (browser -> terminal) must drop the previous proof:
	// the first word on the new client stays legacy until it proves support.
	e := newFastPathEngine()
	e.FocusIn()
	if e.hasSurrounding {
		t.Fatalf("expected surrounding proof reset on FocusIn")
	}
	for i, h := range typeRunes(e, "hi") {
		if !h {
			t.Fatalf("char %d: unproven client must stay legacy (true)", i)
		}
	}
	if handled, _ := e.ProcessKeyEvent(IBusKeySpace, 0, 0); !handled {
		t.Fatalf("expected Space consumed (true)")
	}
	// The new client proves support -> fast-path returns.
	pushSurrounding(e)
	for i, h := range typeRunes(e, "hi") {
		if h {
			t.Fatalf("char %d: proven client must forward (false)", i)
		}
	}
}

func TestAsciiFastPath_NoCapsFallsBackToBackspace(t *testing.T) {
	e := newFastPathEngine()
	e.SetCapabilities(0)
	// Without surrounding-text the gate keeps the legacy preedit path:
	// every key is consumed and no eager correction is ever requested.
	for i, h := range typeRunes(e, "cach") {
		if !h {
			t.Fatalf("char %d: legacy path must consume (true)", i)
		}
	}
	if handled, _ := e.ProcessKeyEvent(uint32('s'), 0, 0); !handled {
		t.Fatalf("expected tone key consumed (true)")
	}
	if e.eng.GetWordBuffer() != "cách" {
		t.Fatalf("expected buffer 'cách', got %q", e.eng.GetWordBuffer())
	}
	if e.correctionDeletes != 0 || e.correctionBackspaces != 0 {
		t.Fatalf("expected no eager correction, got del=%d bs=%d", e.correctionDeletes, e.correctionBackspaces)
	}
}

func TestAsciiFastPath_CapsFlapMidWordStaysLegacy(t *testing.T) {
	// Reproduces the Antigravity log: first key arrives with caps=9
	// (no surrounding), the client reports caps=41 only afterwards.
	// The word started legacy must stay legacy: no eager erasure.
	e := newFastPathEngine()
	e.SetCapabilities(0)
	for i, h := range typeRunes(e, "te") {
		if !h {
			t.Fatalf("char %d: expected consume (true)", i)
		}
	}
	e.SetCapabilities(IBusCapSurroundingText) // late arrival, mid-word
	if handled, _ := e.ProcessKeyEvent(uint32('r'), 0, 0); !handled {
		t.Fatalf("expected tone key consumed (true)")
	}
	if e.eng.GetWordBuffer() != "tẻ" {
		t.Fatalf("expected buffer 'tẻ', got %q", e.eng.GetWordBuffer())
	}
	if e.correctionDeletes != 0 || e.correctionBackspaces != 0 {
		t.Fatalf("expected no eager erasure, got del=%d bs=%d", e.correctionDeletes, e.correctionBackspaces)
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen 0, got %d", e.forwardedLen)
	}
}

func TestAsciiFastPath_CapsLostMidWordUsesLiveCaps(t *testing.T) {
	// Word starts fast (caps present), caps drop before the tone key:
	// the erasure mechanism follows the live caps (BackSpace fallback).
	e := newFastPathEngine()
	typeRunes(e, "te")
	e.SetCapabilities(0)
	if handled, _ := e.ProcessKeyEvent(uint32('r'), 0, 0); !handled {
		t.Fatalf("expected tone key to engage composing (true)")
	}
	if e.eng.GetWordBuffer() != "tẻ" {
		t.Fatalf("expected buffer 'tẻ', got %q", e.eng.GetWordBuffer())
	}
	if e.correctionBackspaces != 2 {
		t.Fatalf("expected 2 fallback backspaces, got %d", e.correctionBackspaces)
	}
	if e.correctionDeletes != 0 {
		t.Fatalf("expected 0 surrounding deletions, got %d", e.correctionDeletes)
	}
}

func TestAsciiFastPath_ToneCancelHidesPreedit(t *testing.T) {
	// Telex tone cancel: "tẻ" + "r" -> "ter". The abandoned preedit must
	// be hidden or the client keeps rendering the ghost alongside "ter".
	e := newFastPathEngine()
	typeRunes(e, "te")
	hidesBefore := e.preeditHides
	if handled, _ := e.ProcessKeyEvent(uint32('r'), 0, 0); !handled {
		t.Fatalf("expected tone key consumed (true)")
	}
	if e.eng.GetWordBuffer() != "tẻ" {
		t.Fatalf("expected buffer 'tẻ', got %q", e.eng.GetWordBuffer())
	}
	if handled, _ := e.ProcessKeyEvent(uint32('r'), 0, 0); handled {
		t.Fatalf("expected cancel key forwarded (false)")
	}
	if e.eng.GetWordBuffer() != "ter" {
		t.Fatalf("expected buffer 'ter', got %q", e.eng.GetWordBuffer())
	}
	if e.forwardedLen != 3 {
		t.Fatalf("expected forwardedLen 3, got %d", e.forwardedLen)
	}
	if e.preeditHides <= hidesBefore {
		t.Fatalf("expected a preedit hide on composing->forward transition")
	}
}

func TestAsciiFastPath_ToneCancelRestoresPrefix(t *testing.T) {
	// Typing an English word in Telex mode: "te" + "r" -> "tẻ" (erased),
	// second "r" cancels the tone -> "ter". The client must end up with
	// the full word, not "rminal".
	e := newFastPathEngine()
	typeRunes(e, "te")
	if handled, _ := e.ProcessKeyEvent(uint32('r'), 0, 0); !handled {
		t.Fatalf("expected tone key consumed (true)")
	}
	if e.eng.GetWordBuffer() != "tẻ" {
		t.Fatalf("expected buffer 'tẻ', got %q", e.eng.GetWordBuffer())
	}
	if handled, _ := e.ProcessKeyEvent(uint32('r'), 0, 0); handled {
		t.Fatalf("expected cancel key forwarded (false)")
	}
	if e.eng.GetWordBuffer() != "ter" {
		t.Fatalf("expected buffer 'ter', got %q", e.eng.GetWordBuffer())
	}
	if e.forwardedLen != 3 {
		t.Fatalf("expected forwardedLen 3, got %d", e.forwardedLen)
	}
	if e.restoreCommits != 1 {
		t.Fatalf("expected 1 prefix restore, got %d", e.restoreCommits)
	}
}

func TestAsciiFastPath_TelexDigitCommitsSuffixOnly(t *testing.T) {
	// "te" forwarded, then Telex digit "1" (word break): the client already
	// holds "te", so nothing must be committed again (else "tete").
	e := newFastPathEngine()
	typeRunes(e, "te")
	if handled, _ := e.ProcessKeyEvent(uint32('1'), 0, 0); handled {
		t.Fatalf("expected digit forwarded (false)")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected empty buffer, got %q", e.eng.GetWordBuffer())
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen 0, got %d", e.forwardedLen)
	}
	if len(e.commitLog) != 0 {
		t.Fatalf("expected no commit, got %q", e.commitLog)
	}
}

func TestAsciiFastPath_LegacyDigitCommitsFullWord(t *testing.T) {
	// Same keystrokes on the legacy path: nothing forwarded, so the full
	// word must be committed at the digit boundary.
	cfg := config.DefaultConfig()
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
	e.SetCapabilities(0)
	typeRunes(e, "te")
	if handled, _ := e.ProcessKeyEvent(uint32('1'), 0, 0); handled {
		t.Fatalf("expected digit forwarded (false)")
	}
	if len(e.commitLog) != 1 || e.commitLog[0] != "te" {
		t.Fatalf("expected one commit of 'te', got %q", e.commitLog)
	}
}

func TestAsciiFastPath_EscapeCommitsSuffixOnly(t *testing.T) {
	e := newFastPathEngine()
	typeRunes(e, "te")
	if handled, _ := e.ProcessKeyEvent(IBusKeyEscape, 0, 0); !handled {
		t.Fatalf("expected Escape consumed (true)")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected empty buffer, got %q", e.eng.GetWordBuffer())
	}
	if len(e.commitLog) != 0 {
		t.Fatalf("expected no commit (prefix already forwarded), got %q", e.commitLog)
	}
}

func TestAsciiFastPath_VNIDigitStartStaysLegacy(t *testing.T) {
	// VNI tone digits are consumed, never forwarded: a digit-started word
	// must stay on the legacy path with zero fast-path accounting.
	cfg := config.DefaultConfig()
	cfg.InputMethod = "VNI"
	cfg.AsciiFastPath = true
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
	e.SetCapabilities(IBusCapSurroundingText)
	for _, ch := range "1a" {
		if handled, _ := e.ProcessKeyEvent(uint32(ch), 0, 0); !handled {
			t.Fatalf("char %q: expected consume (true)", string(ch))
		}
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen 0, got %d", e.forwardedLen)
	}
	if e.correctionDeletes != 0 || e.correctionBackspaces != 0 {
		t.Fatalf("expected no correction, got del=%d bs=%d", e.correctionDeletes, e.correctionBackspaces)
	}
}

func TestToggle_CtrlSpaceSwitchesEngine(t *testing.T) {
	e := newFastPathEngine()
	e.engineName = "nepkey"
	typeRunes(e, "chao")
	h, _ := e.ProcessKeyEvent(IBusKeySpace, 0, IBusControlMask)
	if !h {
		t.Fatalf("expected toggle consumed (true)")
	}
	if len(e.switchRequests) != 1 || e.switchRequests[0] != "nepkey-en" {
		t.Fatalf("expected switch to nepkey-en, got %q", e.switchRequests)
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer committed on toggle, got %q", e.eng.GetWordBuffer())
	}
}

func TestToggle_EngineEnSwitchesBack(t *testing.T) {
	e := newFastPathEngine()
	e.engineName = "nepkey-en"
	h, _ := e.ProcessKeyEvent(IBusKeySpace, 0, IBusControlMask)
	if !h {
		t.Fatalf("expected toggle consumed (true)")
	}
	if len(e.switchRequests) != 1 || e.switchRequests[0] != "nepkey" {
		t.Fatalf("expected switch to nepkey, got %q", e.switchRequests)
	}
}

func TestToggle_NonToggleShortcutPassesThrough(t *testing.T) {
	e := newFastPathEngine()
	e.engineName = "nepkey"
	h, _ := e.ProcessKeyEvent(uint32('a'), 0, IBusControlMask) // Ctrl+A
	if h {
		t.Fatalf("expected Ctrl+A passthrough (false)")
	}
	if len(e.switchRequests) != 0 {
		t.Fatalf("expected no switch, got %q", e.switchRequests)
	}
}

func TestToggle_AltShiftMatches(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ToggleKey = "Alt+Shift"
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
	e.engineName = "nepkey"
	h, _ := e.ProcessKeyEvent(IBusKeyShiftL, 0, IBusMod1Mask)
	if !h {
		t.Fatalf("expected Alt+Shift consumed (true)")
	}
	if len(e.switchRequests) != 1 || e.switchRequests[0] != "nepkey-en" {
		t.Fatalf("expected switch to nepkey-en, got %q", e.switchRequests)
	}
}

func TestEngineEnabledByDefault(t *testing.T) {
	if !engineEnabledByDefault("nepkey") {
		t.Fatalf("nepkey must start enabled")
	}
	if engineEnabledByDefault("nepkey-en") {
		t.Fatalf("nepkey-en must start disabled")
	}
}

func TestFastPathBlocklist_MatchesX11Class(t *testing.T) {
	e := newFastPathEngine()
	cases := map[string]bool{
		"":                   false,
		"zalo":               true,
		"Zalo":               true,
		"google-chrome":      false,
		"org.gnome.Nautilus": false,
		"wine":               true,
		"Wine-Proton":        true,
	}
	for class, want := range cases {
		if got := e.blockedClass(class); got != want {
			t.Fatalf("blockedClass(%q) = %v, want %v", class, got, want)
		}
	}
}

func TestFastPathBlocklist_BlockedAppStaysLegacy(t *testing.T) {
	e := newFastPathEngine()
	e.focusClass.Store("zalo")
	for i, h := range typeRunes(e, "you") {
		if !h {
			t.Fatalf("char %d: blocked app must stay on consuming preedit (true)", i)
		}
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen 0, got %d", e.forwardedLen)
	}
	if e.correctionDeletes != 0 || e.correctionBackspaces != 0 {
		t.Fatalf("expected no correction, got del=%d bs=%d", e.correctionDeletes, e.correctionBackspaces)
	}
}

func TestFastPathBlocklist_TerminalClassesBlocked(t *testing.T) {
	e := newFastPathEngine()
	for _, class := range []string{
		"ptyxis", "org.gnome.Ptyxis", "gnome-terminal-server",
		"Alacritty", "kitty", "wezterm", "foot",
		"konsole", "xterm", "tilix", "terminator",
	} {
		if !e.blockedClass(class) {
			t.Fatalf("expected terminal class %q to be blocked", class)
		}
	}
	// Even with capability bit + surrounding proof, a terminal stays legacy.
	e.focusClass.Store("ptyxis")
	for i, h := range typeRunes(e, "you") {
		if !h {
			t.Fatalf("char %d: blocked terminal must stay on consuming preedit (true)", i)
		}
	}
	if e.forwardedLen != 0 {
		t.Fatalf("expected forwardedLen 0, got %d", e.forwardedLen)
	}
}

func TestFastPathBlocklist_AllowedAppUsesFastPath(t *testing.T) {
	e := newFastPathEngine()
	e.focusClass.Store("google-chrome")
	for i, h := range typeRunes(e, "you") {
		if h {
			t.Fatalf("char %d: allowed app must forward (false)", i)
		}
	}
	if e.forwardedLen != 3 {
		t.Fatalf("expected forwardedLen 3, got %d", e.forwardedLen)
	}
}
