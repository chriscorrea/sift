// This consolidates chunk selection, sizing strategies, and output formatting
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
		BaseTokenSize:       200, // ~200 tokens per chunk (substantial paragraph/stanza)
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
	Score float64 // BM25md relevance score (0 for non-search scenarios)
}

// ChunkSelector handles chunk selection and sizing using configurable strategies
type ChunkSelector struct {
	counter              counter.Counter
	maxUnits             int
	strategy             SizingStrategy
	config               ChunkingConfig
	defaultContextBefore int                // default context before chunks for non-search scenarios
	defaultContextAfter  int                // default context after chunks for non-search scenarios
	isSearchMode         bool               // true when processing search results, enables gap detection
	contextCalculator    *ContextCalculator // cached context calculator for smart context
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
	// always chunk content for better searchability
	// even when there's no size limit, chunking enables effective search
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

		// add smart separator between chunks
		if i > 0 && strings.TrimSpace(chunkText) != "" {
			// in search mode, add gap indicator for non-consecutive chunks
			if cs.isSearchMode && selected[i].Index != selected[i-1].Index+1 {
				result.WriteString("\n\n---\n\n")
			} else {
				separator := cs.determineSeparator(selected[i-1].Text, chunkText)
				result.WriteString(separator)
			}
		}

		if strings.TrimSpace(chunkText) != "" {
			result.WriteString(chunkText)
		}
	}

	return result.String()
}

// determineSeparator returns appropriate separator between chunks based on content analysis.
// It preserves original formatting by detecting paragraph vs line boundaries.
func (cs *ChunkSelector) determineSeparator(prevChunk, currentChunk string) string {
	if prevChunk == "" {
		return "" // first chunk, no separator needed
	}

	// Original behavior: check if previous chunk ends with natural paragraph indicators
	prevTrimmed := strings.TrimSpace(prevChunk)
	if prevTrimmed == "" {
		return "\n\n" // default to paragraph break for empty chunks
	}

	// check if previous chunk ends with explicit line breaks from original text
	if strings.HasSuffix(prevChunk, "\n\n") {
		return "\n\n" // preserve explicit paragraph breaks
	}
	if strings.HasSuffix(prevChunk, "\n") {
		return "\n" // preserve single line breaks
	}

	// check if previous chunk ends with substantial sentence
	if (strings.HasSuffix(prevTrimmed, ".") || strings.HasSuffix(prevTrimmed, "!") ||
		strings.HasSuffix(prevTrimmed, "?")) && len(prevTrimmed) > 40 {
		return "\n\n" // substantial sentence gets paragraph break
	}

	// default to single line break for general text flow
	return "\n"
}

