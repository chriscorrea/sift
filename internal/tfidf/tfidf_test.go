package tfidf

import (
	"math"
	"testing"
)

func TestNewCorpus(t *testing.T) {
	tests := []struct {
		name      string
		documents []string
		wantDocs  int
	}{
		{
			name:      "empty corpus",
			documents: []string{},
			wantDocs:  0,
		},
		{
			name:      "single document",
			documents: []string{"hello world"},
			wantDocs:  1,
		},
		{
			name:      "multiple documents",
			documents: []string{"hello world", "goodbye world", "hello goodbye"},
			wantDocs:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corpus := NewCorpus(tt.documents)
			if len(corpus.Documents) != tt.wantDocs {
				t.Errorf("NewCorpus() document count = %d, want %d", len(corpus.Documents), tt.wantDocs)
			}
			if corpus.TotalDocuments != tt.wantDocs {
				t.Errorf("NewCorpus() total documents = %d, want %d", corpus.TotalDocuments, tt.wantDocs)
			}
		})
	}
}

func TestCorpusScore(t *testing.T) {
	documents := []string{
		"the quick brown fox jumps over the lazy dog",
		"the brown dog runs quickly",
		"a fox and a dog are animals",
	}
	corpus := NewCorpus(documents)

	tests := []struct {
		name     string
		query    string
		docIndex int
		wantZero bool // true if we expect score to be 0
	}{
		{
			name:     "valid query and document",
			query:    "brown fox",
			docIndex: 0,
			wantZero: false,
		},
		{
			name:     "query with no matches",
			query:    "elephant",
			docIndex: 0,
			wantZero: true,
		},
		{
			name:     "empty query",
			query:    "",
			docIndex: 0,
			wantZero: true,
		},
		{
			name:     "invalid document index",
			query:    "brown",
			docIndex: 10,
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := corpus.Score(tt.query, tt.docIndex)
			if tt.wantZero && score != 0 {
				t.Errorf("Score() = %f, want 0", score)
			}
			if !tt.wantZero && score == 0 {
				t.Errorf("Score() = 0, want non-zero")
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "empty string",
			text: "",
			want: []string{},
		},
		{
			name: "simple words",
			text: "hello world",
			want: []string{"hello", "world"},
		},
		{
			name: "words with punctuation",
			text: "hello, world!",
			want: []string{"hello", "world"},
		},
		{
			name: "mixed case",
			text: "Hello World",
			want: []string{"hello", "world"},
		},
		{
			name: "numbers and underscores",
			text: "test_123 hello-world",
			want: []string{"test_123", "hello-world"},
		},
		{
			name: "short words filtered out",
			text: "a big cat in the house",
			want: []string{"big", "cat", "the", "house"},
		},
		{
			name: "multiple spaces and newlines",
			text: "hello   world\n\ntest",
			want: []string{"hello", "world", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.text)
			if len(got) != len(tt.want) {
				t.Errorf("tokenize() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, token := range got {
				if token != tt.want[i] {
					t.Errorf("tokenize() token[%d] = %s, want %s", i, token, tt.want[i])
				}
			}
		})
	}
}

func TestCalculateTermFrequency(t *testing.T) {
	tests := []struct {
		name   string
		tokens []string
		want   map[string]float64
	}{
		{
			name:   "empty tokens",
			tokens: []string{},
			want:   map[string]float64{},
		},
		{
			name:   "single token",
			tokens: []string{"hello"},
			want:   map[string]float64{"hello": 1.0},
		},
		{
			name:   "multiple tokens",
			tokens: []string{"hello", "world", "hello"},
			want:   map[string]float64{"hello": 2.0 / 3.0, "world": 1.0 / 3.0},
		},
		{
			name:   "all same tokens",
			tokens: []string{"test", "test", "test"},
			want:   map[string]float64{"test": 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateTermFrequency(tt.tokens)
			if len(got) != len(tt.want) {
				t.Errorf("calculateTermFrequency() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for term, wantFreq := range tt.want {
				gotFreq, exists := got[term]
				if !exists {
					t.Errorf("calculateTermFrequency() missing term %s", term)
					continue
				}
				if math.Abs(gotFreq-wantFreq) > 0.0001 {
					t.Errorf("calculateTermFrequency() term %s = %f, want %f", term, gotFreq, wantFreq)
				}
			}
		})
	}
}

func TestTFIDFScoring(t *testing.T) {
	// test with a realistic corpus
	documents := []string{
		"artificial intelligence and machine learning are new old technology",
		"machine learning algorithms require large datasets for training",
		"deep learning is a subset of machine learning using neural networks",
		"artificial intelligence companies are investing heavily in AGI myths",
	}
	corpus := NewCorpus(documents)

	// test queries that should match specific documents
	testCases := []struct {
		query       string
		expectFirst int // expected index of document with highest score
		description string
	}{
		{
			query:       "neural networks",
			expectFirst: 2, // should prefer doc 2 which has neural
			description: "specific technical term",
		},
		{
			query:       "datasets training",
			expectFirst: 1, // should prefer doc 1 which has both terms
			description: "training-related terms",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			scores := make([]float64, len(documents))
			for i := 0; i < len(documents); i++ {
				scores[i] = corpus.Score(tc.query, i)
			}

			// find the document with the highest score
			maxScore := scores[0]
			maxIndex := 0
			for i, score := range scores {
				if score > maxScore {
					maxScore = score
					maxIndex = i
				}
			}

			if maxIndex != tc.expectFirst {
				t.Errorf("TF-IDF scoring for query '%s': expected document %d to have highest score, but document %d did (scores: %v)",
					tc.query, tc.expectFirst, maxIndex, scores)
			}
		})
	}
}
