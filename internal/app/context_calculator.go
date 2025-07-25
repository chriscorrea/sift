// Package app contains context calculation logic for smart search result presentation
package app

import (
	"log/slog"
	"strings"

	"github.com/chriscorrea/bm25md"
	"github.com/chriscorrea/sift/internal/counter"
)

// ContextStrategy defines how context should be distributed around a search result
type ContextStrategy struct {
	BeforeRatio float64 // context budget allocated before target chunk
	AfterRatio  float64 // context budget allocated after target chunk
	Name        string  // descriptive name (for debugging)
}

// ContextCalculator handles intelligent context selection for search results
// based on token/character budgets and markdown field types
type ContextCalculator struct {
	counter         counter.Counter
	maxContextUnits int
	patterns        *regexPatterns // shared regex patterns
}

// NewContextCalculator creates a new context calculator with the specified budget
func NewContextCalculator(textCounter counter.Counter, maxContextUnits int) (*ContextCalculator, error) {
	return &ContextCalculator{
		counter:         textCounter,
		maxContextUnits: maxContextUnits,
		patterns:        getRegexPatterns(),
	}, nil
}

// ContextRequest encapsulates a request for context calculation
type ContextRequest struct {
	TargetChunk ChunkWithIndex
	AllChunks   []string
}

// ContextResult contains the calculated context chunks and metadata
type ContextResult struct {
	SelectedChunks []ChunkWithIndex
	TotalUnits     int
	Strategy       ContextStrategy
	FieldType      ChunkFieldType
}

// CalculateSmartContext determines the best context chunks for a search result
// based on the target chunk field type and available token/character budget
func (cc *ContextCalculator) CalculateSmartContext(req ContextRequest) ContextResult {
	return cc.CalculateSmartContextWithBudget(req, cc.maxContextUnits)
}

// CalculateSmartContextWithBudget determines the best context chunks for a search result
// with a specific budget limit, allowing for dynamic budget management without creating
// new calculator instances
func (cc *ContextCalculator) CalculateSmartContextWithBudget(req ContextRequest, budgetUnits int) ContextResult {
	// detect the primary field type of the target chunk
	fieldType := cc.detectPrimaryFieldType(req.TargetChunk.Text)

	// get the appropriate context strategy for this field type
	strategy := cc.getContextStrategy(fieldType)

	// calculate target chunk size
	targetUnits := cc.counter.Count(req.TargetChunk.Text)

	// calculate available context budget
	availableContextUnits := budgetUnits - targetUnits
	if availableContextUnits <= 0 {
		// target chunk exceeds budget, truncate it to fit
		if targetUnits > budgetUnits {
			truncatedText := cc.createPartialChunk(req.TargetChunk.Text, budgetUnits)
			truncatedChunk := ChunkWithIndex{
				Text:  truncatedText,
				Index: req.TargetChunk.Index,
				Score: req.TargetChunk.Score,
			}
			truncatedUnits := cc.counter.Count(truncatedText)
			return ContextResult{
				SelectedChunks: []ChunkWithIndex{truncatedChunk},
				TotalUnits:     truncatedUnits,
				Strategy:       strategy,
				FieldType:      fieldType,
			}
		}
		// target chunk exactly fits budget, no room for context
		return ContextResult{
			SelectedChunks: []ChunkWithIndex{req.TargetChunk},
			TotalUnits:     targetUnits,
			Strategy:       strategy,
			FieldType:      fieldType,
		}
	}

	// distribute context budget according to strategy
	beforeBudget := int(float64(availableContextUnits) * strategy.BeforeRatio)
	afterBudget := availableContextUnits - beforeBudget

	// collect context chunks
	var selectedChunks []ChunkWithIndex
	totalUnits := targetUnits

	// add target chunk
	selectedChunks = append(selectedChunks, req.TargetChunk)

	// add preceding context
	var beforeUnits, afterUnits int
	if beforeBudget > 0 {
		beforeChunks, units := cc.collectContextChunks(req.AllChunks, req.TargetChunk.Index-1, -1, beforeBudget)
		selectedChunks = append(selectedChunks, beforeChunks...)
		totalUnits += units
		beforeUnits = units

		slog.Debug("Collected preceding context",
			"chunks", len(beforeChunks),
			"units", units,
			"budget", beforeBudget,
			"indices", cc.getChunkIndices(beforeChunks))
	}

	// add following context
	if afterBudget > 0 {
		afterChunks, units := cc.collectContextChunks(req.AllChunks, req.TargetChunk.Index+1, 1, afterBudget)
		selectedChunks = append(selectedChunks, afterChunks...)
		totalUnits += units
		afterUnits = units

		slog.Debug("Collected following context",
			"chunks", len(afterChunks),
			"units", units,
			"budget", afterBudget,
			"indices", cc.getChunkIndices(afterChunks))
	}

	slog.Debug("Context distribution",
		"strategy", strategy.Name,
		"targetUnits", targetUnits,
		"beforeUnits", beforeUnits,
		"afterUnits", afterUnits,
		"totalUnits", totalUnits)

	return ContextResult{
		SelectedChunks: selectedChunks,
		TotalUnits:     totalUnits,
		Strategy:       strategy,
		FieldType:      fieldType,
	}
}

