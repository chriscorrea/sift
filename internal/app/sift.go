// Package app contains the core application logic for the sift CLI tool.
// It handles the main business logic separated from CLI concerns.
package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/chriscorrea/sift/internal/classify"
	"github.com/chriscorrea/sift/internal/counter"
	"github.com/chriscorrea/sift/internal/extract"
	"github.com/chriscorrea/sift/internal/fetch"
	"github.com/chriscorrea/sift/internal/spinner"
	"github.com/chriscorrea/sift/internal/tfidf"
)

// OutputFormat defines the output format for results
type OutputFormat int

const (
	// markdown output format (default)
	Markdown OutputFormat = iota
	// plaintext output format
	Text
	// JSON output format
	JSON
)

// String returns the string representation of the output
func (f OutputFormat) String() string {
	switch f {
	case Markdown:
		return "Markdown"
	case Text:
		return "Text"
	case JSON:
		return "JSON"
	default:
		return "Unknown"
	}
}

// ChunkScore represents a chunk with its TF-IDF score and original index
type ChunkScore struct {
	Chunk string  // text content of the chunk
	Score float64 // TF-IDF score (higher = more relevant)
	Index int     // original index in the document
}

// Config holds all configuration options for the sift application.
type Config struct {
	Sources        []string               // URLs, file paths, or "-" for stdin
	Selector       string                 // CSS selector for content extraction
	MaxUnits       int                    // max output units (tokens/words/characters)
	CountingMethod counter.CountingMethod // method for counting text units
	SizingStrategy SizingStrategy
	SearchQuery    string
	OutputFormat   OutputFormat // output format (md/txt/json)
	ContextBefore  int          // chunks to include before targeted search result chunk (default: 1)
	ContextAfter   int          // chunks to include after targeted search result chunk (default: 2)
	Quiet          bool         // suppress info messages
	Debug          bool
	IncludeAll     bool // include all content without readability or classification filtering
}

// Run executes the main sift application logic with the given configuration.
//
// Processing Pipeline:
// 1. Extract and combine content from all sources (extractAndCombineContent)
// 2. Apply relevant transformations (applyContentTransformations)
//
// ctx allows for cancellation and timeout control of long-running operations.
func Run(ctx context.Context, cfg Config) (string, error) {
	if len(cfg.Sources) == 0 {
		return "", fmt.Errorf("no sources provided")
	}

	// step 1: extract and combine content from all sources
	combinedContent, err := extractAndCombineContent(ctx, cfg.Sources, cfg.Selector, cfg.IncludeAll, cfg.Quiet)
	if err != nil {
		return "", err
	}

	// step 2: apply relevant transformations
	if cfg.MaxUnits > 0 {
		return applyContentTransformations(ctx, combinedContent, cfg.CountingMethod, cfg.MaxUnits, cfg.SizingStrategy, cfg.IncludeAll, cfg.SearchQuery, cfg.Quiet, cfg.ContextBefore, cfg.ContextAfter)
	}

	return combinedContent, nil
}

// extractAndCombineContent processes all sources and combines their content with appropriate separators.
func extractAndCombineContent(ctx context.Context, sources []string, selector string, includeAll, quiet bool) (string, error) {
	var combinedContent strings.Builder

	for _, source := range sources {
		content, err := processSource(ctx, source, selector, includeAll, quiet)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Warning: failed to process source %q: %v\n", source, err)
			}
			continue
		}

		if combinedContent.Len() > 0 {
			combinedContent.WriteString("\n\n")
		}
		combinedContent.WriteString(content)
	}

	if combinedContent.Len() == 0 {
		return "", fmt.Errorf("no content extracted from any source")
	}

	return combinedContent.String(), nil
}

// processSource fetches content from a single source and converts it to markdown
// TODO: implement streaming; current approach loads full content into memory
func processSource(ctx context.Context, source, selector string, includeAll, quiet bool) (string, error) {
	// fetch content
	reader, err := fetch.GetContent(ctx, source)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content: %w", err)
	}
	defer reader.Close()

	// parse source URL for context (if it's a URL)
	var baseURL *url.URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		baseURL, _ = url.Parse(source) // ignore parse errors, will use nil
	}

	// extract and convert to Markdown
	markdown, err := extract.ToMarkdown(reader, selector, includeAll, baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	if strings.TrimSpace(markdown) == "" {
		return "", fmt.Errorf("no content extracted")
	}

	return markdown, nil
}

