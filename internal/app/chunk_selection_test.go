package app

import (
	"strings"
	"testing"

	"github.com/chriscorrea/sift/internal/counter"
)

// test data for chunk selection tests
var testChunks = []string{
	"First chunk with five words here",     // 6 words
	"Second chunk has four words only",     // 6 words
	"Third chunk contains three words now", // 6 words
	"Fourth chunk has two words",           // 5 words
	"Fifth chunk one word",                 // 4 words
}

func TestNewChunkSelector(t *testing.T) {
	tests := []struct {
		name           string
		countingMethod counter.CountingMethod
		maxUnits       int
		strategy       SizingStrategy
		expectError    bool
	}{
		{
			name:           "valid word counter",
			countingMethod: counter.Words,
			maxUnits:       100,
			strategy:       Beginning,
			expectError:    false,
		},
		{
			name:           "valid token counter",
			countingMethod: counter.Tokens,
			maxUnits:       1000,
			strategy:       Middle,
			expectError:    false,
		},
		{
			name:           "valid character counter",
			countingMethod: counter.Characters,
			maxUnits:       2000,
			strategy:       End,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := NewChunkSelector(tt.countingMethod, tt.maxUnits, tt.strategy)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewChunkSelector() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewChunkSelector() unexpected error: %v", err)
				return
			}

			if selector == nil {
				t.Errorf("NewChunkSelector() returned nil selector")
				return
			}

			if selector.maxUnits != tt.maxUnits {
				t.Errorf("NewChunkSelector() maxUnits = %d, want %d", selector.maxUnits, tt.maxUnits)
			}

			if selector.strategy != tt.strategy {
				t.Errorf("NewChunkSelector() strategy = %v, want %v", selector.strategy, tt.strategy)
			}
		})
	}
}

