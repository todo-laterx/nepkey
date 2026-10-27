package ibus

import (
	"testing"

	"github.com/todo-laterx/nepkey/pkg/config"
)

// P12 must not lose P: Telex digits are word-breaks, buffer commits then digit passes.
func TestDigitAfterLetterTelex(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test-digit", cfg)

	if handled, _ := e.ProcessKeyEvent(uint32('P'), 0, 0); !handled {
		t.Fatalf("expected 'P' to be handled as preedit")
	}
	if handled, _ := e.ProcessKeyEvent(uint32('1'), 0, 0); handled {
		t.Fatalf("expected '1' after 'P' to pass through (handled=false)")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer empty after digit, got %q", e.eng.GetWordBuffer())
	}
	if handled, _ := e.ProcessKeyEvent(uint32('2'), 0, 0); handled {
		t.Fatalf("expected '2' to pass through")
	}
}

// VNI digits are tone keys and must keep composing.
func TestDigitAsToneVNI(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InputMethod = "VNI"
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test-vni", cfg)

	if handled, _ := e.ProcessKeyEvent(uint32('a'), 0, 0); !handled {
		t.Fatalf("expected 'a' to be handled")
	}
	if handled, _ := e.ProcessKeyEvent(uint32('1'), 0, 0); !handled {
		t.Fatalf("expected '1' in VNI to be handled as tone mark")
	}
	if got := e.eng.GetWordBuffer(); got != "\u00e1" {
		t.Fatalf("expected buffer 'á', got %q", got)
	}
}

// Enter luôn forward (single-press submit/search).
// Tới suggestion cụ thể thì Tab/Down/click chọn trước rồi Enter.
func TestEnterAsciiVsVietnamese(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"

	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test-enter", cfg)
	for _, ch := range "you" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if handled, _ := e.ProcessKeyEvent(IBusKeyReturn, 36, 0); handled {
		t.Fatalf("expected Enter after 'you' to pass through (handled=false)")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer empty after Enter, got %q", e.eng.GetWordBuffer())
	}

	e2 := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test-enter-vi", cfg)
	for _, ch := range "chaof" {
		e2.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if got := e2.eng.GetWordBuffer(); got != "ch\u00e0o" {
		t.Fatalf("expected buffer 'chào', got %q", got)
	}
	if handled, _ := e2.ProcessKeyEvent(IBusKeyReturn, 36, 0); handled {
		t.Fatalf("expected Enter after Vietnamese to pass through (single-press submit)")
	}
}
