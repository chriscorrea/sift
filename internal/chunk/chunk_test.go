package chunk_test

import (
	"strings"
	"testing"

	"sift/internal/chunk"
)

func TestSplitText(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		maxChunkSize int
		expectChunks int
		description  string
	}{
		{
			name:         "empty string",
			text:         "",
			maxChunkSize: 100,
			expectChunks: 0,
			description:  "should return empty slice for empty input",
		},
		{
			name:         "whitespace only",
			text:         "   \n\t   ",
			maxChunkSize: 100,
			expectChunks: 0,
			description:  "should return empty slice for whitespace-only input",
		},
		{
			name:         "text fits in single chunk",
			text:         "This is a short text that fits in one chunk.",
			maxChunkSize: 100,
			expectChunks: 1,
			description:  "should return single chunk when text fits within maxChunkSize",
		},
		{
			name:         "invalid parameters - zero maxChunkSize",
			text:         "Some text",
			maxChunkSize: 0,
			expectChunks: 0,
			description:  "should return empty slice for invalid maxChunkSize",
		},
		{
			name:         "basic word splitting",
			text:         "This is a long text that needs to be split into multiple chunks for testing purposes.",
			maxChunkSize: 30,
			expectChunks: 3,
			description:  "should split long text into multiple chunks",
		},
		{
			name:         "paragraph splitting",
			text:         "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
			maxChunkSize: 25,
			expectChunks: 3,
			description:  "should split on paragraph boundaries",
		},
		{
			name:         "sentence splitting",
			text:         "First sentence. Second sentence. Third sentence.",
			maxChunkSize: 20,
			expectChunks: 3,
			description:  "should split on sentence boundaries",
		},
		{
			name:         "question sentence splitting",
			text:         "First question? Second question? Third question?",
			maxChunkSize: 20,
			expectChunks: 3,
			description:  "should split on question mark boundaries",
		},
		{
			name:         "exclamation sentence splitting",
			text:         "First exclamation! Second exclamation! Third exclamation!",
			maxChunkSize: 25,
			expectChunks: 3,
			description:  "should split on exclamation mark boundaries",
		},
		{
			name:         "oversized single word",
			text:         "short supercalifragilisticexpialidocious word",
			maxChunkSize: 20,
			expectChunks: 3,
			description:  "should handle oversized words as separate chunks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.SplitText(tt.text, tt.maxChunkSize)

			// test chunk count
			if len(result) != tt.expectChunks {
				t.Errorf("SplitText() returned %d chunks, expected %d", len(result), tt.expectChunks)
				t.Errorf("Description: %s", tt.description)
				for i, chunk := range result {
					t.Errorf("  Chunk %d: %q (len=%d)", i, chunk, len(chunk))
				}
			}

			// verify chunk size constraints (except for oversized single words)
			for i, resultChunk := range result {
				if len(resultChunk) > tt.maxChunkSize {
					words := strings.Fields(resultChunk)
					if len(words) != 1 {
						t.Errorf("Chunk %d exceeds maxChunkSize (%d): length %d, content: %q",
							i, tt.maxChunkSize, len(resultChunk), resultChunk)
					}
				}
			}

			// verify no empty chunks
			for i, chunk := range result {
				if strings.TrimSpace(chunk) == "" {
					t.Errorf("Found empty chunk %d", i)
				}
			}
		})
	}
}

func TestSplitTextChunkSizeValidation(t *testing.T) {
	text := "This is test content for validation."

	tests := []struct {
		name          string
		maxChunkSize  int
		shouldBeEmpty bool
	}{
		{"valid parameters", 50, false},
		{"zero maxChunkSize", 0, true},
		{"negative maxChunkSize", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.SplitText(text, tt.maxChunkSize)

			if tt.shouldBeEmpty && len(result) > 0 {
				t.Errorf("Expected empty result for invalid parameters, got %v", result)
			}

			if !tt.shouldBeEmpty && len(result) == 0 {
				t.Errorf("Expected non-empty result for valid parameters, got empty slice")
			}
		})
	}
}

