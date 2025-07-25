package app

import (
	"strings"
	"testing"

	"github.com/chriscorrea/bm25md"
	"github.com/chriscorrea/sift/internal/counter"
)

func TestContextCalculator_FieldDetection(t *testing.T) {
	// create a mock counter for testing
	textCounter, err := counter.NewCounter(counter.Words)
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	calculator, err := NewContextCalculator(textCounter, 100)
	if err != nil {
		t.Fatalf("Failed to create context calculator: %v", err)
	}

	tests := []struct {
		name           string
		input          string
		expectedField  bm25md.Field
		expectedIsList bool
		description    string
	}{
		// Header detection tests
		{
			name:           "H1 header",
			input:          "# Main Title",
			expectedField:  bm25md.FieldH1,
			expectedIsList: false,
			description:    "Should detect H1 header with single #",
		},
		{
			name:           "H2 header",
			input:          "## Section Header",
			expectedField:  bm25md.FieldH2,
			expectedIsList: false,
			description:    "Should detect H2 header with double ##",
		},
		{
			name:           "Not a header - missing space",
			input:          "#hashtag content",
			expectedField:  bm25md.FieldBody,
			expectedIsList: false,
			description:    "Should not detect header without space after #",
		},

		// list detection (supports -, *, and + markers)
		{
			name:           "Bullet list",
			input:          "- First bullet item",
			expectedField:  bm25md.FieldBody,
			expectedIsList: true,
			description:    "Should detect list item (-, *, or + markers)",
		},

		// Numbered list detection
		{
			name:           "Numbered list",
			input:          "1. First numbered item",
			expectedField:  bm25md.FieldBody,
			expectedIsList: true,
			description:    "Should detect numbered list item",
		},
		{
			name:           "Not a numbered list - no space",
			input:          "1.No space after period",
			expectedField:  bm25md.FieldBody,
			expectedIsList: false,
			description:    "Should not detect numbered list without trailing space",
		},

		// code block detection
		{
			name:           "Code block",
			input:          "```python\ndef hello():\n    print('world')",
			expectedField:  bm25md.FieldCode,
			expectedIsList: false,
			description:    "Should detect code block",
		},
		{
			name:           "Inline code",
			input:          "Use the `print()` function to output text",
			expectedField:  bm25md.FieldCode,
			expectedIsList: false,
			description:    "Should detect inline code",
		},

		// emphasis text formatting
		{
			name:           "Bold text",
			input:          "This is **very important** information",
			expectedField:  bm25md.FieldBold,
			expectedIsList: false,
			description:    "Should detect bold formatting",
		},
		{
			name:           "Italic text",
			input:          "This is *emphasized* text",
			expectedField:  bm25md.FieldItalic,
			expectedIsList: false,
			description:    "Should detect italic formatting",
		},

		// edge cases
		{
			name:           "Empty chunk",
			input:          "",
			expectedField:  bm25md.FieldBody,
			expectedIsList: false,
			description:    "Should handle empty chunks gracefully",
		},
		{
			name:           "Plain body text",
			input:          "This is regular paragraph text without any special formatting.",
			expectedField:  bm25md.FieldBody,
			expectedIsList: false,
			description:    "Should detect plain body text as default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.detectPrimaryFieldType(tt.input)

			if result.Primary != tt.expectedField {
				t.Errorf("Field detection mismatch for %q:\n  Expected field: %v\n  Got field: %v\n  Description: %s",
					tt.input, tt.expectedField, result.Primary, tt.description)
			}

			if result.IsList != tt.expectedIsList {
				t.Errorf("List detection mismatch for %q:\n  Expected IsList: %v\n  Got IsList: %v\n  Description: %s",
					tt.input, tt.expectedIsList, result.IsList, tt.description)
			}
		})
	}
}

