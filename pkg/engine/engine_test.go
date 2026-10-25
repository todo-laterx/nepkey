package engine

import (
	"testing"
)

func TestEngine_TypingWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tiếng",
			input:    "tieengs",
			expected: "tiếng",
		},
		{
			name:     "việt",
			input:    "vieetj",
			expected: "việt",
		},
		{
			name:     "chuẩn",
			input:    "chuaarn",
			expected: "chuẩn",
		},
		{
			name:     "đường",
			input:    "dduwowngf",
			expected: "đường",
		},
		{
			name:     "người",
			input:    "ngwowif",
			expected: "người",
		},
		{
			name:     "thoát dấu khi gõ lặp z",
			input:    "casz",
			expected: "ca",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine("Telex 2")
			var screen []rune

			for _, ch := range tt.input {
				act := eng.ProcessKey(ch)
				if act.DirectPass {
					screen = append(screen, ch)
				} else {
					if act.Backspaces > 0 {
						screen = screen[:len(screen)-act.Backspaces]
					}
					screen = append(screen, act.AppendRunes...)
				}
			}

			result := string(screen)
			if result != tt.expected {
				t.Errorf("input [%s]: got [%s], want [%s]", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEngine_SpecialCharacters(t *testing.T) {
	eng := NewEngine("Telex 2")

	// Typing "@" should reset buffer and be DirectPass
	act := eng.ProcessKey('@')
	if !act.DirectPass {
		t.Errorf("expected '@' to be DirectPass, got backspaces=%d", act.Backspaces)
	}

	// Typing "/" should reset buffer and be DirectPass
	act = eng.ProcessKey('/')
	if !act.DirectPass {
		t.Errorf("expected '/' to be DirectPass, got backspaces=%d", act.Backspaces)
	}
}

func TestEngine_Macro(t *testing.T) {
	eng := NewEngine("Telex 2")
	macros := map[string]string{
		"vn":  "Việt Nam",
		"hn":  "Hà Nội",
		"ko":  "không",
		"dđ":  "đường",
	}
	eng.SetMacros(macros, true)

	// Test 1: typing "vn" then space
	eng.Reset()
	eng.ProcessKey('v')
	eng.ProcessKey('n')
	if eng.GetWordBuffer() != "vn" {
		t.Fatalf("expected buffer 'vn', got '%s'", eng.GetWordBuffer())
	}

	act := eng.CheckMacro(' ')
	if act == nil {
		t.Fatalf("expected macro expansion for 'vn '")
	}
	if act.Backspaces != 2 {
		t.Errorf("expected 2 backspaces, got %d", act.Backspaces)
	}
	if string(act.AppendRunes) != "Việt Nam " {
		t.Errorf("expected 'Việt Nam ', got '%s'", string(act.AppendRunes))
	}

	// Test 2: typing "VN" (uppercase) then space
	eng.Reset()
	eng.ProcessKey('V')
	eng.ProcessKey('N')
	act = eng.CheckMacro(' ')
	if act == nil || string(act.AppendRunes) != "VIỆT NAM " {
		t.Errorf("expected 'VIỆT NAM ', got '%v'", act)
	}

	// Test 3: typing "hn." (punctuation trigger)
	eng.Reset()
	eng.ProcessKey('h')
	eng.ProcessKey('n')
	actPunct := eng.ProcessKey('.')
	if actPunct.DirectPass || string(actPunct.AppendRunes) != "Hà Nội." {
		t.Errorf("expected 'Hà Nội.', got '%s'", string(actPunct.AppendRunes))
	}
}

func TestEngine_ToneStyles(t *testing.T) {
	// 1. Modern tone style (òa, ùy) - default
	engModern := NewEngine("Telex 2")
	engModern.SetOptions(true, true, true, false, nil)
	engModern.ProcessKey('o')
	engModern.ProcessKey('a')
	engModern.ProcessKey('f')
	if engModern.GetWordBuffer() != "òa" {
		t.Errorf("Modern tone: expected 'òa', got '%s'", engModern.GetWordBuffer())
	}

	// 2. Traditional tone style (oà, uỳ)
	engTrad := NewEngine("Telex 2")
	engTrad.SetOptions(true, false, true, false, nil)
	engTrad.ProcessKey('o')
	engTrad.ProcessKey('a')
	engTrad.ProcessKey('f')
	if engTrad.GetWordBuffer() != "oà" {
		t.Errorf("Traditional tone: expected 'oà', got '%s'", engTrad.GetWordBuffer())
	}
}

func TestEngine_NumbersAndLetters(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"20p", "20p"},
		{"20phuts", "20phút"},
		{"wwin10", "win10"},
		{"mp3", "mp3"},
		{"gặp 20 bạn", "gặp 20 bạn"},
	}

	for _, tt := range tests {
		eng := NewEngine("Telex 2")
		var screen []rune
		for _, ch := range tt.input {
			act := eng.ProcessKey(ch)
			if act.DirectPass {
				screen = append(screen, ch)
			} else {
				if act.Backspaces > 0 {
					if act.Backspaces > len(screen) {
						act.Backspaces = len(screen)
					}
					screen = screen[:len(screen)-act.Backspaces]
				}
				screen = append(screen, act.AppendRunes...)
			}
		}
		got := string(screen)
		if got != tt.expected {
			t.Errorf("input [%s]: got [%s], want [%s]", tt.input, got, tt.expected)
		}
	}
}

func TestEngine_Charsets(t *testing.T) {
	tests := []struct {
		charset  string
		input    string
		expected string
	}{
		{"Unicode", "vieetj", "việt"},
		{"Unicode tổ hợp", "vieetj", "việt"},
		{"TCVN3 (ABC)", "vieetj", "viÖt"},
		{"VNI Windows", "vieetj", "vieät"},
	}

	for _, tt := range tests {
		eng := NewEngine("Telex 2")
		eng.SetCharset(tt.charset)
		var screen []rune
		for _, ch := range tt.input {
			act := eng.ProcessKey(ch)
			if act.DirectPass {
				screen = append(screen, ch)
			} else {
				if act.Backspaces > 0 {
					if act.Backspaces > len(screen) {
						act.Backspaces = len(screen)
					}
					screen = screen[:len(screen)-act.Backspaces]
				}
				screen = append(screen, act.AppendRunes...)
			}
		}
		got := string(screen)
		if got != tt.expected {
			t.Errorf("charset [%s], input [%s]: got [%s], want [%s]", tt.charset, tt.input, got, tt.expected)
		}
	}
}