func TestSizingStrategyString(t *testing.T) {
	tests := []struct {
		strategy SizingStrategy
		expected string
	}{
		{Beginning, "Beginning"},
		{Middle, "Middle"},
		{End, "End"},
		{SizingStrategy(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.strategy.String()
			if result != tt.expected {
				t.Errorf("SizingStrategy.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestChunkSelector_ApplySizeConstraints_Beginning(t *testing.T) {
	selector, err := NewChunkSelector(counter.Words, 12, Beginning)
	if err != nil {
		t.Fatalf("Failed to create ChunkSelector: %v", err)
	}

	result, err := selector.ApplySizeConstraints(testChunks)
	if err != nil {
		t.Fatalf("ApplySizeConstraints() error = %v", err)
	}

	if result == "" {
		t.Errorf("ApplySizeConstraints() returned empty result")
	}

	// should start with first chunks
	if !strings.Contains(result, "First chunk") {
		t.Errorf("Beginning strategy should include first chunk, got: %s", result)
	}

	// count words to verify constraint
	words := strings.Fields(result)
	if len(words) > 12 {
		t.Errorf("Result exceeds word limit: got %d words, want ≤ 12", len(words))
	}
}

func TestChunkSelector_ApplySizeConstraints_End(t *testing.T) {
	selector, err := NewChunkSelector(counter.Words, 10, End)
	if err != nil {
		t.Fatalf("Failed to create ChunkSelector: %v", err)
	}

	result, err := selector.ApplySizeConstraints(testChunks)
	if err != nil {
		t.Fatalf("ApplySizeConstraints() error = %v", err)
	}

	if result == "" {
		t.Errorf("ApplySizeConstraints() returned empty result")
	}

	// should end with last chunks
	if !strings.Contains(result, "Fifth chunk") {
		t.Errorf("End strategy should include last chunk, got: %s", result)
	}

	// count words to verify constraint
	words := strings.Fields(result)
	if len(words) > 10 {
		t.Errorf("Result exceeds word limit: got %d words, want ≤ 10", len(words))
	}
}

func TestChunkSelector_ApplySizeConstraints_Middle(t *testing.T) {
	selector, err := NewChunkSelector(counter.Words, 15, Middle)
	if err != nil {
		t.Fatalf("Failed to create ChunkSelector: %v", err)
	}

	result, err := selector.ApplySizeConstraints(testChunks)
	if err != nil {
		t.Fatalf("ApplySizeConstraints() error = %v", err)
	}

	if result == "" {
		t.Errorf("ApplySizeConstraints() returned empty result")
	}

	// should include middle chunk (index 2)
	if !strings.Contains(result, "Third chunk") {
		t.Errorf("Middle strategy should include middle chunk, got: %s", result)
	}

	// count words to verify constraint
	words := strings.Fields(result)
	if len(words) > 15 {
		t.Errorf("Result exceeds word limit: got %d words, want ≤ 15", len(words))
	}
}

func TestChunkSelector_ApplySizeConstraints_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		maxUnits int
		strategy SizingStrategy
	}{
		{
			name:     "empty chunks",
			chunks:   []string{},
			maxUnits: 10,
			strategy: Beginning,
		},
		{
			name:     "single chunk",
			chunks:   []string{"single chunk content"},
			maxUnits: 5,
			strategy: Middle,
		},
		{
			name:     "zero max units",
			chunks:   []string{"chunk1", "chunk2"},
			maxUnits: 0,
			strategy: Beginning,
		},
		{
			name:     "negative max units",
			chunks:   []string{"chunk1", "chunk2"},
			maxUnits: -5,
			strategy: Beginning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := NewChunkSelector(counter.Words, tt.maxUnits, tt.strategy)
			if err != nil {
				t.Fatalf("Failed to create ChunkSelector: %v", err)
			}

			// should not panic
			result, err := selector.ApplySizeConstraints(tt.chunks)
			if err != nil {
				t.Errorf("ApplySizeConstraints() unexpected error: %v", err)
			}

			// basic validation
			if len(tt.chunks) == 0 {
				if result != "" {
					t.Errorf("Expected empty result for empty input, got %q", result)
				}
			}

			if tt.maxUnits <= 0 && len(tt.chunks) > 0 {
				// should return concatenated original chunks when no size limit
				expected := strings.Join(tt.chunks, "\n\n")
				if result != expected && result != tt.chunks[0] { // single chunk case
					t.Errorf("Expected original content when maxUnits≤0, got %q", result)
				}
			}
		})
	}
}

func TestChunkSelector_removeOverlapPrefix(t *testing.T) {
	selector, _ := NewChunkSelector(counter.Words, 100, Middle)

	tests := []struct {
		name         string
		currentChunk string
		prevChunk    string
		expected     string
	}{
		{
			name:         "no overlap",
			currentChunk: "current chunk content",
			prevChunk:    "previous chunk content",
			expected:     "current chunk content",
		},
		{
			name:         "exact word overlap",
			currentChunk: "overlap content here",
			prevChunk:    "some text overlap content",
			expected:     "here",
		},
		{
			name:         "full overlap",
			currentChunk: "same content",
			prevChunk:    "prefix same content",
			expected:     "",
		},
		{
			name:         "empty chunks",
			currentChunk: "",
			prevChunk:    "previous",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selector.removeOverlapPrefix(tt.currentChunk, tt.prevChunk)
			if result != tt.expected {
				t.Errorf("removeOverlapPrefix() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestChunkSelector_slicesEqual(t *testing.T) {
	selector, _ := NewChunkSelector(counter.Words, 100, Middle)

	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "equal slices",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "different length",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "different content",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "x", "c"},
			expected: false,
		},
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selector.slicesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("slicesEqual() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestChunkSelector_Integration(t *testing.T) {
	// test the full integration with different counting methods
	testContent := []string{
		"The quick brown fox jumps over the lazy dog.",
		"This is a second sentence with more words for testing purposes.",
		"Finally, this is the third and last sentence in our test data.",
	}

	tests := []struct {
		name           string
		countingMethod counter.CountingMethod
		maxUnits       int
		strategy       SizingStrategy
	}{
		{
			name:           "word counting with beginning strategy",
			countingMethod: counter.Words,
			maxUnits:       15,
			strategy:       Beginning,
		},
		{
			name:           "character counting with middle strategy",
			countingMethod: counter.Characters,
			maxUnits:       100,
			strategy:       Middle,
		},
		{
			name:           "token counting with end strategy",
			countingMethod: counter.Tokens,
			maxUnits:       25,
			strategy:       End,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := NewChunkSelector(tt.countingMethod, tt.maxUnits, tt.strategy)
			if err != nil {
				t.Fatalf("Failed to create ChunkSelector: %v", err)
			}

			result, err := selector.ApplySizeConstraints(testContent)
			if err != nil {
				t.Fatalf("ApplySizeConstraints() error = %v", err)
			}

			if result == "" {
				t.Errorf("ApplySizeConstraints() returned empty result")
			}

			// verify size constraints with some tolerance for partial chunks
			actualUnits := selector.counter.Count(result)
			tolerance := tt.maxUnits / 10 // 10% tolerance for partial chunk rounding
			if actualUnits > tt.maxUnits+tolerance {
				t.Errorf("Result significantly exceeds unit limit: got %d %s, want ≤ %d (tolerance: %d)",
					actualUnits, selector.counter.Name(), tt.maxUnits, tolerance)
			}
		})
	}
}

func TestChunkSelector_PrepareChunks(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		maxUnits       int
		countingMethod counter.CountingMethod
		expectedChunks int // approximate number of chunks expected
	}{
		{
			name:           "no size constraint",
			text:           "This is a test document with multiple sentences.",
			maxUnits:       0,
			countingMethod: counter.Words,
			expectedChunks: 1, // entire text as single chunk
		},
		{
			name:           "token-based chunking",
			text:           strings.Repeat("Sift your flour very carefully. ", 200), // 5 X 200 = 1000 words
			maxUnits:       500,
			countingMethod: counter.Tokens,
			expectedChunks: 2, // should create multiple chunks
		},
		{
			name:           "word-based chunking",
			text:           strings.Repeat("sugar ", 500), // 500 words
			maxUnits:       100,
			countingMethod: counter.Words,
			expectedChunks: 2, // should create multiple chunks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunkSelector, err := NewChunkSelector(tt.countingMethod, tt.maxUnits, Beginning)
			if err != nil {
				t.Fatalf("Failed to create ChunkSelector: %v", err)
			}

			chunks := chunkSelector.PrepareChunks(tt.text)

			// basic validation for some reasonable output
			if len(chunks) == 0 {
				t.Errorf("PrepareChunks() returned no chunks")
			}

			// verify content is reasonably preserved (allow some chunking artifacts)
			totalContent := strings.Join(chunks, "")
			originalWords := strings.Fields(tt.text)
			if len(originalWords) > 0 && len(totalContent) < len(tt.text)/2 {
				t.Errorf("PrepareChunks() significantly reduced content: got %d chars, expected at least %d", len(totalContent), len(tt.text)/2)
			}
		})
	}
}

func TestChunkSelector_SelectAndSizeChunks(t *testing.T) {
	// Create test data
	allChunks := []string{
		"chunk0", "chunk1", "chunk2", "chunk3", "chunk4", "chunk5",
	}

	scoredChunks := []ChunkScore{
		{Chunk: "chunk2", Score: 0.9, Index: 2}, // highest score
		{Chunk: "chunk4", Score: 0.7, Index: 4}, // second highest
		{Chunk: "chunk1", Score: 0.5, Index: 1}, // third highest
		{Chunk: "chunk0", Score: 0.3, Index: 0},
		{Chunk: "chunk3", Score: 0.2, Index: 3},
		{Chunk: "chunk5", Score: 0.1, Index: 5},
	}

	tests := []struct {
		name      string
		maxUnits  int
		minChunks int // minimum number of chunks we expect in result
	}{
		{
			name:      "small limit",
			maxUnits:  50,
			minChunks: 1,
		},
		{
			name:      "medium limit",
			maxUnits:  200,
			minChunks: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunkSelector, err := NewChunkSelector(counter.Words, tt.maxUnits, Beginning)
			if err != nil {
				t.Fatalf("Failed to create ChunkSelector: %v", err)
			}

			// use the new unified pathway: PrepareForSearch + Select
			orderedChunks := chunkSelector.PrepareForSearch(scoredChunks)
			result, err := chunkSelector.Select(orderedChunks, allChunks, 1, 2) // Default context
			if err != nil {
				t.Errorf("Select() failed: %v", err)
			}

			if len(result) == 0 {
				t.Errorf("Select() returned empty result")
			}
		})
	}
}

func TestChunkSelector_StrategyOrder(t *testing.T) {
	testChunks := []string{"chunk0", "chunk1", "chunk2", "chunk3", "chunk4"}

	tests := []struct {
		name     string
		strategy SizingStrategy
		maxUnits int
		expected []string // expected order for first few chunks
	}{
		{
			name:     "Beginning strategy",
			strategy: Beginning,
			maxUnits: 100,
			expected: []string{"chunk0", "chunk1", "chunk2", "chunk3", "chunk4"},
		},
		{
			name:     "End strategy",
			strategy: End,
			maxUnits: 100,
			expected: []string{"chunk4", "chunk3", "chunk2", "chunk1", "chunk0"},
		},
		{
			name:     "Middle strategy",
			strategy: Middle,
			maxUnits: 100,
			expected: []string{"chunk2", "chunk3", "chunk1", "chunk4", "chunk0"}, // Middle-out order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunkSelector, err := NewChunkSelector(counter.Words, tt.maxUnits, tt.strategy)
			if err != nil {
				t.Fatalf("Failed to create ChunkSelector: %v", err)
			}

			// use the new unified pathway: PrepareForStrategy
			orderedChunks := chunkSelector.PrepareForStrategy(testChunks)

			// for strategy testing, we'll verify that we get the expected order
			if len(orderedChunks) != len(tt.expected) {
				t.Errorf("PrepareForStrategy() returned %d chunks, expected %d", len(orderedChunks), len(tt.expected))
			}

			// check at least the first few chunks match expected order
			checkCount := min(3, min(len(orderedChunks), len(tt.expected)))
			for i := 0; i < checkCount; i++ {
				if orderedChunks[i].Text != tt.expected[i] {
					t.Errorf("PrepareForStrategy() chunk %d = %q, want %q", i, orderedChunks[i].Text, tt.expected[i])
				}
			}
		})
	}
}