func TestContextCalculator_ContextStrategies(t *testing.T) {
	textCounter, err := counter.NewCounter(counter.Words)
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	calculator, err := NewContextCalculator(textCounter, 100)
	if err != nil {
		t.Fatalf("Failed to create context calculator: %v", err)
	}

	tests := []struct {
		name           string
		fieldType      ChunkFieldType
		expectedBefore float64
		expectedAfter  float64
		expectedName   string
		description    string
	}{
		{
			name:           "Header strategy - following emphasis",
			fieldType:      ChunkFieldType{Primary: bm25md.FieldH1, IsList: false},
			expectedBefore: 0.2,
			expectedAfter:  0.8,
			expectedName:   "header-following",
			description:    "Headers should include more context after",
		},
		{
			name:           "List strategy - preceding emphasis",
			fieldType:      ChunkFieldType{Primary: bm25md.FieldBody, IsList: true},
			expectedBefore: 0.8,
			expectedAfter:  0.2,
			expectedName:   "list-preceding",
			description:    "Lists should include more context before",
		},
		{
			name:           "Code strategy - following emphasis",
			fieldType:      ChunkFieldType{Primary: bm25md.FieldCode, IsList: false},
			expectedBefore: 0.3,
			expectedAfter:  0.7,
			expectedName:   "code-following",
			description:    "Code should include more context after",
		},
		{
			name:           "Body strategy - balanced",
			fieldType:      ChunkFieldType{Primary: bm25md.FieldBody, IsList: false},
			expectedBefore: 0.5,
			expectedAfter:  0.5,
			expectedName:   "balanced",
			description:    "Body text should include balanced surrounding context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := calculator.getContextStrategy(tt.fieldType)

			if strategy.BeforeRatio != tt.expectedBefore {
				t.Errorf("BeforeRatio mismatch:\n  Expected: %v\n  Got: %v\n  Description: %s",
					tt.expectedBefore, strategy.BeforeRatio, tt.description)
			}

			if strategy.AfterRatio != tt.expectedAfter {
				t.Errorf("AfterRatio mismatch:\n  Expected: %v\n  Got: %v\n  Description: %s",
					tt.expectedAfter, strategy.AfterRatio, tt.description)
			}

			if strategy.Name != tt.expectedName {
				t.Errorf("Strategy name mismatch:\n  Expected: %q\n  Got: %q\n  Description: %s",
					tt.expectedName, strategy.Name, tt.description)
			}

			// verify ratios sum to 1.0
			total := strategy.BeforeRatio + strategy.AfterRatio
			if total != 1.0 {
				t.Errorf("Strategy ratios don't sum to 1.0:\n  BeforeRatio: %v\n  AfterRatio: %v\n  Total: %v",
					strategy.BeforeRatio, strategy.AfterRatio, total)
			}
		})
	}
}

func TestContextCalculator_TokenBudgetDistribution(t *testing.T) {
	textCounter, err := counter.NewCounter(counter.Words)
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	tests := []struct {
		name             string
		maxContextUnits  int
		targetChunk      string
		allChunks        []string
		targetIndex      int
		expectedMaxUnits int
		description      string
	}{
		{
			name:             "Basic budget",
			maxContextUnits:  100,
			targetChunk:      "This is the target chunk with some content",
			allChunks:        []string{"Before chunk", "This is the target chunk with some content", "After chunk"},
			targetIndex:      1,
			expectedMaxUnits: 100,
			description:      "Should respect the maximum context units budget",
		},
		{
			name:             "Target chunk exceeds budget",
			maxContextUnits:  5,
			targetChunk:      "This is a very long target chunk with many words that exceeds the budget",
			allChunks:        []string{"Before", "This is a very long target chunk with many words that exceeds the budget", "After"},
			targetIndex:      1,
			expectedMaxUnits: 5,
			description:      "Should handle cases where target chunk alone exceeds budget",
		},
		{
			name:             "Small budget with context",
			maxContextUnits:  20,
			targetChunk:      "Target chunk",
			allChunks:        []string{"Before context chunk", "Target chunk", "After context chunk"},
			targetIndex:      1,
			expectedMaxUnits: 20,
			description:      "Should distribute small budget across target and context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculator, err := NewContextCalculator(textCounter, tt.maxContextUnits)
			if err != nil {
				t.Fatalf("Failed to create context calculator: %v", err)
			}

			request := ContextRequest{
				TargetChunk: ChunkWithIndex{
					Text:  tt.targetChunk,
					Index: tt.targetIndex,
					Score: 1.0,
				},
				AllChunks: tt.allChunks,
			}

			result := calculator.CalculateSmartContext(request)

			// verify total units don't exceed the budget
			if result.TotalUnits > tt.maxContextUnits {
				t.Errorf("Total units exceed budget:\n  Budget: %d\n  Total: %d\n  Description: %s",
					tt.maxContextUnits, result.TotalUnits, tt.description)
			}

			// verify at least the target chunk is included
			if len(result.SelectedChunks) == 0 {
				t.Errorf("No chunks selected:\n  Description: %s", tt.description)
			}

			// verify target chunk is included
			targetFound := false
			for _, chunk := range result.SelectedChunks {
				if chunk.Index == tt.targetIndex {
					targetFound = true
					break
				}
			}
			if !targetFound {
				t.Errorf("Target chunk not found in selected chunks:\n  Description: %s", tt.description)
			}
		})
	}
}