// removeOverlapPrefix removes overlapping text from the start of currentChunk
// that matches the end of previousChunk, using word-boundary detection
func (cs *ChunkSelector) removeOverlapPrefix(currentChunk, previousChunk string) string {
	currentWords := strings.Fields(currentChunk)
	previousWords := strings.Fields(previousChunk)

	if len(currentWords) == 0 || len(previousWords) == 0 {
		return currentChunk
	}

	// find longest seq of words at end of previous chunk matching beginning of current chunk
	maxCheck := min(len(currentWords), len(previousWords))
	if maxCheck > 15 {
		maxCheck = 15 // reasonable limit
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
			candidates = append(candidates, ChunkWithIndex{Text: allChunks[i], Index: i, Score: 0})
		}
	}

	// add target chunk
	if !addedIndices[targetIndex] {
		candidates = append(candidates, ChunkWithIndex{Text: allChunks[targetIndex], Index: targetIndex, Score: 0})
	}

	// add following chunks (contextAfter)
	for i := targetIndex + 1; i <= targetIndex+contextAfter; i++ {
		if i < len(allChunks) && !addedIndices[i] {
			candidates = append(candidates, ChunkWithIndex{Text: allChunks[i], Index: i, Score: 0})
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

	// for token counting, use precise tokenization
	if cs.counter.Name() == "tokens (cl100k_base)" {
		// type assert to TokenCounter to access CreatePartialText method
		if tokenCounter, ok := cs.counter.(*counter.TokenCounter); ok {
			return tokenCounter.CreatePartialText(chunkText, remainingUnits)
		}
		// fallback to old approximation method if type assertion fails
		slog.Debug("Failed to cast to TokenCounter, using approximation fallback")
	}

	// for characters, use precise character-based truncation
	if cs.counter.Name() == "characters" {
		if len(chunkText) <= remainingUnits {
			return chunkText
		}
		// find word boundary to avoid cutting mid-word
		cutoff := remainingUnits
		for cutoff > 0 && cutoff < len(chunkText) && chunkText[cutoff-1] != ' ' {
			cutoff--
		}
		if cutoff > 0 {
			return strings.TrimSpace(chunkText[:cutoff])
		}
		// if no word boundary found, cut at exact character limit
		return chunkText[:remainingUnits]
	}

	// fallback for unknown counting methods - use old approximation
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

	// enable search mode for gap detection
	cs.isSearchMode = true

	orderedChunks := make([]ChunkWithIndex, len(scoredChunks))
	for i, scored := range scoredChunks {
		orderedChunks[i] = ChunkWithIndex{
			Text:  scored.Chunk,
			Index: scored.Index,
			Score: scored.Score,
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
			Score: 0, // no score for non-search scenarios
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
	return cs.SelectWithContextConfig(orderedChunks, allChunks, contextBefore, contextAfter, 0, false)
}

// SelectWithContextConfig provides chunk selection with optional smart context calculation
func (cs *ChunkSelector) SelectWithContextConfig(orderedChunks []ChunkWithIndex, allChunks []string, contextBefore, contextAfter, contextUnits int, useSmartContext bool) (string, error) {
	// handle empty input
	if len(orderedChunks) == 0 {
		return "", nil
	}

	slog.Debug("Starting unified chunk selection", "orderedChunks", len(orderedChunks), "maxUnits", cs.maxUnits, "contextBefore", contextBefore, "contextAfter", contextAfter, "useSmartContext", useSmartContext)

	// if smart context is enabled and we're in search mode, use the context calculator
	if useSmartContext && contextUnits > 0 && cs.isSearchMode {
		return cs.selectWithSmartContext(orderedChunks, allChunks, contextUnits)
	}

	// fallback to original context selection logic
	return cs.selectWithFixedContext(orderedChunks, allChunks, contextBefore, contextAfter)
}

// selectWithSmartContext uses the ContextCalculator for intelligent context selection
func (cs *ChunkSelector) selectWithSmartContext(orderedChunks []ChunkWithIndex, allChunks []string, contextUnits int) (string, error) {
	// lazily create and cache the context calculator
	if cs.contextCalculator == nil || cs.contextCalculator.maxContextUnits != contextUnits {
		calculator, err := NewContextCalculator(cs.counter, contextUnits)
		if err != nil {
			return "", fmt.Errorf("failed to create context calculator: %w", err)
		}
		cs.contextCalculator = calculator
	}

	var allSelectedChunks []ChunkWithIndex
	addedIndices := make(map[int]bool)

	currentUnits := 0

	for _, orderedChunk := range orderedChunks {
		if addedIndices[orderedChunk.Index] {
			continue // skip already processed chunks
		}

		// calculate remaining budget for this search result
		remainingBudget := contextUnits - currentUnits
		if remainingBudget <= 0 {
			break // no budget left
		}

		// use smart context calculation for this chunk with remaining budget
		contextResult := cs.contextCalculator.CalculateSmartContextWithBudget(ContextRequest{
			TargetChunk: orderedChunk,
			AllChunks:   allChunks,
		}, remainingBudget)

		slog.Debug("Smart context calculated", "targetIndex", orderedChunk.Index, "strategy", contextResult.Strategy.Name, "fieldType", contextResult.FieldType.Primary, "totalUnits", contextResult.TotalUnits, "selectedChunks", len(contextResult.SelectedChunks), "remainingBudget", remainingBudget)

		// add all selected chunks from this context result
		for _, chunk := range contextResult.SelectedChunks {
			if !addedIndices[chunk.Index] {
				chunkUnits := cs.counter.Count(chunk.Text)
				if currentUnits+chunkUnits <= contextUnits {
					allSelectedChunks = append(allSelectedChunks, chunk)
					addedIndices[chunk.Index] = true
					currentUnits += chunkUnits
				} else {
					// try to add a partial chunk if possible
					remainingUnits := contextUnits - currentUnits
					if remainingUnits > 0 {
						partial := cs.createPartialChunk(chunk.Text, remainingUnits)
						if partial != "" {
							partialChunk := ChunkWithIndex{Text: partial, Index: chunk.Index}
							allSelectedChunks = append(allSelectedChunks, partialChunk)
							currentUnits = contextUnits // budget is now fully used
						}
					}
					break // budget exhausted
				}
			}
		}

		if currentUnits >= contextUnits {
			break // budget exhausted
		}
	}

	return cs.formatSelectedChunks(allSelectedChunks), nil
}

// selectWithFixedContext uses the original fixed-count context selection logic
func (cs *ChunkSelector) selectWithFixedContext(orderedChunks []ChunkWithIndex, allChunks []string, contextBefore, contextAfter int) (string, error) {
	slog.Debug("Using fixed context selection", "contextBefore", contextBefore, "contextAfter", contextAfter)

	// handle no size limit case (maxUnits <= 0)
	if cs.maxUnits <= 0 {
		if cs.isSearchMode {
			slog.Debug("No size limit specified in search mode, selecting only relevant chunks")

			// minimum score threshold to filter out low-relevance chunks
			const minScoreThreshold = 0.01 // reasonable threshold to filter noise

			// step 1: filter by minimum score threshold
			var scoreFilteredChunks []ChunkWithIndex
			for _, chunk := range orderedChunks {
				if chunk.Score > minScoreThreshold { // use > instead of >= to exclude true zeros
					scoreFilteredChunks = append(scoreFilteredChunks, chunk)
				}
			}

			// step 2: limit to top 50% of remaining chunks or first 5 chunks (whichever smaller)
			maxRelevantChunks := len(scoreFilteredChunks) / 2
			if maxRelevantChunks == 0 && len(scoreFilteredChunks) > 0 {
				maxRelevantChunks = 1 // at least take the top chunk if any passed threshold
			}
			if maxRelevantChunks > 5 {
				maxRelevantChunks = 5 // hard-coded default limit
			}

			relevantChunks := scoreFilteredChunks
			if len(scoreFilteredChunks) > maxRelevantChunks {
				relevantChunks = scoreFilteredChunks[:maxRelevantChunks]
			}

			slog.Debug("Search filtering applied", "originalChunks", len(orderedChunks), "afterScoreFilter", len(scoreFilteredChunks), "finalRelevant", len(relevantChunks))

			// fallback: if no chunks passed threshold, take top 2 chunks anyway
			if len(relevantChunks) == 0 && len(orderedChunks) > 0 {
				maxFallback := 2
				if len(orderedChunks) < maxFallback {
					maxFallback = len(orderedChunks)
				}
				relevantChunks = orderedChunks[:maxFallback]
				slog.Debug("Applied fallback selection", "fallbackChunks", len(relevantChunks))
			}

			var selectedChunks []ChunkWithIndex
			addedIndices := make(map[int]bool)

			for _, orderedChunk := range relevantChunks {
				chunkCandidates := cs.getChunkWithConfigurableContext(orderedChunk.Index, allChunks, contextBefore, contextAfter, addedIndices)

				for _, candidate := range chunkCandidates {
					if !addedIndices[candidate.Index] {
						selectedChunks = append(selectedChunks, candidate)
						addedIndices[candidate.Index] = true
						slog.Debug("Added relevant chunk (search no limit)", "index", candidate.Index)
					}
				}
			}

			return cs.formatSelectedChunks(selectedChunks), nil
		} else {
			slog.Debug("No size limit specified, selecting all chunks")

			// for non-search no-limit scenarios, add all chunks with context for comprehensive coverage
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

	slog.Debug("Fixed context selection complete", "selectedChunks", len(selectedChunks), "finalUnits", currentUnits)
	return cs.formatSelectedChunks(selectedChunks), nil
}

// SetSearchMode enables or disables search mode for gap detection
func (cs *ChunkSelector) SetSearchMode(enabled bool) {
	cs.isSearchMode = enabled
}
