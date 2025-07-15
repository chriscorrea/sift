// This file consolidates chunk selection, sizing strategies, and output formatting
package app

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/chriscorrea/sift/internal/chunk"
	"github.com/chriscorrea/sift/internal/counter"
)

// ChunkingConfig centralizes all chunk sizing parameters to eliminate hard-coded constants
type ChunkingConfig struct {
	// base chunk sizes for different counting methods
	BaseTokenSize int
	BaseWordSize  int
	BaseCharSize  int

	// large text thresholds and multipliers
	TokenTextThreshold int
	WordTextThreshold  int
	CharTextThreshold  int

	// size multipliers for looong text
	LargeTextMultiplier float64
}

// DefaultChunkingConfig provides semantic-aware chunk sizing configuration
func DefaultChunkingConfig() ChunkingConfig {
	return ChunkingConfig{
		BaseTokenSize:       120, // ~120 tokens per chunk (typical paragraph)
		BaseWordSize:        150, // ~150 words per chunk
		BaseCharSize:        700, // ~700 characters per chunk
		TokenTextThreshold:  2500,
		WordTextThreshold:   1800,
		CharTextThreshold:   9500,
		LargeTextMultiplier: 1.5, // gentle scaling factor for large text
	}
}

// SizingStrategy defines how chunks are selected when applying size constraints
type SizingStrategy int

const (
	// beginning selects chunks from the start of the document
	Beginning SizingStrategy = iota
	// middle selects chunks from the middle outward (current default behavior)
	Middle
	// end selects chunks from the end of the document
	End
)

// String returns the string representation of the sizing strategy
func (s SizingStrategy) String() string {
	switch s {
	case Beginning:
		return "Beginning"
	case Middle:
		return "Middle"
	case End:
		return "End"
	default:
		return "Unknown"
	}
}

// ChunkWithIndex pairs a chunk of text with its original document index
type ChunkWithIndex struct {
	Text  string
	Index int
}

// ChunkSelector handles chunk selection and sizing using configurable strategies
type ChunkSelector struct {
	counter              counter.Counter
	maxUnits             int
	strategy             SizingStrategy
	config               ChunkingConfig
	defaultContextBefore int // default context before chunks for non-search scenarios
	defaultContextAfter  int // default context after chunks for non-search scenarios
}

// NewChunkSelector creates a new ChunkSelector with the specified configuration
func NewChunkSelector(countingMethod counter.CountingMethod, maxUnits int, strategy SizingStrategy) (*ChunkSelector, error) {
	textCounter, err := counter.NewCounter(countingMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to create counter: %w", err)
	}

	return &ChunkSelector{
		counter:              textCounter,
		maxUnits:             maxUnits,
		strategy:             strategy,
		config:               DefaultChunkingConfig(),
		defaultContextBefore: 0, // no context by default for non-search scenarios
		defaultContextAfter:  0, // no context by default for non-search scenarios
	}, nil
}

// PrepareChunks breaks text into manageable chunks w/ unit-aware sizing
// TODO: Implement streaming chunking for large documents
func (cs *ChunkSelector) PrepareChunks(text string) []string {
	if cs.maxUnits <= 0 {
		// no size constraint, treat entire text as single chunk
		return []string{text}
	}

	chunkSize := cs.calculateChunkSize(text)
	slog.Debug("Preparing text chunks", "countingMethod", cs.counter.Name(), "chunkSize", chunkSize, "textLength", len(text))
	// use iterative strategy-based chunking
	return chunk.SplitText(text, chunkSize)
}

// calculateChunkSize determines appropriate chunk size based on counting method and text length
func (cs *ChunkSelector) calculateChunkSize(text string) int {
	textLen := len(text)
	var baseSize, threshold int

	switch cs.counter.Name() {
	case "tokens (cl100k_base)":
		baseSize = cs.config.BaseTokenSize
		threshold = cs.config.TokenTextThreshold
	case "words":
		baseSize = cs.config.BaseWordSize
		threshold = cs.config.WordTextThreshold
	case "characters":
		baseSize = cs.config.BaseCharSize
		threshold = cs.config.CharTextThreshold
	default:
		// fallback to character-based chunking
		baseSize = cs.config.BaseCharSize
		threshold = cs.config.CharTextThreshold
	}

	// use larger chunks for very long text
	if textLen > threshold {
		return int(float64(baseSize) * cs.config.LargeTextMultiplier)
	}

	return baseSize
}

