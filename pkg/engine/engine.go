package engine

import (
	"strings"
	"unicode"

	bamboo "github.com/BambooEngine/bamboo-core"
)

// Action describes what the input handler needs to do on the virtual keyboard
type Action struct {
	// Number of backspaces to delete previous erroneous/unaccented characters
	Backspaces int
	// Characters to append after backspacing
	AppendRunes []rune
	// If DirectPass is true, the original key event can just be forwarded 1:1
	DirectPass bool
}

// Engine wraps bamboo-core and maintains the word buffer
type Engine struct {
	bEngine         bamboo.IEngine
	currentOnScreen []rune
	inputMethodName string
	charset         string
	enabled         bool
	freeToneMarking bool
	modernToneStyle bool
	autoCorrect     bool
	macroEnabled    bool
	macros          map[string]string
}

// NewEngine creates a new Vietnamese processing engine
func NewEngine(inputMethodName string) *Engine {
	if inputMethodName == "" {
		inputMethodName = "Telex 2"
	}
	eng := &Engine{
		currentOnScreen: make([]rune, 0, 16),
		inputMethodName: inputMethodName,
		charset:         "Unicode",
		enabled:         true,
		freeToneMarking: true,
		modernToneStyle: true,
		autoCorrect:     true,
		macroEnabled:    true,
		macros:          make(map[string]string),
	}
	im := bamboo.ParseInputMethod(bamboo.InputMethodDefinitions, inputMethodName)
	eng.bEngine = bamboo.NewEngine(im, eng.calculateFlags())
	return eng
}

func (e *Engine) calculateFlags() uint {
	flags := uint(0)
	if e.freeToneMarking {
		flags |= bamboo.EfreeToneMarking
	}
	if e.modernToneStyle {
		// In bamboo-core, EstdToneStyle places tone on the first vowel (òa, òe, ùy)
		flags |= bamboo.EstdToneStyle
	}
	if e.autoCorrect {
		flags |= bamboo.EautoCorrectEnabled
	}
	return flags
}

// SetOptions configures tone style flags and macro settings
func (e *Engine) SetOptions(freeTone, modernTone, autoCorrect, macroEnabled bool, macros map[string]string) {
	e.freeToneMarking = freeTone
	e.modernToneStyle = modernTone
	e.autoCorrect = autoCorrect
	e.macroEnabled = macroEnabled
	if macros != nil {
		e.macros = macros
	}
	e.bEngine.SetFlag(e.calculateFlags())
}

// SetMacros updates the macro expansion table
func (e *Engine) SetMacros(macros map[string]string, enabled bool) {
	if macros != nil {
		e.macros = macros
	}
	e.macroEnabled = enabled
}

// SetEnabled toggles Vietnamese processing on or off
func (e *Engine) SetEnabled(enabled bool) {
	e.enabled = enabled
	if !enabled {
		e.Reset()
	}
}

// IsEnabled returns current enabled status
func (e *Engine) IsEnabled() bool {
	return e.enabled
}

// SetInputMethod switches input method (e.g., "Telex 2", "Telex", "VNI")
func (e *Engine) SetInputMethod(name string) {
	im := bamboo.ParseInputMethod(bamboo.InputMethodDefinitions, name)
	e.bEngine = bamboo.NewEngine(im, e.calculateFlags())
	e.inputMethodName = name
	e.Reset()
}

// GetInputMethod returns the current input method name
func (e *Engine) GetInputMethod() string {
	return e.inputMethodName
}

// SetCharset configures target output charset (e.g. "Unicode", "Unicode tổ hợp", "TCVN3 (ABC)", "VNI Windows")
func (e *Engine) SetCharset(cs string) {
	if cs == "" {
		cs = "Unicode"
	}
	e.charset = cs
	e.Reset()
}

// GetCharset returns the current output charset
func (e *Engine) GetCharset() string {
	if e.charset == "" {
		return "Unicode"
	}
	return e.charset
}

// Reset clears the current word buffer and bamboo engine state
func (e *Engine) Reset() {
	e.bEngine.Reset()
	e.currentOnScreen = e.currentOnScreen[:0]
}

// HasBuffer returns true if there is an active word in the buffer
func (e *Engine) HasBuffer() bool {
	return len(e.currentOnScreen) > 0
}