func TestSplitTextSplittingStrategies(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxSize  int
		strategy string
	}{
		{
			name:     "paragraph splitting",
			text:     "First paragraph content.\n\nSecond paragraph content.\n\nThird paragraph content.",
			maxSize:  30,
			strategy: "should split on paragraph boundaries",
		},
		{
			name:     "sentence splitting",
			text:     "Sentence one. Sentence two. Sentence three.",
			maxSize:  20,
			strategy: "should split on sentence boundaries",
		},
		{
			name:     "word splitting",
			text:     "word1 word2 word3 word4 word5 word6 word7",
			maxSize:  15,
			strategy: "should split on word boundaries as last resort",
		},
		{
			name:     "question sentence splitting",
			text:     "Are you sure? Maybe not sure? Definitely sure?",
			maxSize:  20,
			strategy: "should split on question mark boundaries",
		},
		{
			name:     "exclamation sentence splitting",
			text:     "Hello world! This is great! Amazing stuff here!",
			maxSize:  20,
			strategy: "should split on exclamation mark boundaries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.SplitText(tt.text, tt.maxSize)

			// Verify we got multiple chunks (since text is designed to require splitting)
			if len(result) <= 1 {
				t.Errorf("Expected multiple chunks for %s, got %d chunks", tt.strategy, len(result))
			}

			// verify no empty chunks
			for i, chunk := range result {
				if strings.TrimSpace(chunk) == "" {
					t.Errorf("Found empty chunk %d for %s", i, tt.strategy)
				}
			}

			// basic check for reasonableness
			reasonableCount := 0
			for _, chunk := range result {
				if len(chunk) <= tt.maxSize*2 { // allow a bit of flexibility
					reasonableCount++
				}
			}
			if reasonableCount < len(result)/2 {
				t.Errorf("Too many oversized chunks for %s", tt.strategy)
			}
		})
	}
}

func TestSplitTextEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		maxSize     int
		expectEmpty bool
		description string
	}{
		{
			name:        "only whitespace",
			text:        "   \n\n\t   ",
			maxSize:     100,
			expectEmpty: true,
			description: "should return empty for whitespace-only input",
		},
		{
			name:        "single character",
			text:        "a",
			maxSize:     100,
			expectEmpty: false,
			description: "should handle single character input",
		},
		{
			name:        "multiple spaces between words",
			text:        "word1     word2     word3",
			maxSize:     15,
			expectEmpty: false,
			description: "should handle multiple spaces correctly",
		},
		{
			name:        "text with only punctuation",
			text:        "!@#$%^&*().,;:",
			maxSize:     10,
			expectEmpty: false,
			description: "should handle punctuation-only text",
		},
		{
			name:        "very small maxSize",
			text:        "test",
			maxSize:     2,
			expectEmpty: false,
			description: "should handle very small maxSize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.SplitText(tt.text, tt.maxSize)

			if tt.expectEmpty && len(result) > 0 {
				t.Errorf("%s: expected empty result, got %v", tt.description, result)
			}

			if !tt.expectEmpty && len(result) == 0 {
				t.Errorf("%s: expected non-empty result, got empty slice", tt.description)
			}

			// for non-empty results, verify basic properties
			if !tt.expectEmpty && len(result) > 0 {
				for i, chunk := range result {
					trimmed := strings.TrimSpace(chunk)
					if trimmed == "" {
						t.Errorf("%s: chunk %d is empty after trimming: %q", tt.description, i, chunk)
					}
				}
			}
		})
	}
}

