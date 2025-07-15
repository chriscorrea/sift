package counter

import (
	"log/slog"
	"unicode/utf8"
)

// CharCounter implements character counting using UTF-8 rune counting.
// Note that this should count Unicode characters properly, not just bytes.
type CharCounter struct{}

// NewCharCounter creates a new CharCounter instance.
func NewCharCounter() Counter {
	return &CharCounter{}
}

// Count returns the number of UTF-8 characters (runes) in the given text.
func (cc *CharCounter) Count(text string) int {
	if text == "" {
		return 0
	}

	// use utf8.RuneCountInString for proper Unicode character counting
	charCount := utf8.RuneCountInString(text)

	slog.Debug("Character count calculated", "textLength", len(text), "charCount", charCount)
	return charCount
}

// Name returns the name of this counting method for logging and debugging.
func (cc *CharCounter) Name() string {
	return "characters"
}