// ApplySizeConstraints applies size constraints to chunks using the configured strategy.
func (cs *ChunkSelector) ApplySizeConstraints(chunks []string) (string, error) {
	// use the unified pathway: PrepareForStrategy + Select
	// all edge case handling (empty chunks, no limits) is now handled in Select
	orderedChunks := cs.PrepareForStrategy(chunks)

	// use configured default context values for non-search scenarios
	result, err := cs.Select(orderedChunks, chunks, cs.defaultContextBefore, cs.defaultContextAfter)
	if err != nil {
		return "", fmt.Errorf("failed to select chunks: %w", err)
	}

	return result, nil
}

// formatSelectedChunks formats chunks with overlap removal and proper separators
func (cs *ChunkSelector) formatSelectedChunks(selected []ChunkWithIndex) string {
	if len(selected) == 0 {
		return ""
	}

	// sort by document order
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Index < selected[j].Index
	})

	slog.Debug("Formatting selected chunks", "count", len(selected))

	var result strings.Builder
	for i, chunk := range selected {
		chunkText := chunk.Text

		// remove overlapping prefix from subsequent chunks
		if i > 0 {
			chunkText = cs.removeOverlapPrefix(chunkText, selected[i-1].Text)
		}

		// add separtor between chunks
		if i > 0 && strings.TrimSpace(chunkText) != "" {
			result.WriteString("\n\n")
		}

		if strings.TrimSpace(chunkText) != "" {
			result.WriteString(chunkText)
		}
	}

	return result.String()
}

// removeOverlapPrefix removes overlapping text from the start of currentChunk
// that matches the end of previousChunk, using word-boundary detection
func (cs *ChunkSelector) removeOverlapPrefix(currentChunk, previousChunk string) string {
	currentWords := strings.Fields(currentChunk)
	previousWords := strings.Fields(previousChunk)

	if len(currentWords) == 0 || len(previousWords) == 0 {
		return currentChunk
	}

	// find longest sequence of words at end of previous chunk matching beginning of current chunk
	maxCheck := min(len(currentWords), len(previousWords))
	if maxCheck > 15 {
		maxCheck = 15 // reasonable limit to prevent excessive computation
	}

	for i := maxCheck; i > 0; i-- {
		if len(previousWords) < i || len(currentWords) < i {
			continue
		}

		prevSuffix := previousWords[len(previousWords)-i:]
		currPrefix := currentWords[:i]

		// compare word sequences
		if cs.slicesEqual(prevSuffix, currPrefix) {
			// found overlap, return remainder of current chunk
			if i < len(currentWords) {
				return strings.Join(currentWords[i:], " ")
			}
			return "" // entire chunk was overlap
		}
	}

	return currentChunk // no overlap detected
}

// slicesEqual compares two string slices for equality
func (cs *ChunkSelector) slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// getChunkWithConfigurableContext returns the target chunk and its configurable context neighbors
func (cs *ChunkSelector) getChunkWithConfigurableContext(targetIndex int, allChunks []string, contextBefore, contextAfter int, addedIndices map[int]bool) []ChunkWithIndex {
	var candidates []ChunkWithIndex

	// add preceding chunks (contextBefore)
	for i := targetIndex - contextBefore; i < targetIndex; i++ {
		if i >= 0 && !addedIndices[i] {
			candidates = append(candidates, ChunkWithIndex{Text: allChunks[i], Index: i})
		}
	}

	// add target chunk
	if !addedIndices[targetIndex] {
		candidates = append(candidates, ChunkWithIndex{Text: allChunks[targetIndex], Index: targetIndex})
	}

	// add following chunks (contextAfter)
	for i := targetIndex + 1; i <= targetIndex+contextAfter; i++ {
		if i < len(allChunks) && !addedIndices[i] {
			candidates = append(candidates, ChunkWithIndex{Text: allChunks[i], Index: i})
		}
	}

	return candidates
}

// allowPartialChunks determines if partial chunks should be created for exact size limits
func (cs *ChunkSelector) allowPartialChunks() bool {
	// allow partial chunks for more precise size control
	return true
}