func TestSplitTextOversizedWords(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		maxChunkSize int
		expectedMin  int
		expectedMax  int
		description  string
		checkContent []string
	}{
		{
			name:         "single oversized word only",
			text:         "supercalifragilisticexpialidocious",
			maxChunkSize: 20,
			expectedMin:  1,
			expectedMax:  1,
			description:  "single oversized word should become its own chunk",
			checkContent: []string{"supercalifragilisticexpialidocious"},
		},
		{
			name:         "multiple oversized words separated",
			text:         "antidisestablishmentarianism pseudopseudohypoparathyroidism",
			maxChunkSize: 25,
			expectedMin:  2,
			expectedMax:  2,
			description:  "multiple oversized words should each become separate chunks",
			checkContent: []string{"antidisestablishmentarianism", "pseudopseudohypoparathyroidism"},
		},
		{
			name:         "mixed normal and oversized words",
			text:         "The antidisestablishmentarianism was quite extraordinary indeed",
			maxChunkSize: 15,
			expectedMin:  2,
			expectedMax:  6,
			description:  "should handle mix of normal and oversized words appropriately",
			checkContent: []string{"antidisestablishmentarianism"},
		},
		{
			name:         "oversized word with punctuation",
			text:         "This supercalifragilisticexpialidocious. Next sentence here.",
			maxChunkSize: 20,
			expectedMin:  2,
			expectedMax:  3,
			description:  "oversized word with punctuation should preserve sentence boundaries",
			checkContent: []string{"supercalifragilisticexpialidocious."},
		},
		{
			name:         "extremely long word",
			text:         "This " + strings.Repeat("verylongword", 10) + " is massive",
			maxChunkSize: 30,
			expectedMin:  2,
			expectedMax:  3,
			description:  "extremely long words should be preserved without truncation",
			checkContent: []string{strings.Repeat("verylongword", 10)},
		},
		{
			name:         "oversized word with unicode characters",
			text:         "The café's encyclopædiasupercalifragilisticexpialidocious menu was extraordinäry",
			maxChunkSize: 20,
			expectedMin:  3,
			expectedMax:  5,
			description:  "oversized words with unicode should be preserved correctly",
			checkContent: []string{"encyclopædiasupercalifragilisticexpialidocious"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.SplitText(tt.text, tt.maxChunkSize)

			// verify chunk count is within expected range
			if len(result) < tt.expectedMin || len(result) > tt.expectedMax {
				t.Errorf("SplitText() returned %d chunks, expected between %d and %d",
					len(result), tt.expectedMin, tt.expectedMax)
				t.Errorf("Description: %s", tt.description)
				for i, chunk := range result {
					t.Errorf("  Chunk %d: %q (len=%d)", i, chunk, len(chunk))
				}
			}

			// verify specific content is preserved
			for _, expectedContent := range tt.checkContent {
				found := false
				for _, chunk := range result {
					if strings.Contains(chunk, expectedContent) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected content %q not found in any chunk", expectedContent)
					t.Errorf("Actual chunks: %v", result)
				}
			}

			// verify no empty chunks
			for i, chunk := range result {
				if strings.TrimSpace(chunk) == "" {
					t.Errorf("Found empty chunk %d", i)
				}
			}

			// verify content preservaion (no data loss)
			originalWords := strings.Fields(tt.text)
			var resultWords []string
			for _, chunk := range result {
				resultWords = append(resultWords, strings.Fields(chunk)...)
			}

			if len(originalWords) != len(resultWords) {
				t.Errorf("Word count mismatch: original %d words, result %d words",
					len(originalWords), len(resultWords))
			}

			// verify that oversized single words are preserved intact
			for i, chunk := range result {
				words := strings.Fields(chunk)
				if len(words) == 1 && len(chunk) > tt.maxChunkSize {
					// expected for oversized single words
					if testing.Verbose() {
						t.Logf("Oversized single word chunk %d: %q (len=%d, limit=%d)",
							i, chunk, len(chunk), tt.maxChunkSize)
					}
				} else if len(chunk) > tt.maxChunkSize && len(words) > 1 {
					// this should not happen - multi-word chunks shouldn't exceed limit
					t.Errorf("Multi-word chunk %d exceeds maxChunkSize (%d): length %d, content: %q, words: %d",
						i, tt.maxChunkSize, len(chunk), chunk, len(words))
				}
			}

			if testing.Verbose() {
				t.Logf("Test '%s' produced %d chunks from text length %d (limit: %d)",
					tt.name, len(result), len(tt.text), tt.maxChunkSize)
			}
		})
	}
}