// ChunkFieldType represents the detected field type of a chunk, including special cases
type ChunkFieldType struct {
	Primary bm25md.Field
	IsList  bool // special flag for list items that need different context strategy
}

// detectPrimaryFieldType analyzes chunk content to determine its primary markdown field type
func (cc *ContextCalculator) detectPrimaryFieldType(chunk string) ChunkFieldType {
	trimmed := strings.TrimSpace(chunk)
	if trimmed == "" {
		return ChunkFieldType{Primary: bm25md.FieldBody, IsList: false}
	}

	// check for headers (most specific first)
	if cc.patterns.headerRegex.MatchString(trimmed) {
		// count #'s to determine header level
		headerLevel := 0
		for _, r := range trimmed {
			if r == '#' {
				headerLevel++
			} else {
				break
			}
		}

		var fieldType bm25md.Field
		switch headerLevel {
		case 1:
			fieldType = bm25md.FieldH1
		case 2:
			fieldType = bm25md.FieldH2
		case 3:
			fieldType = bm25md.FieldH3
		case 4:
			fieldType = bm25md.FieldH4
		case 5:
			fieldType = bm25md.FieldH5
		case 6:
			fieldType = bm25md.FieldH6
		default:
			fieldType = bm25md.FieldH4 // fallback for excessive #'s
		}

		return ChunkFieldType{Primary: fieldType, IsList: false}
	}

	// check for bullet lists
	if cc.patterns.bulletListRegex.MatchString(trimmed) {
		return ChunkFieldType{Primary: bm25md.FieldBody, IsList: true}
	}

	// check for numbered lists
	if cc.patterns.numberListRegex.MatchString(trimmed) {
		return ChunkFieldType{Primary: bm25md.FieldBody, IsList: true}
	}

	// check for code blocks
	if cc.patterns.codeBlockRegex.MatchString(chunk) {
		return ChunkFieldType{Primary: bm25md.FieldCode, IsList: false}
	}

	// check for inline code (before checking bold/italic)
	if cc.patterns.inlineCodeRegex.MatchString(chunk) {
		return ChunkFieldType{Primary: bm25md.FieldCode, IsList: false}
	}

	// check for bold formatting (more specific pattern)
	if cc.patterns.boldRegex.MatchString(chunk) {
		return ChunkFieldType{Primary: bm25md.FieldBold, IsList: false}
	}

	// check for italic formatting (avoid false positives)
	if cc.patterns.italicRegex.MatchString(chunk) {
		return ChunkFieldType{Primary: bm25md.FieldItalic, IsList: false}
	}

	// default to body text
	return ChunkFieldType{Primary: bm25md.FieldBody, IsList: false}
}

