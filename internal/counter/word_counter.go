package counter

import (
	"log/slog"
	"strings"
)

// WordCounter implements word counting using whitespace splitting.
type WordCounter struct{}

// NewWordCounter creates a new WordCounter instance.
func NewWordCounter() Counter {
	return &WordCounter{}
}

// Count returns the number of words in the given text using strings.Fields()
// This method splits on any Unicode whitespace and filters out empty strings.
func (wc *WordCounter) Count(text string) int {
	if text == "" {
		return 0
	}

	// strings.Fields splits on whitespace and filters empty strings
	words := strings.Fields(text)
	wordCount := len(words)

	slog.Debug("Word count calculated", "textLength", len(text), "wordCount", wordCount)
	return wordCount
}

// Name returns the name of this counting method for logging and debugging.
func (wc *WordCounter) Name() string {
	return "words"
}