func TestChunkSelector_OutputOrderWithSizeConstraints(t *testing.T) {
	// test that final output is in document order regardless of strategy
	testChunks := []string{"chunk0", "chunk1", "chunk2", "chunk3", "chunk4"}

	for i, chunk := range testChunks {
		words := strings.Fields(chunk)
		t.Logf("Chunk %d (%q) has %d words: %v", i, chunk, len(words), words)
	}

	tests := []struct {
		name           string
		strategy       SizingStrategy
		maxWords       int
		expectedOutput string   // expected final output (should be in document order)
		expectedChunks []string // expected chunks included (for verification)
	}{
		{
			name:           "Beginning strategy with 2 chunks",
			strategy:       Beginning,
			maxWords:       2, // 2 words total - should fit exactly 2 chunks
			expectedOutput: "chunk0\n\nchunk1",
			expectedChunks: []string{"chunk0", "chunk1"},
		},
		{
			name:           "End strategy with 2 chunks",
			strategy:       End,
			maxWords:       2,
			expectedOutput: "chunk3\n\nchunk4", // should be last 2 chunks *in document order*
			expectedChunks: []string{"chunk3", "chunk4"},
		},
		{
			name:           "Middle strategy with 2 chunks",
			strategy:       Middle,
			maxWords:       2,                  // 2 words total
			expectedOutput: "chunk2\n\nchunk3", // should be middle chunks *in document order*
			expectedChunks: []string{"chunk2", "chunk3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunkSelector, err := NewChunkSelector(counter.Words, tt.maxWords, tt.strategy)
			if err != nil {
				t.Fatalf("Failed to create ChunkSelector: %v", err)
			}

			// apply size constraints and get final output
			result, err := chunkSelector.ApplySizeConstraints(testChunks)
			if err != nil {
				t.Fatalf("ApplySizeConstraints() error = %v", err)
			}

			t.Logf("Strategy %s with %d words produced: %q", tt.strategy, tt.maxWords, result)

			// verify final output is in document order
			if result != tt.expectedOutput {
				t.Errorf("ApplySizeConstraints() output = %q, want %q", result, tt.expectedOutput)
			}

			// verify the correct chunks are included
			for _, expectedChunk := range tt.expectedChunks {
				if !strings.Contains(result, expectedChunk) {
					t.Errorf("Expected chunk %q not found in output: %q", expectedChunk, result)
				}
			}

			// verify chunks appear in document order (low index chunks before high index chunks)
			for i := 0; i < len(tt.expectedChunks)-1; i++ {
				chunk1Pos := strings.Index(result, tt.expectedChunks[i])
				chunk2Pos := strings.Index(result, tt.expectedChunks[i+1])
				if chunk1Pos >= chunk2Pos {
					t.Errorf("Chunks not in document order: %q should appear before %q in output",
						tt.expectedChunks[i], tt.expectedChunks[i+1])
				}
			}
		})
	}
}