func TestContextCalculator_BudgetParameter(t *testing.T) {
	// test that CalculateSmartContextWithBudget correctly uses the budget param
	textCounter, err := counter.NewCounter(counter.Words)
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	calculator, err := NewContextCalculator(textCounter, 100) // default budget
	if err != nil {
		t.Fatalf("Failed to create context calculator: %v", err)
	}

	chunks := []string{
		"First chunk with some content",
		"# Header Chunk",
		"Third chunk with more content",
		"Fourth chunk here",
	}

	targetChunk := ChunkWithIndex{
		Text:  chunks[1],
		Index: 1,
		Score: 0.9,
	}

	request := ContextRequest{
		TargetChunk: targetChunk,
		AllChunks:   chunks,
	}

	tests := []struct {
		name           string
		budgetUnits    int
		expectIncludes []int // chunk indices we expect to be included
		description    string
	}{
		{
			name:           "Small budget - target only",
			budgetUnits:    3, // just enough for target
			expectIncludes: []int{1},
			description:    "Should only include target chunk when budget is minimal",
		},
		{
			name:           "Medium budget - target plus some context",
			budgetUnits:    15,
			expectIncludes: []int{1, 2}, // header gets following context
			description:    "Should include target and following context for header",
		},
		{
			name:           "Large budget - more context",
			budgetUnits:    30,
			expectIncludes: []int{1, 2, 3}, // more following context w/ larger budget
			description:    "Should include more following context with larger budget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.CalculateSmartContextWithBudget(request, tt.budgetUnits)

			// check that expected chunks are included
			includedIndices := make(map[int]bool)
			for _, chunk := range result.SelectedChunks {
				includedIndices[chunk.Index] = true
			}

			for _, expectedIndex := range tt.expectIncludes {
				if !includedIndices[expectedIndex] {
					t.Errorf("Expected chunk %d to be included but it wasn't: %s", expectedIndex, tt.description)
				}
			}

			// verify total units don't exceed budget
			if result.TotalUnits > tt.budgetUnits {
				t.Errorf("Total units %d exceeds budget %d: %s", result.TotalUnits, tt.budgetUnits, tt.description)
			}
		})
	}
}

func TestContextCalculator_PartialChunkTruncation(t *testing.T) {
	textCounter, err := counter.NewCounter(counter.Words)
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	calculator, err := NewContextCalculator(textCounter, 100)
	if err != nil {
		t.Fatalf("Failed to create context calculator: %v", err)
	}

	tests := []struct {
		name           string
		chunkText      string
		remainingUnits int
		expectPartial  bool
		description    string
	}{
		{
			name:           "Chunk truncation and fitting",
			chunkText:      "This is a very long chunk with many words that should be truncated to fit within the remaining budget allocation",
			remainingUnits: 5,
			expectPartial:  true,
			description:    "Should create partial chunk when full chunk exceeds budget, or return full chunk when it fits",
		},
		{
			name:           "Zero budget",
			chunkText:      "Any chunk text",
			remainingUnits: 0,
			expectPartial:  false,
			description:    "Should return empty string when no budget remaining",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.createPartialChunk(tt.chunkText, tt.remainingUnits)

			if tt.remainingUnits <= 0 {
				if result != "" {
					t.Errorf("Expected empty result for zero budget, got: %q", result)
				}
				return
			}

			resultWordCount := len(strings.Fields(result))
			originalWordCount := len(strings.Fields(tt.chunkText))

			if tt.expectPartial {
				if resultWordCount == 0 {
					t.Errorf("Expected partial chunk but got empty result:\n  Original: %q\n  Budget: %d\n  Description: %s",
						tt.chunkText, tt.remainingUnits, tt.description)
				}
				if resultWordCount > tt.remainingUnits {
					t.Errorf("Partial chunk exceeds budget:\n  Result: %q\n  Word count: %d\n  Budget: %d\n  Description: %s",
						result, resultWordCount, tt.remainingUnits, tt.description)
				}
				if resultWordCount >= originalWordCount {
					t.Errorf("Expected truncation but got full chunk:\n  Original words: %d\n  Result words: %d\n  Description: %s",
						originalWordCount, resultWordCount, tt.description)
				}
			} else {
				if originalWordCount <= tt.remainingUnits {
					// should include the full chunk if it fits
					if resultWordCount != originalWordCount {
						t.Errorf("Expected full chunk when it fits budget:\n  Original words: %d\n  Result words: %d\n  Budget: %d\n  Description: %s",
							originalWordCount, resultWordCount, tt.remainingUnits, tt.description)
					}
				}
			}
		})
	}
}