// getContextStrategy returns the appropriate context distribution strategy for a field type
func (cc *ContextCalculator) getContextStrategy(fieldType ChunkFieldType) ContextStrategy {
	// special handling for lists
	if fieldType.IsList {
		// lists often conclude or elaborate on preceding content
		return ContextStrategy{
			BeforeRatio: 0.8,
			AfterRatio:  0.2,
			Name:        "list-preceding",
		}
	}

	// handle primary field types
	switch fieldType.Primary {
	case bm25md.FieldH1, bm25md.FieldH2, bm25md.FieldH3, bm25md.FieldH4, bm25md.FieldH5, bm25md.FieldH6:
		// headers to focus on context that follows
		return ContextStrategy{
			BeforeRatio: 0.2,
			AfterRatio:  0.8,
			Name:        "header-following",
		}

	case bm25md.FieldCode:
		// code blocks may require more context that follows?
		return ContextStrategy{
			BeforeRatio: 0.3,
			AfterRatio:  0.7,
			Name:        "code-following",
		}

	case bm25md.FieldBold:
		// emphasized content may conclude a thought, so prefer context preceding
		return ContextStrategy{
			BeforeRatio: 0.65,
			AfterRatio:  0.35,
			Name:        "emphasis-preceding",
		}

	default: // FieldBody, FieldItalic, etc.
		// default balanced context for paragraphs, lists, etc.
		return ContextStrategy{
			BeforeRatio: 0.5,
			AfterRatio:  0.5,
			Name:        "balanced",
		}
	}
}

// collectContextChunks gathers context chunks in a specific direction until budget is exhausted
func (cc *ContextCalculator) collectContextChunks(allChunks []string, startIndex, direction, budget int) ([]ChunkWithIndex, int) {
	var contextChunks []ChunkWithIndex
	totalUnits := 0

	for i := startIndex; i >= 0 && i < len(allChunks); i += direction {
		if budget <= 0 {
			break
		}

		chunk := allChunks[i]
		chunkUnits := cc.counter.Count(chunk)

		if totalUnits+chunkUnits <= budget {
			contextChunks = append(contextChunks, ChunkWithIndex{
				Text:  chunk,
				Index: i,
				Score: 0, // context chunks don't have scores
			})
			totalUnits += chunkUnits
			budget -= chunkUnits
		} else if budget > 0 {
			// chunk would exceed budget, try to include partial chunk
			partial := cc.createPartialChunk(chunk, budget)
			if partial != "" {
				contextChunks = append(contextChunks, ChunkWithIndex{
					Text:  partial,
					Index: i,
					Score: 0,
				})
				totalUnits += cc.counter.Count(partial)
			}
			break
		}
	}

	// reverse preceding chunks for natural reading order
	if direction < 0 {
		for i, j := 0, len(contextChunks)-1; i < j; i, j = i+1, j-1 {
			contextChunks[i], contextChunks[j] = contextChunks[j], contextChunks[i]
		}
	}

	return contextChunks, totalUnits
}

// createPartialChunk creates a partial chunk up to the specified unit limit
func (cc *ContextCalculator) createPartialChunk(chunkText string, remainingUnits int) string {
	if remainingUnits <= 0 {
		return ""
	}

	// for word counting, we can do precise partial chunks
	if cc.counter.Name() == "words" {
		words := strings.Fields(chunkText)
		if remainingUnits > 0 && len(words) > 0 {
			maxWords := min(remainingUnits, len(words))
			return strings.Join(words[:maxWords], " ")
		}
		return ""
	}

	// for token counting, use precise tokenization
	if cc.counter.Name() == "tokens (cl100k_base)" {
		// type assert to TokenCounter to access CreatePartialText method
		if tokenCounter, ok := cc.counter.(*counter.TokenCounter); ok {
			return tokenCounter.CreatePartialText(chunkText, remainingUnits)
		}
		// fallback to old approximation method if type assertion fails
		slog.Debug("Failed to cast to TokenCounter, using approximation fallback")
	}

	// for characters, use precise character-based truncation
	if cc.counter.Name() == "characters" {
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
	chunkUnits := cc.counter.Count(chunkText)
	if chunkUnits > 0 && remainingUnits > 0 {
		ratio := float64(remainingUnits) / float64(chunkUnits)
		if ratio > 0 && ratio < 1 {
			cutoff := int(float64(len(chunkText)) * ratio)
			if cutoff > 0 && cutoff < len(chunkText) {
				// find word boundary
				for cutoff > 0 && chunkText[cutoff-1] != ' ' {
					cutoff--
				}
				if cutoff > 0 {
					return strings.TrimSpace(chunkText[:cutoff])
				}
			}
		}
	}

	return ""
}

// getChunkIndices extracts the indices from a slice of chunks for debug logging
func (cc *ContextCalculator) getChunkIndices(chunks []ChunkWithIndex) []int {
	indices := make([]int, len(chunks))
	for i, chunk := range chunks {
		indices[i] = chunk.Index
	}
	return indices
}
