// Package chunk provides text chunking functionality for the sift CLI tool.
//
// This package implements an iterative, strategy-based text splitting system that breaks down large
// text documents into manageable chunks while preserving semantic boundaries.
//
// The chunking process uses a multi-wave approach with hierarchical splitting strategies:
//  1. Paragraph boundaries (double newlines) - preserves document structure
//  2. Sentence boundaries - keeps complete thoughts together
//  3. Line boundaries (single newlines) - maintains formatting context
//  4. Word boundaries - last resort for oversized content
//
// Usage Example:
//
//	chunks := chunk.SplitText(content, 250)
//	// Creates chunks of max 250 characters
//
// The package is designed to work with various text formats including Markdown,
// plain text, and structured documents while maintaining readability and context.
package chunk

import (
	"log/slog"
	"strings"
)

// splitStrategy defines a method for breaking up text.
type splitStrategy struct {
	name      string
	delimiter string
}

// strategies are ordered from largest semantic unit to smallest; each applied iteratively
// Note: this is package-level variable but is maintained for pragmatic simplicity.
// Note: Regex delimiters are not used here to maintain efficient implementation.
var strategies = []splitStrategy{
	{name: "paragraph", delimiter: "\n\n"},
	{name: "sentence", delimiter: ". "},
	{name: "sentence-question", delimiter: "? "},
	{name: "sentence-exclamation", delimiter: "! "},
	{name: "line", delimiter: "\n"},
	{name: "word", delimiter: " "},
}

// SplitText breaks text into manageable chunks using an iterative, strategy-based approach.
// It processes chunks in waves, applying each strategy to all oversized chunks from the previous wave.
//
// Parameters:
//   - text: the input text to split
//   - maxChunkSize: maximum size for each chunk in characters
//
// Returns a slice of text chunks, each respecting the maxChunkSize limit.
func SplitText(text string, maxChunkSize int) []string {
	slog.Debug("SplitText called", "textLength", len(text), "maxChunkSize", maxChunkSize)

	// validate input parameters
	if maxChunkSize <= 0 {
		slog.Debug("Invalid maxChunkSize", "maxChunkSize", maxChunkSize)
		return []string{}
	}

	// handle empty input - check for pure whitespace first
	if strings.TrimSpace(text) == "" {
		slog.Debug("Empty text after full whitespace trimming")
		return []string{}
	}
	
	// use gentle trimming to preserve intentional line breaks
	text = trimSpacesOnly(text)

	// if text fits in one chunk, return it
	if len(text) <= maxChunkSize {
		slog.Debug("Text fits in single chunk", "textLength", len(text))
		return []string{text}
	}

	var finalChunks []string
	chunksToProcess := []string{text} // start with the full text

	// iteratively apply each splitting strategy
	for _, strategy := range strategies {
		if len(chunksToProcess) == 0 {
			break // No more chunks to process
		}

		slog.Debug("Applying strategy", "strategy", strategy.name, "chunksToProcess", len(chunksToProcess))

		var nextQueue []string
		for _, chunk := range chunksToProcess {
			if len(chunk) <= maxChunkSize {
				// this chunk is good, add it to our final list
				finalChunks = append(finalChunks, chunk)
				continue
			}

			// this chunk is too big–split it with the current strategy
			slog.Debug("Splitting oversized chunk", "strategy", strategy.name, "chunkLength", len(chunk))
			subChunks := splitByDelimiter(chunk, strategy.delimiter, strategy.name, maxChunkSize)

			// add the newly split chunks to the queue for the next level of processing
			for _, sub := range subChunks {
				if subTrimmed := trimSpacesOnly(sub); subTrimmed != "" {
					nextQueue = append(nextQueue, subTrimmed)
				}
			}
		}

		// The next iteration will process the newly split chunks
		chunksToProcess = nextQueue
	}

	// Add any remaining chunks that were processed by the last strategy
	for _, chunk := range chunksToProcess {
		if trimmed := trimSpacesOnly(chunk); trimmed != "" {
			finalChunks = append(finalChunks, trimmed)
		}
	}

	slog.Debug("SplitText completed", "finalChunkCount", len(finalChunks))
	return finalChunks
}

