// Package classify provides text classification capabilities for filtering out extraneous content.
//
// The classify package implements a simple classifier that identifies and filters
// non-essential text chunks such as headers, footers, navigation elements, and publishing
// metadata. It uses stopword analysis and position-based thresholding to determine which
// chunks should be considered extraneous.
package classify

import (
	"math"
	"regexp"
	"strings"

	"github.com/kljensen/snowball"
)

// extraneousStopwords contains stemmed words that commonly appear in extraneous content
// such as headers, footers, navigation, and publishing metadata
// TODO: expand corpus and refine stopword list based on real-world data
var extraneousStopwords = map[string]struct{}{
	// --- Publishing & Document Structure ---
	"author":    {},
	"appendix":  {},
	"book":      {},
	"chapter":   {},
	"content":   {}, // from "table of contents"
	"edit":      {}, // from "edition"
	"ebook":     {},
	"footer":    {},
	"glossari":  {},
	"gutenberg": {}, // from "Project Gutenberg"
	"navig":     {},
	"note":      {},
	"page":      {},
	"project":   {},
	"publish":   {},
	"text":      {}, // from "full text", "plain text"

	// --- Navigation & Interaction ---
	"about":  {},
	"locat":  {}, // from "location"
	"profil": {},
	"share":  {},
	"updat":  {},

	// --- Legal & Footer Text ---
	"copyright": {},
	"manag":     {},
	"permiss":   {},
	"polici":    {},
	"privaci":   {},
	"public":    {},
	"purpos":    {},
	"reproduc":  {},
	"reserv":    {},
	"right":     {},
	"risk":      {},
	"standard":  {},
	"term":      {},
	"use":       {},

	// --- Academic & Technical References ---
	"citat":   {},
	"depart":  {},
	"edu":     {},
	"feder":   {},
	"foundat": {},
	"https":   {}, // from URLs
	"isbn":    {},
	"refer":   {},
}

// Classifier identifies and filters extraneous text chunks using stopword analysis
// and position-based thresholding
type Classifier struct {
	// tokenRegex extracts word tokens from text
	tokenRegex *regexp.Regexp
}

// NewClassifier creates and initializes a new Classifier instance
func NewClassifier() *Classifier {
	return &Classifier{
		tokenRegex: regexp.MustCompile(`\b[a-zA-Z]+\b`),
	}
}

// IsExtraneous determines if a chunk should be classified as extraneous content.
// It analyzes the ratio of stopwords to total tokens and applies a position-adjusted
// threshold that is lower for chunks at the beginning and end of documents.
//
// Parameters:
//   - chunkText: the text content of the chunk to analyze
//   - chunkIndex: zero-based index of the chunk within the document
//   - totalChunks: total number of chunks in the document
//
// Returns true if the chunk is classified as extraneous and should be filtered out.
func (c *Classifier) IsExtraneous(chunkText string, chunkIndex int, totalChunks int) bool {
	// edge cases; invalid params should not be classified as extraneous
	if totalChunks <= 0 || chunkIndex < 0 || chunkIndex >= totalChunks {
		return false
	}

	// extract word tokens from the chunk text
	tokens := c.tokenRegex.FindAllString(strings.ToLower(chunkText), -1)
	if len(tokens) == 0 {
		// empty chunks are considered extraneous
		return true
	}

	// count stopwords by stemming each token and checking against our stopword set
	stopwordCount := 0
	for _, token := range tokens {
		// stem the token using English stemmer
		stemmed, err := snowball.Stem(token, "english", true)
		if err != nil {
			// if stemming fails, use the original token
			stemmed = token
		}

		if _, isStopword := extraneousStopwords[stemmed]; isStopword {
			stopwordCount++
		}
	}

	// calculate the ratio of stopwords to total tokens
	stopwordRatio := float64(stopwordCount) / float64(len(tokens))

	// calculate position-adjusted threshold
	threshold := c.calculateThreshold(chunkIndex, totalChunks)

	// classify as extraneous if stopword ratio exceeds the threshold
	return stopwordRatio > threshold
}

// calculateThreshold computes a dynamic threshold based on chunk position.
// The threshold is lower for chunks at the beginning and end of documents
// (where headers, footers, and navigation are most commonly placed) and higher
// for chunks in the middle (where higher-density content is more likely).
func (c *Classifier) calculateThreshold(chunkIndex int, totalChunks int) float64 {
	// edge cases
	if totalChunks <= 0 {
		return 0.33 // Default moderate threshold
	}
	if chunkIndex < 0 || chunkIndex >= totalChunks {
		return 0.33 // default for out-of-bounds indices
	}
	if totalChunks <= 3 {
		// For small docs, use a high threshold to avoid false positive
		return 0.5
	}

	// calculate relative position (0.0 to 1.0)
	relativePosition := float64(chunkIndex) / float64(totalChunks-1)

	// deploy inverted V curve
	positionFactor := 1.0 - math.Abs(2.0*relativePosition-1.0)

	// define threshold range: 0.1 (edges) to 0.33 (middle)
	minThreshold := 0.1  // Low threshold for first/last 10%
	maxThreshold := 0.33 // High threshold for middle content

	// interpolate between min & max based on position factor
	threshold := minThreshold + (maxThreshold-minThreshold)*positionFactor

	return threshold
}