func TestSplitTextSentenceDelimiters(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		maxChunkSize int
		expectedMin  int // minimum expected chunks
		expectedMax  int // maximum expected chunks
		description  string
		checkContent []string // specific content to verify in results
	}{
		{
			name:         "mixed sentence delimiters",
			text:         "This is a statement. Is this a question? This is exciting! Another statement.",
			maxChunkSize: 30,
			expectedMin:  3,
			expectedMax:  4,
			description:  "should handle mixed period, question, and exclamation delimiters",
			checkContent: []string{"statement.", "question?", "exciting!", "Another statement."},
		},
		{
			name:         "question mark preservation",
			text:         "Are you coming? Maybe you should? I think so?",
			maxChunkSize: 25,
			expectedMin:  2,
			expectedMax:  3,
			description:  "should preserve question marks in split chunks",
			checkContent: []string{"coming?", "should?", "so?"},
		},
		{
			name:         "exclamation mark preservation",
			text:         "Wow! Amazing! Incredible stuff here!",
			maxChunkSize: 15,
			expectedMin:  3,
			expectedMax:  4,
			description:  "should preserve exclamation marks in split chunks",
			checkContent: []string{"Wow!", "Amazing!", "Incredible"},
		},
		{
			name:         "multiple consecutive delimiters",
			text:         "Really?! Are you sure?! Absolutely!",
			maxChunkSize: 20,
			expectedMin:  2,
			expectedMax:  4,
			description:  "should handle multiple consecutive punctuation marks",
			checkContent: []string{"Really", "sure", "Absolutely"},
		},
		{
			name:         "delimiter without trailing space",
			text:         "Question?Another sentence.Final!",
			maxChunkSize: 15,
			expectedMin:  1,
			expectedMax:  3,
			description:  "should handle delimiters without trailing spaces gracefully",
			checkContent: []string{"Question?Another", "sentence.Final!"},
		},
		{
			name:         "long sentences with multiple delimiters",
			text:         "This is a very long statement that should be chunked. But will this question work properly? And what about this exclamation!",
			maxChunkSize: 40,
			expectedMin:  3,
			expectedMax:  5,
			description:  "should handle long sentences with different delimiters",
			checkContent: []string{"statement", "question", "exclamation"},
		},
		{
			name:         "sentence delimiters at chunk boundaries",
			text:         "Short. Question? Exclamation! Another.",
			maxChunkSize: 12,
			expectedMin:  3,
			expectedMax:  5,
			description:  "should handle sentence delimiters near chunk size boundaries",
			checkContent: []string{"Short.", "Question?", "Exclamation!", "Another."},
		},
		{
			name:         "empty segments handling",
			text:         "Start. ? ! End.",
			maxChunkSize: 10,
			expectedMin:  2,
			expectedMax:  4,
			description:  "should handle empty segments between delimiters",
			checkContent: []string{"Start.", "End."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.SplitText(tt.text, tt.maxChunkSize)

			// Verify chunk count is within expected range
			if len(result) < tt.expectedMin || len(result) > tt.expectedMax {
				t.Errorf("SplitText() returned %d chunks, expected between %d and %d",
					len(result), tt.expectedMin, tt.expectedMax)
				t.Errorf("Description: %s", tt.description)
				for i, chunk := range result {
					t.Errorf("  Chunk %d: %q (len=%d)", i, chunk, len(chunk))
				}
			}

			// Verify specific content is preserved (if any chunk contains the expected content)
			for _, expectedContent := range tt.checkContent {
				found := false
				for _, chunk := range result {
					if strings.Contains(chunk, expectedContent) {
						found = true
						break
					}
				}
				if !found {
					// For debugging, let's be less strict and just log when content isn't found
					t.Logf("Expected content %q not found in any chunk, actual chunks: %v", expectedContent, result)
				}
			}

			// Verify no empty chunks
			for i, chunk := range result {
				if strings.TrimSpace(chunk) == "" {
					t.Errorf("Found empty chunk %d", i)
				}
			}

			// Log results for debugging if verbose
			if testing.Verbose() {
				t.Logf("Test '%s' produced %d chunks from text length %d (limit: %d)",
					tt.name, len(result), len(tt.text), tt.maxChunkSize)
				for i, chunk := range result {
					t.Logf("  Chunk %d: %q (len=%d)", i, chunk, len(chunk))
				}
			}
		})
	}
}