// OnBackspace handles backspace event
func (e *Engine) OnBackspace() {
	if len(e.currentOnScreen) > 0 {
		e.bEngine.RemoveLastChar(false)
		e.currentOnScreen = e.currentOnScreen[:len(e.currentOnScreen)-1]
	}
}

// GetWordBuffer returns the current string in the word buffer
func (e *Engine) GetWordBuffer() string {
	return string(e.currentOnScreen)
}

func (e *Engine) findMacro(word string) (string, bool) {
	if len(e.macros) == 0 {
		return "", false
	}
	// 1. Exact match
	if rep, ok := e.macros[word]; ok {
		return rep, true
	}
	// 2. Lowercase match
	lower := strings.ToLower(word)
	rep, ok := e.macros[lower]
	if !ok {
		return "", false
	}
	// If word was all caps (e.g. VN -> VIỆT NAM)
	rWord := []rune(word)
	if word == strings.ToUpper(word) && len(rWord) > 1 {
		return strings.ToUpper(rep), true
	}
	// If word was Title Case (e.g. Vn -> Việt Nam)
	if unicode.IsUpper(rWord[0]) {
		rRep := []rune(rep)
		if len(rRep) > 0 {
			rRep[0] = unicode.ToUpper(rRep[0])
			return string(rRep), true
		}
	}
	return rep, true
}

// CheckMacro checks if the word currently on screen matches an abbreviation.
// If sep is non-zero, sep is appended to the replacement runes.
func (e *Engine) CheckMacro(sep rune) *Action {
	if !e.macroEnabled || len(e.currentOnScreen) == 0 {
		return nil
	}
	word := string(e.currentOnScreen)
	rep, ok := e.findMacro(word)
	if !ok {
		return nil
	}
	if e.charset != "" && e.charset != "Unicode" {
		rep = bamboo.Encode(e.charset, rep)
	}

	bs := len(e.currentOnScreen)
	e.Reset()

	var appendRunes []rune
	if sep > 0 {
		appendRunes = append([]rune(rep), sep)
	} else {
		appendRunes = []rune(rep)
	}

	return &Action{
		Backspaces:  bs,
		AppendRunes: appendRunes,
		DirectPass:  false,
	}
}

// IsWordCharacter returns true if the rune can be part of a Vietnamese word
func IsWordCharacter(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\''
}

// ProcessKey processes a single typed rune and calculates diff actions
func (e *Engine) ProcessKey(r rune) Action {
	// If engine is disabled, bypass completely
	if !e.enabled {
		return Action{DirectPass: true}
	}

	// Non-word characters (spaces, punctuation, @, /, #, etc.)
	if !IsWordCharacter(r) {
		if act := e.CheckMacro(r); act != nil {
			return *act
		}
		e.Reset()
		return Action{DirectPass: true}
	}

	// Feed rune to bamboo engine
	e.bEngine.ProcessKey(r, bamboo.VietnameseMode)
	newStr := e.bEngine.GetProcessedString(bamboo.VietnameseMode)
	if e.charset != "" && e.charset != "Unicode" {
		newStr = bamboo.Encode(e.charset, newStr)
	}
	newRunes := []rune(newStr)

	// If bamboo produced nothing (e.g. digits in Telex, symbols), reset buffer cleanly without polluting it
	if len(newRunes) == 0 {
		e.Reset()
		return Action{DirectPass: true}
	}

	// Calculate common prefix between currentOnScreen and newRunes
	oldRunes := e.currentOnScreen
	prefixLen := 0
	minLen := len(oldRunes)
	if len(newRunes) < minLen {
		minLen = len(newRunes)
	}

	for prefixLen < minLen && oldRunes[prefixLen] == newRunes[prefixLen] {
		prefixLen++
	}

	backspaces := len(oldRunes) - prefixLen
	appendRunes := newRunes[prefixLen:]

	// Check if this was a simple append of the same rune with 0 backspaces
	if backspaces == 0 && len(appendRunes) == 1 && appendRunes[0] == r {
		e.currentOnScreen = newRunes
		return Action{
			Backspaces:  0,
			AppendRunes: appendRunes,
			DirectPass:  true,
		}
	}

	// Transformation occurred (e.g. "tie" + "e" -> "tiê" or "tieng" + "s" -> "tiếng")
	e.currentOnScreen = newRunes
	return Action{
		Backspaces:  backspaces,
		AppendRunes: appendRunes,
		DirectPass:  false,
	}
}