// createPartialChunk creates a partial chunk up to the specified unit limit
func (cs *ChunkSelector) createPartialChunk(chunkText string, remainingUnits int) string {
	if remainingUnits <= 0 {
		return ""
	}

	// for word counting, we can do precise partial chunks
	if cs.counter.Name() == "words" {
		words := strings.Fields(chunkText)
		if remainingUnits > 0 && len(words) > 0 {
			maxWords := min(remainingUnits, len(words))
			return strings.Join(words[:maxWords], " ")
		}
		return ""
	}

	// for tokens/characters, use approximation by taking a percentage of the chunk
	chunkUnits := cs.counter.Count(chunkText)
	if chunkUnits > 0 && remainingUnits > 0 {
		ratio := float64(remainingUnits) / float64(chunkUnits)
		if ratio > 0 && ratio < 1 {
			cutoff := int(float64(len(chunkText)) * ratio)
			if cutoff > 0 && cutoff < len(chunkText) {
				return chunkText[:cutoff]
			}
		}
	}

	return ""
}

// PrepareForSearch converts scored search results into the unified ChunkWithIndex format.
// Chunks are already ordered by relevance score (highest first).
func (cs *ChunkSelector) PrepareForSearch(scoredChunks []ChunkScore) []ChunkWithIndex {
	if len(scoredChunks) == 0 {
		return []ChunkWithIndex{}
	}

	orderedChunks := make([]ChunkWithIndex, len(scoredChunks))
	for i, scored := range scoredChunks {
		orderedChunks[i] = ChunkWithIndex{
			Text:  scored.Chunk,
			Index: scored.Index,
		}
	}

	slog.Debug("Prepared chunks for search", "totalChunks", len(orderedChunks))
	return orderedChunks
}

// PrepareForStrategy converts plain text chunks into the unified ChunkWithIndex format,
// ordered according to the specified sizing strategy.
func (cs *ChunkSelector) PrepareForStrategy(chunks []string) []ChunkWithIndex {
	if len(chunks) == 0 {
		return []ChunkWithIndex{}
	}

	// create ChunkWithIndex entries with original indices
	chunksWithIndex := make([]ChunkWithIndex, len(chunks))
	for i, chunk := range chunks {
		chunksWithIndex[i] = ChunkWithIndex{
			Text:  chunk,
			Index: i,
		}
	}

	// apply strategy approach/ordering
	orderedChunks := cs.applyStrategyOrderToChunksWithIndex(chunksWithIndex)

	slog.Debug("Prepared chunks for strategy", "strategy", cs.strategy, "totalChunks", len(orderedChunks))
	return orderedChunks
}

// applyStrategyOrderToChunksWithIndex reorders ChunkWithIndex based on sizing strategy
func (cs *ChunkSelector) applyStrategyOrderToChunksWithIndex(chunks []ChunkWithIndex) []ChunkWithIndex {
	switch cs.strategy {
	case Beginning:
		// return chunks in original order (beginning to end)
		return chunks

	case End:
		// return chunks in reverse order (end to beginning)
		reversed := make([]ChunkWithIndex, len(chunks))
		for i, chunk := range chunks {
			reversed[len(chunks)-1-i] = chunk
		}
		return reversed

	case Middle:
		// return chunks starting from middle and expanding outward
		return cs.applyMiddleOutStrategyToChunksWithIndex(chunks)

	default:
		// default to beginning
		return chunks
	}
}

// applyMiddleOutStrategyToChunksWithIndex implements middle-out chunk selection for ChunkWithIndex
func (cs *ChunkSelector) applyMiddleOutStrategyToChunksWithIndex(chunks []ChunkWithIndex) []ChunkWithIndex {
	if len(chunks) == 0 {
		return chunks
	}

	if len(chunks) == 1 {
		return chunks
	}

	// start from the middle index
	middle := len(chunks) / 2
	result := []ChunkWithIndex{chunks[middle]}

	// expand outward alternating between right and left
	left := middle - 1
	right := middle + 1

	for len(result) < len(chunks) {
		if right < len(chunks) {
			result = append(result, chunks[right])
			right++
		}
		if left >= 0 {
			result = append(result, chunks[left])
			left--
		}
	}

	return result
}

