package ibus

import (
	"testing"

	"nepkey/pkg/config"
)

func TestNepKeyEngine_TypingWorkflow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"

	// Create engine with nil conn (mocking ProcessKeyEvent logic directly)
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)

	// Test "lai" + "j" -> "lại"
	for _, ch := range "laij" {
		handled, _ := e.ProcessKeyEvent(uint32(ch), 0, 0)
		if !handled {
			t.Fatalf("expected character %q to be handled in zero-underline preedit mode", string(ch))
		}
	}
	if e.eng.GetWordBuffer() != "lại" {
		t.Fatalf("expected buffer 'lại', got %q", e.eng.GetWordBuffer())
	}
	e.ProcessKeyEvent(IBusKeySpace, 0, 0)

	// Test "chuwx" -> "chữ"
	for _, ch := range "chuwx" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "chữ" {
		t.Fatalf("expected buffer 'chữ', got %q", e.eng.GetWordBuffer())
	}
	e.ProcessKeyEvent(IBusKeySpace, 0, 0)

	// Type "tieengs" -> "tiếng"
	for _, ch := range "tieengs" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "tiếng" {
		t.Fatalf("expected buffer 'tiếng', got %q", e.eng.GetWordBuffer())
	}

	// Type Backspace (removes 'g' from "tiếng" -> "tiến")
	handledBS, _ := e.ProcessKeyEvent(IBusKeyBackSpace, EvdevKeyBackSpace, 0)
	if !handledBS {
		t.Fatalf("expected Backspace with buffer to return true (handled by IME)")
	}
	if e.eng.GetWordBuffer() != "tiến" {
		t.Fatalf("expected buffer after backspace 'tiến', got %q", e.eng.GetWordBuffer())
	}

	// Type 'g' -> "tiến" + "g" = "tiếng"
	e.ProcessKeyEvent(uint32('g'), 0, 0)
	if e.eng.GetWordBuffer() != "tiếng" {
		t.Fatalf("expected buffer 'tiếng', got %q", e.eng.GetWordBuffer())
	}
}

func TestNepKeyEngine_HyphenAndSlashTyping(t *testing.T) {
	cfg := config.DefaultConfig()
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)

	// 1. Test "bài-học"
	for _, ch := range "baif" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "bài" {
		t.Fatalf("expected buffer 'bài', got %q", e.eng.GetWordBuffer())
	}

	// Type '-' -> should reset buffer and allow next word to compose Vietnamese
	e.ProcessKeyEvent(uint32('-'), 0, 0)
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer to be empty after '-', got %q", e.eng.GetWordBuffer())
	}

	// Type "hocj" -> should compose "học" without being blocked
	for _, ch := range "hocj" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "học" {
		t.Fatalf("expected buffer 'học', got %q", e.eng.GetWordBuffer())
	}

	e.ProcessKeyEvent(IBusKeySpace, 0, 0)

	// 2. Test "và/hoặc" and "anh/chị"
	for _, ch := range "vaf" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "và" {
		t.Fatalf("expected buffer 'và', got %q", e.eng.GetWordBuffer())
	}

	e.ProcessKeyEvent(uint32('/'), 0, 0)
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer to be empty after '/', got %q", e.eng.GetWordBuffer())
	}

	for _, ch := range "hoawjc" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "hoặc" {
		t.Fatalf("expected buffer 'hoặc', got %q", e.eng.GetWordBuffer())
	}
}

func TestNepKeyEngine_ScopedSlashAndMultiSegmentPaths(t *testing.T) {
	cfg := config.DefaultConfig()
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)

	// 1. Multi-segment path at word boundary: "/media/tuannt/Data/DEV/nepkey"
	for _, ch := range "/media/tuannt/Data/DEV/nepkey" {
		handled, _ := e.ProcessKeyEvent(uint32(ch), 0, 0)
		if handled {
			t.Fatalf("expected %q in path to pass through at 0ms (handled=false)", string(ch))
		}
		if !e.slashCommandMode {
			t.Fatalf("expected slashCommandMode/rawTokenMode to remain true throughout path at %q", string(ch))
		}
		if e.eng.HasBuffer() {
			t.Fatalf("expected Vietnamese buffer to remain empty during path typing, got %q", e.eng.GetWordBuffer())
		}
	}

	// Press Space to exit path mode
	e.ProcessKeyEvent(IBusKeySpace, 0, 0)
	if e.slashCommandMode {
		t.Fatalf("expected slashCommandMode to be false after Space")
	}

	// 2. Code property & URL: "fmt.Errorf" and "github.com/search"
	for _, ch := range "fmt.Errorf" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer to be empty after 'fmt.Errorf', got %q", e.eng.GetWordBuffer())
	}
	e.ProcessKeyEvent(IBusKeySpace, 0, 0)

	for _, ch := range "github.com/search" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer to be empty after 'github.com/search', got %q", e.eng.GetWordBuffer())
	}
	e.ProcessKeyEvent(IBusKeySpace, 0, 0)

	// 3. CLI flags: "--force" and "-rf"
	for _, ch := range "--force" {
		handled, _ := e.ProcessKeyEvent(uint32(ch), 0, 0)
		if handled {
			t.Fatalf("expected CLI flag char %q to pass through directly", string(ch))
		}
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer to be empty after '--force', got %q", e.eng.GetWordBuffer())
	}
	e.ProcessKeyEvent(IBusKeySpace, 0, 0)

	// 4. Normal Vietnamese typing after technical tokens works seamlessly
	for _, ch := range "chaof" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "chào" {
		t.Fatalf("expected buffer 'chào', got %q", e.eng.GetWordBuffer())
	}
}

func TestNepKeyEngine_SinglePressEnter(t *testing.T) {
	cfg := config.DefaultConfig()
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)

	// Type "chaof" -> "chào"
	for _, ch := range "chaof" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "chào" {
		t.Fatalf("expected buffer 'chào', got %q", e.eng.GetWordBuffer())
	}

	// Press Enter (IBusKeyReturn):
	// In Direct Commit mode, "chào" is already in the client field (isComposing = false).
	// Return must reset buffer and return false so Chrome Omnibox / Web Chat executes Enter in 1 single press!
	handled, err := e.ProcessKeyEvent(IBusKeyReturn, 36, 0)
	if err != nil {
		t.Fatalf("unexpected error on Return: %v", err)
	}
	if handled {
		t.Fatalf("expected Return with buffer to return false (so client app gets Enter directly), got true")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer to be empty after Return, got %q", e.eng.GetWordBuffer())
	}
}

func TestNepKeyEngine_BracketsVietnamese(t *testing.T) {
	cfg := config.DefaultConfig()
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test", cfg)

	// Type '('
	handled, _ := e.ProcessKeyEvent(uint32('('), 0, 0)
	if handled {
		t.Fatalf("expected '(' to pass through directly (false)")
	}

	// Type "vieetj" -> should compose "việt" inside brackets
	for _, ch := range "vieetj" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if e.eng.GetWordBuffer() != "việt" {
		t.Fatalf("expected buffer 'việt', got %q", e.eng.GetWordBuffer())
	}
}
