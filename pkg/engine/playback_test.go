package engine

import (
	"fmt"
	"testing"
)

func TestPlayback_Lai(t *testing.T) {
	eng := NewEngine("Telex 2")

	// User sequence: l, a, i, j
	keys := []rune{'l', 'a', 'i', 'j'}
	var screen []rune

	for _, k := range keys {
		act := eng.ProcessKey(k)
		fmt.Printf("Key '%c': DirectPass=%v, Backspaces=%d, Append=%q\n", k, act.DirectPass, act.Backspaces, string(act.AppendRunes))
		if act.DirectPass {
			screen = append(screen, k)
		} else {
			if act.Backspaces > 0 {
				screen = screen[:len(screen)-act.Backspaces]
			}
			screen = append(screen, act.AppendRunes...)
		}
		fmt.Printf("   -> Screen so far: %q\n", string(screen))
	}

	if string(screen) != "lại" {
		t.Fatalf("Expected 'lại', got %q", string(screen))
	}
}

func TestPlayback_Viet(t *testing.T) {
	eng := NewEngine("Telex 2")

	// User sequence: v, i, e, e, t, j
	keys := []rune{'v', 'i', 'e', 'e', 't', 'j'}
	var screen []rune

	for _, k := range keys {
		act := eng.ProcessKey(k)
		fmt.Printf("Key '%c': DirectPass=%v, Backspaces=%d, Append=%q\n", k, act.DirectPass, act.Backspaces, string(act.AppendRunes))
		if act.DirectPass {
			screen = append(screen, k)
		} else {
			if act.Backspaces > 0 {
				screen = screen[:len(screen)-act.Backspaces]
			}
			screen = append(screen, act.AppendRunes...)
		}
		fmt.Printf("   -> Screen so far: %q\n", string(screen))
	}

	if string(screen) != "việt" {
		t.Fatalf("Expected 'việt', got %q", string(screen))
	}
}

func TestPlayback_On(t *testing.T) {
	eng := NewEngine("Telex 2")

	// User sequence: o, o, n, r
	keys := []rune{'o', 'o', 'n', 'r'}
	var screen []rune

	for _, k := range keys {
		act := eng.ProcessKey(k)
		fmt.Printf("Key '%c': DirectPass=%v, Backspaces=%d, Append=%q\n", k, act.DirectPass, act.Backspaces, string(act.AppendRunes))
		if act.DirectPass {
			screen = append(screen, k)
		} else {
			if act.Backspaces > 0 {
				screen = screen[:len(screen)-act.Backspaces]
			}
			screen = append(screen, act.AppendRunes...)
		}
		fmt.Printf("   -> Screen so far: %q\n", string(screen))
	}

	if string(screen) != "ổn" {
		t.Fatalf("Expected 'ổn', got %q", string(screen))
	}
}