// applyContentTransformations coordinates the application of size constraints and transformations.
//
// Transformation Pipeline:
// 1. prepare chunks (chunking, filtering)
// 2. apply transformations (search, sizing)
//
// ctx allows for cancellation of search operations within size constraint application.
func applyContentTransformations(ctx context.Context, text string, countingMethod counter.CountingMethod, maxUnits int, sizingStrategy SizingStrategy, includeAll bool, searchQuery string, quiet bool, contextBefore, contextAfter int) (string, error) {
	// step 1: prepare chunks for processing
	selector, chunks, err := prepareChunksForProcessing(text, countingMethod, maxUnits, sizingStrategy, includeAll)
	if err != nil {
		return "", err
	}

	if len(chunks) == 0 {
		return "", nil
	}

	// step 2: apply transformations
	return applyTransformations(ctx, chunks, selector, searchQuery, quiet, contextBefore, contextAfter)
}

// prepareChunksForProcessing sets up the ChunkSelector and prepares filtered chunks ready for transformation
func prepareChunksForProcessing(text string, countingMethod counter.CountingMethod, maxUnits int, sizingStrategy SizingStrategy, includeAll bool) (*ChunkSelector, []string, error) {
	// create a ChunkSelector for unit-aware chunking
	selector, err := NewChunkSelector(countingMethod, maxUnits, sizingStrategy)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create chunk selector: %w", err)
	}

	// use unit-aware chunking
	chunks := selector.PrepareChunks(text)

	if len(chunks) == 0 {
		return selector, chunks, nil
	}

	// apply classification filtering *unless includeAll is true*
	if !includeAll && len(chunks) > 0 {
		classifier := classify.NewClassifier()
		filtered := make([]string, 0, len(chunks))

		for i, chunk := range chunks {
			if !classifier.IsExtraneous(chunk, i, len(chunks)) {
				filtered = append(filtered, chunk)
			}
		}

		chunks = filtered
	}

	return selector, chunks, nil
}

// applyTransformations handles chunk selection using a unified pathway for both search and normal sizing
func applyTransformations(ctx context.Context, chunks []string, selector *ChunkSelector, searchQuery string, quiet bool, contextBefore, contextAfter int) (string, error) {
	var orderedChunks []ChunkWithIndex
	var finalContextBefore, finalContextAfter int

	// determine chunk ordering and context based on whether search is configured
	if strings.TrimSpace(searchQuery) != "" {
		// search path: get scored chunks
		scoredChunks, err := performLexicalSearch(ctx, chunks, searchQuery, quiet)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Warning: search failed: %v\n", err)
			}
			// fall back to strategy-based selection
			orderedChunks = selector.PrepareForStrategy(chunks)
			finalContextBefore = selector.defaultContextBefore
			finalContextAfter = selector.defaultContextAfter
		} else {
			orderedChunks = selector.PrepareForSearch(scoredChunks)
			finalContextBefore = contextBefore
			finalContextAfter = contextAfter
		}
	} else {
		// strategy path
		orderedChunks = selector.PrepareForStrategy(chunks)
		finalContextBefore = selector.defaultContextBefore
		finalContextAfter = selector.defaultContextAfter
	}

	// single point of chunk selection (unified pathway)
	result, err := selector.Select(orderedChunks, chunks, finalContextBefore, finalContextAfter)
	if err != nil {
		return "", fmt.Errorf("failed to select chunks: %w", err)
	}

	return result, nil
}

// performLexicalSearch sorts chunks by relevance using TF-IDF keyword matching
// ctx allows for cancellation of search operations.
func performLexicalSearch(ctx context.Context, chunks []string, searchQuery string, quiet bool) ([]ChunkScore, error) {
	if len(chunks) == 0 {
		return []ChunkScore{}, nil
	}

	// display spinner for longer operations
	var sp *spinner.Spinner
	if !quiet {
		sp = spinner.New(ctx, os.Stderr, "Searching text...")
		sp.Start()
		defer sp.Stop()
	}

	// create TF-IDF corpus from chunks
	corpus := tfidf.NewCorpus(chunks)

	// score each chunk based on the search query
	var scoredChunks []ChunkScore
	for i, chunk := range chunks {
		score := corpus.Score(searchQuery, i)
		scoredChunks = append(scoredChunks, ChunkScore{
			Chunk: chunk,
			Score: score,
			Index: i,
		})
	}

	// sort by score (highest first) using Go sort algorithm
	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].Score > scoredChunks[j].Score
	})

	return scoredChunks, nil
}