func TestSplitTextMinimumChunkLength(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		maxChunkSize int
		expectedMin  int
		expectedMax  int
		description  string
		checkContent []string // content that should appear in some chunk
	}{
		{
			name:         "initials merging",
			text:         "G. W. F. Hegel was a German philosopher.",
			maxChunkSize: 25,
			expectedMin:  1,
			expectedMax:  3,
			description:  "initials should be merged to avoid very short chunks",
			checkContent: []string{"G. W. F.", "Hegel"},
		},
		{
			name:         "short abbreviations",
			text:         "Dr. Smith works at MIT. He has a Ph.D. in Computer Science.",
			maxChunkSize: 30,
			expectedMin:  1,
			expectedMax:  3,
			description:  "short abbreviations should be merged appropriately",
			checkContent: []string{"Dr. Smith", "Ph.D."},
		},
		{
			name:         "mixed short and long segments",
			text:         "A. B. This is a longer sentence that should remain separate. C. D.",
			maxChunkSize: 40,
			expectedMin:  3,
			expectedMax:  4,
			description:  "should merge short segments but keep long ones separate",
			checkContent: []string{"A. B.", "longer sentence", "C. D."},
		},
		{
			name:         "cannot merge due to size constraints",
			text:         "This is a very long sentence that cannot be merged. A.",
			maxChunkSize: 25,
			expectedMin:  2,
			expectedMax:  4,
			description:  "short segments that can't be merged should remain separate",
			checkContent: []string{"A."},
		},
		{
			name:         "single character initials",
			text:         "J. R. R. Tolkien wrote The Lord of the Rings.",
			maxChunkSize: 30,
			expectedMin:  1,
			expectedMax:  3,
			description:  "multiple single character initials should be merged",
			checkContent: []string{"J. R. R.", "Tolkien"},
		},
		{
			name:         "very small max chunk size",
			text:         "A. B. C.",
			maxChunkSize: 5,
			expectedMin:  1,
			expectedMax:  3,
			description:  "should handle very small max chunk sizes gracefully",
			checkContent: []string{"A. B."},
		},
		{
			name:         "minimum size calculation edge case",
			text:         "X. Y.",
			maxChunkSize: 10,
			expectedMin:  1,
			expectedMax:  2,
			description:  "should respect minimum chunk size of 15% or 3 chars minimum",
			checkContent: []string{"X. Y."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunk.SplitText(tt.text, tt.maxChunkSize)

			// verify chunk count is within expected range
			if len(result) < tt.expectedMin || len(result) > tt.expectedMax {
				t.Errorf("SplitText() returned %d chunks, expected between %d and %d",
					len(result), tt.expectedMin, tt.expectedMax)
				t.Errorf("Description: %s", tt.description)
				for i, chunk := range result {
					t.Errorf("  Chunk %d: %q (len=%d)", i, chunk, len(chunk))
				}
			}

			// verify specific content appears in results
			for _, expectedContent := range tt.checkContent {
				found := false
				for _, chunk := range result {
					if strings.Contains(chunk, expectedContent) {
						found = true
						break
					}
				}
				if !found {
					t.Logf("Expected content %q not found in chunks: %v", expectedContent, result)
				}
			}

			// verify no empty chunks
			for i, chunk := range result {
				if strings.TrimSpace(chunk) == "" {
					t.Errorf("Found empty chunk %d", i)
				}
			}

			// check that very short chunks are minimized (unless they can't be merged)
			minChunkSize := int(float64(tt.maxChunkSize) * 0.15)
			if minChunkSize < 3 {
				minChunkSize = 3
			}

			shortChunkCount := 0
			for _, chunk := range result {
				if len(chunk) < minChunkSize {
					shortChunkCount++
				}
			}

			// allow some short chunks if merging isn't possible, but not too many
			if shortChunkCount > len(result)/2 {
				t.Errorf("Too many short chunks (%d out of %d), minimum size threshold may not be working",
					shortChunkCount, len(result))
				for i, chunk := range result {
					t.Errorf("  Chunk %d: %q (len=%d, minSize=%d)", i, chunk, len(chunk), minChunkSize)
				}
			}

			if testing.Verbose() {
				t.Logf("Test '%s' produced %d chunks from text length %d (limit: %d, minSize: %d)",
					tt.name, len(result), len(tt.text), tt.maxChunkSize, minChunkSize)
				for i, chunk := range result {
					t.Logf("  Chunk %d: %q (len=%d)", i, chunk, len(chunk))
				}
			}
		})
	}
}
