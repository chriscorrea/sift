// Package tfidf provides TF-IDF (Term Frequency-Inverse Document Frequency) lexical search functionality.
//
// This package implements a corpus-based approach to document ranking using classical
// information retrieval techniques. It pre-calculates term frequencies and document
// frequencies for efficient query processing.
//
// The TF-IDF algorithm combines:
//   - Term Frequency (TF): How frequently a term appears in a document
//   - Inverse Document Frequency (IDF): How rare a term is across the corpus
//
// Usage Example:
//
//	corpus := tfidf.NewCorpus(documents)
//	score := corpus.Score("search query", documentIndex)
//
// The package uses simple tokenization suitable for keyword-based search,
// filtering out short words and normalizing case for better matching.
package tfidf

import (
	"log/slog"
	"math"
	"regexp"
	"strings"
)

// tokenRegex is compiled once at package initialization for efficient tokenization
var tokenRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// corpus holds the docs and pre-calculated TF-IDF data for efficient querying.
type Corpus struct {
	Documents       []string             // Original documents
	TermFrequencies []map[string]float64 // TF for each document
	DocFrequencies  map[string]int       // Document frequency for each term
	TotalDocuments  int                  // Total number of documents
}

// NewCorpus creates a new TF-IDF corpus from a collection of documents.
// It pre-calculates term frequencies and document frequencies for efficient scoring.
//
// Parameters:
//   - documents: slice of text documents to analyze
//
// Returns:
//   - *Corpus: configured corpus ready for scoring queries
//
// Constructor performs one-time analysis of all documents to calculate
// TF and IDF values, making subsequent query scoring very fast.
func NewCorpus(documents []string) *Corpus {
	if len(documents) == 0 {
		slog.Debug("Empty document collection provided")
		return &Corpus{
			Documents:       []string{},
			TermFrequencies: []map[string]float64{},
			DocFrequencies:  map[string]int{},
			TotalDocuments:  0,
		}
	}

	corpus := &Corpus{
		Documents:       documents,
		TermFrequencies: make([]map[string]float64, len(documents)),
		DocFrequencies:  make(map[string]int),
		TotalDocuments:  len(documents),
	}

	slog.Debug("Creating TF-IDF corpus", "documentCount", len(documents))

	// calculate term frequencies for each document
	for docIdx, doc := range documents {
		tokens := tokenize(doc)
		corpus.TermFrequencies[docIdx] = calculateTermFrequency(tokens)

		// track document frequency for each unique term
		uniqueTerms := make(map[string]bool)
		for _, token := range tokens {
			uniqueTerms[token] = true
		}
		for term := range uniqueTerms {
			corpus.DocFrequencies[term]++
		}

		// slog.Debug("Processed document", "docIndex", docIdx, "uniqueTerms", len(uniqueTerms), "totalTokens", len(tokens))
	}

	// slog.Debug("TF-IDF corpus created", "totalTerms", len(corpus.DocFrequencies), "documents", corpus.TotalDocuments)
	return corpus
}

// Score calculates the TF-IDF relevance score for a query against a specific document.
//
// Parameters:
//   - query: search query string
//   - docIndex: index of the document to score against
//
// Returns:
//   - float64: TF-IDF relevance score (higher scores indicate better matches)
//
// The score is calculated by summing TF-IDF values for each query term that
// appears in the document: terms that appear frequently in the document but
// rarely in the corpus receive higher scores.
func (c *Corpus) Score(query string, docIndex int) float64 {
	if docIndex < 0 || docIndex >= len(c.Documents) {
		slog.Debug("Invalid document index", "docIndex", docIndex, "totalDocs", len(c.Documents))
		return 0.0
	}

	queryTerms := tokenize(query)
	if len(queryTerms) == 0 {
		slog.Debug("Empty query after tokenization")
		return 0.0
	}

	docTF := c.TermFrequencies[docIndex]
	var totalScore float64

	for _, term := range queryTerms {
		tf := docTF[term]
		if tf == 0 {
			continue // term not in document
		}

		// calculate IDF: log(total_docs / docs_containing_term)
		docFreq := c.DocFrequencies[term]
		if docFreq == 0 {
			continue // term not in any document (shouldn't happen)
		}

		idf := math.Log(float64(c.TotalDocuments) / float64(docFreq))
		tfidf := tf * idf
		totalScore += tfidf

		slog.Debug("TF-IDF calculation", "term", term, "tf", tf, "df", docFreq, "idf", idf, "tfidf", tfidf)
	}

	slog.Debug("Document scoring completed", "docIndex", docIndex, "queryTerms", len(queryTerms), "totalScore", totalScore)
	return totalScore
}

// tokenize breaks text into normalized tokens suitable for TF-IDF analysis.
// It converts to lowercase, splits on non-alphanumeric characters, and filters
// out very short words that typically don't contribute to search relevance.
//
// Parameters:
//   - text: input text to tokenize
//
// Returns:
//   - []string: slice of normalized tokens
//
// Tokenization here is intentionally simple and designed for keyword-based search.
func tokenize(text string) []string {
	if text == "" {
		return []string{}
	}

	// convert to lowercase for case-insensitive matching
	text = strings.ToLower(text)

	// split on non-alphanumeric characters (excluding underscores and dashes)
	tokens := tokenRegex.Split(text, -1)

	// filter out empty strings and very short words (basic stop-word filtering)
	var filtered []string
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if len(token) >= 3 { // filter out words shorter than 3 characters
			filtered = append(filtered, token)
		}
	}

	return filtered
}

// calculateTermFrequency computes the term frequency for a slice of tokens.
// Returns a map where keys are terms and values are their frequencies.
//
// Parameters:
//   - tokens: slice of tokens from a document
//
// Returns:
//   - map[string]float64: term frequencies
//
// Term frequency is calculated as: (count of term in document) / (total terms in document)
func calculateTermFrequency(tokens []string) map[string]float64 {
	if len(tokens) == 0 {
		return map[string]float64{}
	}

	termCounts := make(map[string]int)
	for _, token := range tokens {
		termCounts[token]++
	}

	// calculate TF as relative frequency
	totalTerms := float64(len(tokens))
	termFreqs := make(map[string]float64)
	for term, count := range termCounts {
		termFreqs[term] = float64(count) / totalTerms
	}

	return termFreqs
}