// splitByDelimiter splits text by a delimiter and packs segments together up to the size limit.
func splitByDelimiter(text, delimiter, strategyName string, maxChunkSize int) []string {
	if !strings.Contains(text, delimiter) {
		// no delimiter found, return original text
		return []string{text}
	}

	parts := strings.Split(text, delimiter)
	slog.Debug("Split by delimiter", "strategy", strategyName, "delimiter", delimiter, "parts", len(parts))

	// prepare segments with proper delimiter restoration
	var segments []string
	switch strategyName {
	case "sentence":
		// for sentences, add the period back to each part (except the last)
		for i, part := range parts {
			if trimSpacesOnly(part) == "" {
				continue // skip empty parts
			}

			if i < len(parts)-1 {
				// add period back to maintain sentence structure
				segments = append(segments, trimSpacesOnly(part)+".")
			} else {
				// last part doesn't get a period added
				segments = append(segments, trimSpacesOnly(part))
			}
		}

	case "sentence-question":
		// for question sentences, add the question mark back to each part (except the last)
		for i, part := range parts {
			if trimSpacesOnly(part) == "" {
				continue // Skip empty parts
			}

			if i < len(parts)-1 {
				// add question mark back to maintain sentence structure
				segments = append(segments, trimSpacesOnly(part)+"?")
			} else {
				// last part doesn't get a question mark added
				segments = append(segments, trimSpacesOnly(part))
			}
		}

	case "sentence-exclamation":
		// for exclamation sentences, add the exclamation mark back to each part (except the last)
		for i, part := range parts {
			if trimSpacesOnly(part) == "" {
				continue // Skip empty parts
			}

			if i < len(parts)-1 {
				// add exclamation mark back to maintain sentence structure
				segments = append(segments, trimSpacesOnly(part)+"!")
			} else {
				// last part doesn't get an exclamation mark added
				segments = append(segments, trimSpacesOnly(part))
			}
		}

	case "line":
		// for lines, preserve newlines when recombining
		for i, part := range parts {
			if trimmed := trimSpacesOnly(part); trimmed != "" {
				if i < len(parts)-1 {
					// add newline back to maintain line structure
					segments = append(segments, trimmed+"\n")
				} else {
					// last part doesn't get a newline added
					segments = append(segments, trimmed)
				}
			}
		}

	case "paragraph":
		// for paragraphs, preserve double newlines when recombining
		for i, part := range parts {
			if trimmed := trimSpacesOnly(part); trimmed != "" {
				if i < len(parts)-1 {
					// add double newline back to maintain paragraph structure
					segments = append(segments, trimmed+"\n\n")
				} else {
					// last part doesn't get double newlines added
					segments = append(segments, trimmed)
				}
			}
		}

	default:
		// for words - preserve formatting while trimming spaces
		for _, part := range parts {
			if trimmed := trimSpacesOnly(part); trimmed != "" {
				segments = append(segments, trimmed)
			}
		}
	}

	// we're trying to simply prevent over-splitting while still breaking up oversized chunks
	minChunkSize := calculateMinimumChunkSize(maxChunkSize)
	return packSegments(segments, strategyName, maxChunkSize, minChunkSize)
}

// packSegments combines multiple segments into reasonably-sized chunks.
// This is the key improvement over naive splitting - we try to keep related content together.
// For non-word strategies, it also merges segments below minChunkSize to prevent overly short chunks.
func packSegments(segments []string, strategyName string, maxChunkSize int, minChunkSize int) []string {
	if len(segments) == 0 {
		return []string{}
	}

	// For word-level splitting, we want to pack multiple words together
	if strategyName == "word" {
		return packWords(segments, maxChunkSize)
	}

	// For higher-level splitting (sentences, paragraphs, lines), merge segments below minimum size
	return mergeShortSegments(segments, maxChunkSize, minChunkSize)
}

// calculateMinimumChunkSize determines the minimum acceptable chunk size
// to prevent overly short segments from remaining isolated
func calculateMinimumChunkSize(maxChunkSize int) int {
	// use 25% of maxChunkSize as minimum, with an absolute minimum of 3 characters
	minSize := int(float64(maxChunkSize) * 0.25)
	if minSize < 3 {
		minSize = 3
	}
	return minSize
}

// packWords combines word segments into reasonably-sized chunks
func packWords(segments []string, maxChunkSize int) []string {
	var result []string
	var currentChunk strings.Builder

	for _, segment := range segments {
		// calculate space needed
		spaceNeeded := len(segment)
		if currentChunk.Len() > 0 {
			spaceNeeded += 1 // for space separator
		}

		// if adding this segment would exceed the maxChunkSize, finalize current chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+spaceNeeded > maxChunkSize {
			// current chunk is getting big enough, finalize it
			if chunk := trimSpacesOnly(currentChunk.String()); chunk != "" {
				result = append(result, chunk)
			}
			currentChunk.Reset()
		}

		// add the segment to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(segment)
	}

	// add final chunk
	if chunk := trimSpacesOnly(currentChunk.String()); chunk != "" {
		result = append(result, chunk)
	}

	return result
}

// mergeShortSegments merges segments below minChunkSize with adjacent segments
// to prevent overly short chunks like initials from remaining isolated
func mergeShortSegments(segments []string, maxChunkSize int, minChunkSize int) []string {
	if len(segments) <= 1 {
		return segments
	}

	var result []string
	i := 0

	for i < len(segments) {
		currentSegment := segments[i]

		// if current segment is long enough, keep it as-is
		if len(currentSegment) >= minChunkSize {
			result = append(result, currentSegment)
			i++
			continue
		}

		// current segment is too short, try to merge with next segment
		if i+1 < len(segments) {
			nextSegment := segments[i+1]
			combined := currentSegment + " " + nextSegment

			// if combining doesn't exceed maxChunkSize, merge them
			if len(combined) <= maxChunkSize {
				// merged successfully, continue with the combined segment
				segments[i+1] = combined
				i++ // skip current, process combined segment in next iteration
				continue
			}
		}

		// can't merge with next, try to merge with previous (if we have accumulated results)
		if len(result) > 0 {
			lastResult := result[len(result)-1]
			combined := lastResult + " " + currentSegment

			// if combining doesn't exceed maxChunkSize, merge with previous
			if len(combined) <= maxChunkSize {
				result[len(result)-1] = combined
				i++
				continue
			}
		}

		// can't merge anywhere, keep the short segment as-is
		result = append(result, currentSegment)
		i++
	}

	return result
}

// trimSpacesOnly removes leading and trailing spaces and tabs but preserves line breaks.
// This is used to clean up chunks while maintaining intentional formatting like poetry line breaks.
func trimSpacesOnly(s string) string {
	// Handle empty string
	if s == "" {
		return s
	}

	// Find first non-space, non-tab character
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	// Find last non-space, non-tab character
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}