func TestChunkSelector_SelectWithContextWindows(t *testing.T) {
	// test the Select function's context window behavior with various settings
	testChunks := []string{"A", "B", "C", "D", "E", "F", "G"}

	tests := []struct {
		name           string
		targetIndices  []int    // chunks to target for selection
		contextBefore  int      // # chunks to include before each target
		contextAfter   int      // # of chunks to include after each target
		maxWords       int      // size limit
		expectedChunks []string // expected chunks in final output (in document order)
		expectedOutput string   // expected formatted output
	}{
		{
			name:           "No context - single target",
			targetIndices:  []int{3}, // target chunk "D"
			contextBefore:  0,
			contextAfter:   0,
			maxWords:       10,
			expectedChunks: []string{"D"},
			expectedOutput: "D",
		},
		{
			name:           "Before context only - single target",
			targetIndices:  []int{3}, // target chunk "D"
			contextBefore:  2,
			contextAfter:   0,
			maxWords:       10,
			expectedChunks: []string{"B", "C", "D"}, // 2 before + target
			expectedOutput: "B\n\nC\n\nD",
		},
		{
			name:           "After context only - single target",
			targetIndices:  []int{3}, // target chunk "D"
			contextBefore:  0,
			contextAfter:   2,
			maxWords:       10,
			expectedChunks: []string{"D", "E", "F"}, // target + 2 after
			expectedOutput: "D\n\nE\n\nF",
		},
		{
			name:           "Symmetric context - single target",
			targetIndices:  []int{3}, // Target chunk "D"
			contextBefore:  1,
			contextAfter:   1,
			maxWords:       10,
			expectedChunks: []string{"C", "D", "E"}, // 1 before + target + 1 after
			expectedOutput: "C\n\nD\n\nE",
		},
		{
			name:           "Context at beginning boundary",
			targetIndices:  []int{0}, // target chunk "A" (first chunk)
			contextBefore:  2,        // should not go below index 0
			contextAfter:   1,
			maxWords:       10,
			expectedChunks: []string{"A", "B"}, // no chunks before index 0
			expectedOutput: "A\n\nB",
		},
		{
			name:           "Context at end boundary",
			targetIndices:  []int{6}, // target chunk "G" (last chunk)
			contextBefore:  1,
			contextAfter:   2, // should not go beyond last index
			maxWords:       10,
			expectedChunks: []string{"F", "G"}, // o chunks after last index
			expectedOutput: "F\n\nG",
		},
		{
			name:           "Multiple targets with overlapping context",
			targetIndices:  []int{2, 4}, // target chunks "C" and "E"
			contextBefore:  1,
			contextAfter:   1,
			maxWords:       10,
			expectedChunks: []string{"B", "C", "D", "E", "F"}, // overlapping context merged
			expectedOutput: "B\n\nC\n\nD\n\nE\n\nF",
		},
		{
			name:           "Context with size limit - cuts off excess",
			targetIndices:  []int{3}, // target chunk "D"
			contextBefore:  2,
			contextAfter:   2,
			maxWords:       3,                       // limit to 3 words total
			expectedChunks: []string{"B", "C", "D"}, // should include target + some context
			expectedOutput: "B\n\nC\n\nD",
		},
		{
			name:           "Large context window",
			targetIndices:  []int{3}, // target chunk "D"
			contextBefore:  5,        // more than available chunks before
			contextAfter:   5,        // more than available chunks after
			maxWords:       10,
			expectedChunks: []string{"A", "B", "C", "D", "E", "F", "G"}, // all chunks
			expectedOutput: "A\n\nB\n\nC\n\nD\n\nE\n\nF\n\nG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunkSelector, err := NewChunkSelector(counter.Words, tt.maxWords, Beginning)
			if err != nil {
				t.Fatalf("Failed to create ChunkSelector: %v", err)
			}

			// create ordered chunks from target indices
			orderedChunks := make([]ChunkWithIndex, len(tt.targetIndices))
			for i, targetIdx := range tt.targetIndices {
				orderedChunks[i] = ChunkWithIndex{
					Text:  testChunks[targetIdx],
					Index: targetIdx,
				}
			}

			// call Select with specified context window
			result, err := chunkSelector.Select(orderedChunks, testChunks, tt.contextBefore, tt.contextAfter)
			if err != nil {
				t.Fatalf("Select() error = %v", err)
			}

			// verify expected output
			if result != tt.expectedOutput {
				t.Errorf("Select() output = %q, want %q", result, tt.expectedOutput)
			}

			// verify all expected chunks are present
			for _, expectedChunk := range tt.expectedChunks {
				if !strings.Contains(result, expectedChunk) {
					t.Errorf("Expected chunk %q not found in output: %q", expectedChunk, result)
				}
			}

			// verify chunks appear in document order
			for i := 0; i < len(tt.expectedChunks)-1; i++ {
				chunk1Pos := strings.Index(result, tt.expectedChunks[i])
				chunk2Pos := strings.Index(result, tt.expectedChunks[i+1])
				if chunk1Pos >= chunk2Pos {
					t.Errorf("Chunks not in document order: %q should appear before %q",
						tt.expectedChunks[i], tt.expectedChunks[i+1])
				}
			}

			// verify no unexpected chunks are included
			resultChunks := strings.Split(result, "\n\n")
			if len(resultChunks) != len(tt.expectedChunks) {
				t.Errorf("Result has %d chunks, expected %d chunks. Result chunks: %v",
					len(resultChunks), len(tt.expectedChunks), resultChunks)
			}
		})
	}
}