func TestContextCalculator_PreciseTokenTruncation(t *testing.T) {
	tokenCounter, err := counter.NewCounter(counter.Tokens)
	if err != nil {
		t.Fatalf("Failed to create token counter: %v", err)
	}

	calculator, err := NewContextCalculator(tokenCounter, 100)
	if err != nil {
		t.Fatalf("Failed to create context calculator: %v", err)
	}

	tests := []struct {
		name           string
		chunkText      string
		remainingUnits int
		description    string
	}{
		{
			name:           "Precise token truncation",
			chunkText:      "This is a test sentence with various punctuation marks, numbers like 358, and some technical terms like HTTP and JSON that should be tokenized precisely.",
			remainingUnits: 10,
			description:    "Should create partial chunk with exactly 10 tokens",
		},
		{
			name:           "Single token",
			chunkText:      "Hello world this is a longer sentence",
			remainingUnits: 1,
			description:    "Should create partial chunk with exactly 1 token",
		},
		{
			name:           "Code-like content",
			chunkText:      "function calculateSum(a, b) { return a + b; } // This is a JavaScript function",
			remainingUnits: 5,
			description:    "Should tokenize code content accurately",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// get original token count
			originalTokens := tokenCounter.Count(tt.chunkText)

			// create partial chunk
			result := calculator.createPartialChunk(tt.chunkText, tt.remainingUnits)

			if result == "" && tt.remainingUnits > 0 {
				t.Errorf("Expected non-empty result for budget %d, got empty string", tt.remainingUnits)
				return
			}

			// verify the result has exactly the requested number of tokens
			resultTokens := tokenCounter.Count(result)

			if originalTokens <= tt.remainingUnits {
				// full chunk should fit
				if resultTokens != originalTokens {
					t.Errorf("Full chunk should be returned when it fits:\n  Original tokens: %d\n  Result tokens: %d\n  Budget: %d\n  Result: %q",
						originalTokens, resultTokens, tt.remainingUnits, result)
				}
			} else {
				// partial chunk expected
				if resultTokens != tt.remainingUnits {
					t.Errorf("Partial chunk should have exactly %d tokens:\n  Got: %d tokens\n  Budget: %d\n  Result: %q\n  Description: %s",
						tt.remainingUnits, resultTokens, tt.remainingUnits, result, tt.description)
				}

				// result should be shorter than original
				if resultTokens >= originalTokens {
					t.Errorf("Expected truncation but got full or longer result:\n  Original tokens: %d\n  Result tokens: %d",
						originalTokens, resultTokens)
				}
			}
		})
	}
}

func TestContextCalculator_PreciseCharacterTruncation(t *testing.T) {
	charCounter, err := counter.NewCounter(counter.Characters)
	if err != nil {
		t.Fatalf("Failed to create character counter: %v", err)
	}

	calculator, err := NewContextCalculator(charCounter, 100)
	if err != nil {
		t.Fatalf("Failed to create context calculator: %v", err)
	}

	tests := []struct {
		name           string
		chunkText      string
		remainingUnits int
		description    string
	}{
		{
			name:           "Character truncation with word boundary",
			chunkText:      "This is a test sentence that should be truncated at word boundaries when possible.",
			remainingUnits: 25,
			description:    "Should truncate at word boundary when possible",
		},
		{
			name:           "Character truncation exact limit",
			chunkText:      "Short text",
			remainingUnits: 5,
			description:    "Should truncate to exact character limit",
		},
		{
			name:           "No truncation needed",
			chunkText:      "Short",
			remainingUnits: 10,
			description:    "Should return full text when it fits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalLength := len(tt.chunkText)
			result := calculator.createPartialChunk(tt.chunkText, tt.remainingUnits)

			if originalLength <= tt.remainingUnits {
				// full text should be returned
				if result != tt.chunkText {
					t.Errorf("Expected full text when it fits:\n  Expected: %q\n  Got: %q", tt.chunkText, result)
				}
			} else {
				// partial text expected
				if len(result) > tt.remainingUnits {
					t.Errorf("Result exceeds character limit:\n  Length: %d\n  Limit: %d\n  Result: %q",
						len(result), tt.remainingUnits, result)
				}

				if len(result) == 0 && tt.remainingUnits > 0 {
					t.Errorf("Expected non-empty result for budget %d", tt.remainingUnits)
				}
			}
		})
	}
}

func TestContextCalculator_RegexPatternSingleton(t *testing.T) {
	// verify that regex patterns are shared across instances
	textCounter, err := counter.NewCounter(counter.Words)
	if err != nil {
		t.Fatalf("Failed to create counter: %v", err)
	}

	calc1, err := NewContextCalculator(textCounter, 100)
	if err != nil {
		t.Fatalf("Failed to create first calculator: %v", err)
	}

	calc2, err := NewContextCalculator(textCounter, 200)
	if err != nil {
		t.Fatalf("Failed to create second calculator: %v", err)
	}

	// both calculators should use the same regex patterns instance
	if calc1.patterns != calc2.patterns {
		t.Error("Context calculators should share the same regex patterns instance")
	}

	// verify patterns are properly initialized
	if calc1.patterns.headerRegex == nil {
		t.Error("Header regex should be initialized")
	}
	if calc1.patterns.bulletListRegex == nil {
		t.Error("Bullet list regex should be initialized")
	}
	if calc1.patterns.numberListRegex == nil {
		t.Error("Number list regex should be initialized")
	}
}
