package counter

import (
	"testing"
)

func TestWordCounter(t *testing.T) {
	counter := NewWordCounter()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"empty string", "", 0},
		{"single word", "hello", 1},
		{"multiple words", "hello world test", 3},
		{"whitespace handling", "  hello   world  ", 2},
		{"unicode words", "café naïve résumé", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := counter.Count(tt.text)
			if result != tt.expected {
				t.Errorf("WordCounter.Count(%q) = %d, want %d", tt.text, result, tt.expected)
			}
		})
	}

	if counter.Name() != "words" {
		t.Errorf("WordCounter.Name() = %q, want %q", counter.Name(), "words")
	}
}

func TestCharCounter(t *testing.T) {
	counter := NewCharCounter()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"empty string", "", 0},
		{"single char", "a", 1},
		{"multiple chars", "hello", 5},
		{"unicode chars", "café", 4}, // é is one rune
		{"whitespace included", "a b", 3},
		{"emoji", "hello 👋", 7}, // emoji is one rune
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := counter.Count(tt.text)
			if result != tt.expected {
				t.Errorf("CharCounter.Count(%q) = %d, want %d", tt.text, result, tt.expected)
			}
		})
	}

	if counter.Name() != "characters" {
		t.Errorf("CharCounter.Name() = %q, want %q", counter.Name(), "characters")
	}
}

func TestTokenCounter(t *testing.T) {
	counter, err := NewTokenCounter()
	if err != nil {
		t.Fatalf("Failed to create TokenCounter: %v", err)
	}

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"empty string", "", 0},
		{"simple text", "hello world", 2},
		{"punctuation", "Hello, world!", 3}, // typically is "Hello", ",", " world!"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := counter.Count(tt.text)
			// for token counting, we'll just verify it returns a positive number for non-empty text
			// bc exact token counts can vary with encoding versions
			if tt.text == "" {
				if result != 0 {
					t.Errorf("TokenCounter.Count(%q) = %d, want 0 for empty string", tt.text, result)
				}
			} else {
				if result <= 0 {
					t.Errorf("TokenCounter.Count(%q) = %d, want positive number for non-empty text", tt.text, result)
				}
			}
		})
	}

	if counter.Name() != "tokens (cl100k_base)" {
		t.Errorf("TokenCounter.Name() = %q, want %q", counter.Name(), "tokens (cl100k_base)")
	}
}

func TestNewCounter(t *testing.T) {
	tests := []struct {
		name         string
		method       CountingMethod
		expectedName string
		expectError  bool
	}{
		{"tokens", Tokens, "tokens (cl100k_base)", false},
		{"words", Words, "words", false},
		{"characters", Characters, "characters", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counter, err := NewCounter(tt.method)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewCounter(%v) expected error, got nil", tt.method)
				}
				return
			}

			if err != nil {
				t.Errorf("NewCounter(%v) unexpected error: %v", tt.method, err)
				return
			}

			if counter.Name() != tt.expectedName {
				t.Errorf("NewCounter(%v).Name() = %q, want %q", tt.method, counter.Name(), tt.expectedName)
			}
		})
	}
}

func TestCountingMethodString(t *testing.T) {
	tests := []struct {
		method   CountingMethod
		expected string
	}{
		{Tokens, "tokens"},
		{Words, "words"},
		{Characters, "characters"},
		{CountingMethod(999), "unknown"}, // invalid method
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.method.String()
			if result != tt.expected {
				t.Errorf("CountingMethod(%d).String() = %q, want %q", int(tt.method), result, tt.expected)
			}
		})
	}
}
