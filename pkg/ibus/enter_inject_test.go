package ibus

import (
	"testing"
	"time"

	"github.com/todo-laterx/nepkey/pkg/config"
)

type fakeTapper struct {
	taps chan uint16
}

func (f *fakeTapper) Tap(key uint16) error {
	f.taps <- key
	return nil
}

func (f *fakeTapper) Close() error { return nil }

func newInjectEngine(t *testing.T) (*NepKeyEngine, *fakeTapper) {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test-enter", cfg)
	ft := &fakeTapper{taps: make(chan uint16, 4)}
	e.tapper = ft
	return e, ft
}

func withInstantInject(t *testing.T) {
	t.Helper()
	old := enterInjectDelay
	enterInjectDelay = 0
	t.Cleanup(func() { enterInjectDelay = old })
}

func TestEnterInjection_ConsumesThenTapsRealEnter(t *testing.T) {
	withInstantInject(t)
	e, ft := newInjectEngine(t)
	for _, ch := range "chaof" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	handled, _ := e.ProcessKeyEvent(IBusKeyReturn, 36, 0)
	if !handled {
		t.Fatalf("expected Enter consumed in favor of the injected Enter")
	}
	select {
	case key := <-ft.taps:
		if key != keyReturn {
			t.Fatalf("expected Return key code, got %d", key)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected one injected Enter")
	}
	if e.eng.HasBuffer() {
		t.Fatalf("expected buffer committed before the tap, got %q", e.eng.GetWordBuffer())
	}
	if text := e.commitLog; len(text) == 0 || text[len(text)-1] != "chào" {
		t.Fatalf("expected 'chào' committed, got %v", text)
	}
}

func TestEnterInjection_ShiftEnterKeepsNativePath(t *testing.T) {
	withInstantInject(t)
	e, ft := newInjectEngine(t)
	for _, ch := range "chaof" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	handled, _ := e.ProcessKeyEvent(IBusKeyReturn, 36, IBusShiftMask)
	if handled {
		t.Fatalf("Shift+Enter must stay native (handled=false)")
	}
	select {
	case key := <-ft.taps:
		t.Fatalf("unexpected injected key %d for Shift+Enter", key)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestEnterInjection_WithoutTapperKeepsNativePath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.InputMethod = "Telex 2"
	e := NewNepKeyEngine(nil, "/org/freedesktop/IBus/Engine/NepKey/test-enter-native", cfg)
	for _, ch := range "chaof" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if handled, _ := e.ProcessKeyEvent(IBusKeyReturn, 36, 0); handled {
		t.Fatalf("without a tapper the Enter must reach the client (handled=false)")
	}
}

func TestEnterInjection_DebouncesRapidSecondEnter(t *testing.T) {
	old := enterInjectDelay
	enterInjectDelay = 60 * time.Millisecond
	t.Cleanup(func() { enterInjectDelay = old })

	e, ft := newInjectEngine(t)
	for _, ch := range "to" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if handled, _ := e.ProcessKeyEvent(IBusKeyReturn, 36, 0); !handled {
		t.Fatalf("first Enter should be consumed")
	}
	// New word while the first tap is still in flight: the second Enter is
	// consumed but must not queue a second submit.
	for _, ch := range "me" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	e.ProcessKeyEvent(IBusKeyReturn, 36, 0)

	select {
	case <-ft.taps:
	case <-time.After(time.Second):
		t.Fatalf("expected the first injected Enter")
	}
	select {
	case key := <-ft.taps:
		t.Fatalf("unexpected second injected key %d", key)
	case <-time.After(160 * time.Millisecond):
	}
	if e.enterPending {
		t.Fatalf("expected pending flag cleared after the tap")
	}
}

func TestEnterInjection_EchoEnterDoesNotChain(t *testing.T) {
	withInstantInject(t)
	e, ft := newInjectEngine(t)
	for _, ch := range "to" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	e.ProcessKeyEvent(IBusKeyReturn, 36, 0)
	select {
	case <-ft.taps:
	case <-time.After(time.Second):
		t.Fatalf("expected the injected Enter")
	}
	// The tap has landed, so the cooldown is armed. A Return arriving now
	// (the echo, or a very fast second Enter) must go native, not chain.
	for _, ch := range "me" {
		e.ProcessKeyEvent(uint32(ch), 0, 0)
	}
	if handled, _ := e.ProcessKeyEvent(IBusKeyReturn, 36, 0); handled {
		t.Fatalf("expected the cooldown Enter to reach the client (handled=false)")
	}
	select {
	case key := <-ft.taps:
		t.Fatalf("unexpected chained injection %d", key)
	case <-time.After(80 * time.Millisecond):
	}
}
