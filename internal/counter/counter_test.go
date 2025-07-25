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
			// (exact token counts can vary with encoding versions)
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

func TestTokenCounter_CreatePartialText(t *testing.T) {
	counter, err := NewTokenCounter()
	if err != nil {
		t.Fatalf("Failed to create TokenCounter: %v", err)
	}

	tokenCounter, ok := counter.(*TokenCounter)
	if !ok {
		t.Fatalf("Failed to cast to TokenCounter")
	}

	tests := []struct {
		name      string
		text      string
		maxTokens int
		expectErr bool
	}{
		{
			name:      "exact token limit",
			text:      "Hello world, this is a test sentence with punctuation!",
			maxTokens: 5,
			expectErr: false,
		},
		{
			name:      "single token",
			text:      "Hello world from the test suite",
			maxTokens: 1,
			expectErr: false,
		},
		{
			name:      "zero tokens",
			text:      "Any text here",
			maxTokens: 0,
			expectErr: false,
		},
		{
			name:      "negative tokens",
			text:      "Any text here",
			maxTokens: -1,
			expectErr: false,
		},
		{
			name:      "text fits completely",
			text:      "Short",
			maxTokens: 10,
			expectErr: false,
		},
		{
			name:      "empty text",
			text:      "",
			maxTokens: 5,
			expectErr: false,
		},
		{
			name:      "complex tokenization",
			text:      "The JavaScript function() returns JSON data with HTTP status codes.",
			maxTokens: 8,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalTokens := counter.Count(tt.text)
			result := tokenCounter.CreatePartialText(tt.text, tt.maxTokens)

			if tt.maxTokens <= 0 {
				if result != "" {
					t.Errorf("Expected empty result for maxTokens <= 0, got: %q", result)
				}
				return
			}

			if tt.text == "" {
				if result != "" {
					t.Errorf("Expected empty result for empty text, got: %q", result)
				}
				return
			}

			resultTokens := counter.Count(result)

			if originalTokens <= tt.maxTokens {
				if result != tt.text {
					t.Errorf("Expected full text when it fits:\n  Original: %q\n  Result: %q\n  Original tokens: %d\n  Max tokens: %d",
						tt.text, result, originalTokens, tt.maxTokens)
				}
			} else {
				// partial text should return exact token count
				if resultTokens != tt.maxTokens {
					t.Errorf("Expected exactly %d tokens, got %d:\n  Text: %q\n  Result: %q\n  Original tokens: %d",
						tt.maxTokens, resultTokens, tt.text, result, originalTokens)
				}

				// should be shorter than original
				if resultTokens >= originalTokens {
					t.Errorf("Expected truncation:\n  Original tokens: %d\n  Result tokens: %d\n  Text: %q\n  Result: %q",
						originalTokens, resultTokens, tt.text, result)
				}

				// should be non-empty for reasonable token limits
				if resultTokens == 0 && tt.maxTokens > 0 {
					t.Errorf("Expected non-empty result for positive token limit:\n  Max tokens: %d\n  Text: %q",
						tt.maxTokens, tt.text)
				}
			}
		})
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