// Select is a unified chunk selection method that processes pre-ordered chunks and accumulates
// them with configurable context until size limits are reached.
//
// This method serves as the primary entry point for chunk selection, handling both search-based
// and strategy-based scenarios with consistent behavior and formatting.
//
// Parameters:
//   - orderedChunks: Pre-ordered chunks (by relevance for search, by strategy for non-search)
//   - allChunks: Complete slice of all available chunks for context expansion
//   - contextBefore: Number of chunks to include before each selected chunk
//   - contextAfter: Number of chunks to include after each selected chunk
//
// Behavior:
//   - Accumulates chunks with context until size limit is reached
//   - Supports partial chunks when allowPartialChunks() returns true
//   - Prevents duplicate chunks using index tracking
//   - Maintains original document order in final output
//
// Returns:
//   - Formatted string with selected chunks, overlap removed, and proper separators
//   - Error if chunk selection fails
//
// Example usage:
//
//	// Search-based scenario with context
//	result, err := cs.Select(searchResults, allChunks, 1, 1)
//
//	// Strategy-based scenario w/ no context
//	result, err := cs.Select(strategyChunks, allChunks, 0, 0)
func (cs *ChunkSelector) Select(orderedChunks []ChunkWithIndex, allChunks []string, contextBefore, contextAfter int) (string, error) {
	// handle empty input
	if len(orderedChunks) == 0 {
		return "", nil
	}

	slog.Debug("Starting unified chunk selection", "orderedChunks", len(orderedChunks), "maxUnits", cs.maxUnits, "contextBefore", contextBefore, "contextAfter", contextAfter)

	// handle no size limit case (maxUnits <= 0) - return all chunks with consistent formatting
	if cs.maxUnits <= 0 {
		slog.Debug("No size limit specified, selecting all chunks")

		// for no-limit scenarios, add all chunks with context to ensure comprehensive coverage
		var selectedChunks []ChunkWithIndex
		addedIndices := make(map[int]bool)

		for _, orderedChunk := range orderedChunks {
			chunkCandidates := cs.getChunkWithConfigurableContext(orderedChunk.Index, allChunks, contextBefore, contextAfter, addedIndices)

			for _, candidate := range chunkCandidates {
				if !addedIndices[candidate.Index] {
					selectedChunks = append(selectedChunks, candidate)
					addedIndices[candidate.Index] = true
					slog.Debug("Added chunk (no limit)", "index", candidate.Index)
				}
			}
		}

		return cs.formatSelectedChunks(selectedChunks), nil
	}

	// handle normal size-constrained selection
	var selectedChunks []ChunkWithIndex
	var currentUnits int
	addedIndices := make(map[int]bool)

	// unified accumulation loop: iterate through ordered chunks, add w/ context until size limit reached
	for _, orderedChunk := range orderedChunks {
		if currentUnits >= cs.maxUnits {
			break
		}

		// get chunk with context neighbors
		chunkCandidates := cs.getChunkWithConfigurableContext(orderedChunk.Index, allChunks, contextBefore, contextAfter, addedIndices)

		for _, candidate := range chunkCandidates {
			chunkUnits := cs.counter.Count(candidate.Text)
			if currentUnits+chunkUnits <= cs.maxUnits {
				selectedChunks = append(selectedChunks, candidate)
				addedIndices[candidate.Index] = true
				currentUnits += chunkUnits
				slog.Debug("Added chunk with context", "index", candidate.Index, "units", chunkUnits, "totalUnits", currentUnits)
			} else if cs.allowPartialChunks() && currentUnits < cs.maxUnits {
				// add partial chunk to reach exact limit
				remainingUnits := cs.maxUnits - currentUnits
				partial := cs.createPartialChunk(candidate.Text, remainingUnits)
				if partial != "" {
					selectedChunks = append(selectedChunks, ChunkWithIndex{Text: partial, Index: candidate.Index})
					currentUnits = cs.maxUnits
					slog.Debug("Added partial chunk", "index", candidate.Index, "units", remainingUnits, "totalUnits", currentUnits)
				}
				break
			}
		}

		if currentUnits >= cs.maxUnits {
			break
		}
	}

	slog.Debug("Unified chunk selection complete", "selectedChunks", len(selectedChunks), "finalUnits", currentUnits)
	return cs.formatSelectedChunks(selectedChunks), nil
}
