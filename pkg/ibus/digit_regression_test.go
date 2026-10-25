package ibus

import (
	"testing"

	"nepkey/pkg/config"
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
