package ibus

import (
	"testing"

	"nepkey/pkg/config"
)

func newFastPathEngine() *NepKeyEngine {
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"
	cfg.AsciiFastPath = true
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)
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
	if e.correctionBackspaces != 4 {
		t.Fatalf("expected 4 correction backspaces, got %d", e.correctionBackspaces)
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
	if e.correctionBackspaces != 4 {
		t.Fatalf("expected 4 correction backspaces, got %d", e.correctionBackspaces)
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
	if e.correctionBackspaces != 2 {
		t.Fatalf("expected 2 correction backspaces for forwarded 'vn', got %d", e.correctionBackspaces)
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
