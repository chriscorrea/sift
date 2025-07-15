package classify_test

import (
	"testing"

	"github.com/chriscorrea/sift/internal/classify"
)

func TestNewClassifier(t *testing.T) {
	classifier := classify.NewClassifier()
	if classifier == nil {
		t.Fatal("NewClassifier() returned nil")
	}
}

func TestClassifier_IsExtraneous(t *testing.T) {
	classifier := classify.NewClassifier()

	tests := []struct {
		name        string
		chunkText   string
		chunkIndex  int
		totalChunks int
		expected    bool
		description string
	}{
		{
			name:        "empty chunk",
			chunkText:   "",
			chunkIndex:  0,
			totalChunks: 1,
			expected:    true,
			description: "empty chunks should be classified as extraneous",
		},
		{
			name:        "whitespace only chunk",
			chunkText:   "   \n\t  ",
			chunkIndex:  0,
			totalChunks: 1,
			expected:    true,
			description: "whitespace-only chunks should be classified as extraneous",
		},
		{
			name:        "copyright footer at end",
			chunkText:   "Copyright 2026. All rights reserved. This text may not be reproduced without permission.",
			chunkIndex:  9,
			totalChunks: 10,
			expected:    true,
			description: "copyright text at document end should be classified as extraneous",
		},
		{
			name:        "navigation header at beginning",
			chunkText:   "Home About Profile Share Content Navigation Footer",
			chunkIndex:  0,
			totalChunks: 10,
			expected:    true,
			description: "navigation text at document beginning should be classified as extraneous",
		},
		{
			name:        "main content in middle",
			chunkText:   "The carrot cake recipe requires sifting flour through a fine mesh sieve to achieve the perfect texture. This traditional baking technique removes lumps and aerates the flour, ensuring a light and fluffy cake.",
			chunkIndex:  5,
			totalChunks: 10,
			expected:    false,
			description: "main content in middle should not be classified as extraneous",
		},
		{
			name:        "mixed content with some stopwords",
			chunkText:   "The baker carefully sifted confectioner sugar for the icing. The page contained detailed instructions for this important step in carrot cake preparation.",
			chunkIndex:  3,
			totalChunks: 8,
			expected:    false,
			description: "content with moderate stopwords should not be extraneous",
		},
		{
			name:        "isbn and publishing info",
			chunkText:   "ISBN 479-04550 Published by Publications Department of Federal Publishing Standards",
			chunkIndex:  0,
			totalChunks: 5,
			expected:    true,
			description: "publishing metadata should be classified as extraneous",
		},
		{
			name:        "single chunk document",
			chunkText:   "This is the complete content of a very short document about sifting flour for baking.",
			chunkIndex:  0,
			totalChunks: 1,
			expected:    false,
			description: "single chunk documents should use moderate threshold",
		},
		{
			name:        "academic appendix",
			chunkText:   "Appendix A: Figure 1 References: Lorem Ipsum Foundation Publications, 2023.",
			chunkIndex:  7,
			totalChunks: 8,
			expected:    true,
			description: "academic appendices should be classified as extraneous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.IsExtraneous(tt.chunkText, tt.chunkIndex, tt.totalChunks)
			if result != tt.expected {
				t.Errorf("IsExtraneous() = %v, expected %v\nChunk: %q\nPosition: %d/%d\nDescription: %s",
					result, tt.expected, tt.chunkText, tt.chunkIndex+1, tt.totalChunks, tt.description)
			}
		})
	}
}

func TestClassifier_ThresholdCalculation(t *testing.T) {
	classifier := classify.NewClassifier()

	// We can't directly test the threshold calculation,
	// but we can test behavior with identical content at different positions

	// example ambiguous text that could be classified either way.
	moderateStopwordText := "Hello there! This is some valid text that contains a bit of publishing terminology copyright 2025"

	tests := []struct {
		name        string
		chunkIndex  int
		totalChunks int
		description string
	}{
		{
			name:        "beginning position",
			chunkIndex:  0,
			totalChunks: 10,
			description: "first chunk should have lower threshold",
		},
		{
			name:        "end position",
			chunkIndex:  9,
			totalChunks: 10,
			description: "last chunk should have lower threshold",
		},
		{
			name:        "middle position",
			chunkIndex:  5,
			totalChunks: 10,
			description: "middle chunk should have higher threshold",
		},
	}

	beginningResult := classifier.IsExtraneous(moderateStopwordText, tests[0].chunkIndex, tests[0].totalChunks)
	endResult := classifier.IsExtraneous(moderateStopwordText, tests[1].chunkIndex, tests[1].totalChunks)
	middleResult := classifier.IsExtraneous(moderateStopwordText, tests[2].chunkIndex, tests[2].totalChunks)

	t.Logf("Position-based classification results:")
	t.Logf("  Beginning (0/10): %v", beginningResult)
	t.Logf("  End (9/10): %v", endResult)
	t.Logf("  Middle (5/10): %v", middleResult)

	// confirm that edges are classified as extraneous but not middle
	if !beginningResult {
		t.Error("Expected beginning position to be classified as extraneous")
	}
	if !endResult {
		t.Error("Expected end position to be classified as extraneous")
	}
	if middleResult {
		t.Error("Expected middle position to NOT be classified as extraneous")
	}
}

func TestClassifier_EdgeCases(t *testing.T) {
	classifier := classify.NewClassifier()

	tests := []struct {
		name        string
		chunkText   string
		chunkIndex  int
		totalChunks int
		expected    bool
		description string
	}{
		{
			name:        "zero total chunks",
			chunkText:   "some text",
			chunkIndex:  0,
			totalChunks: 0,
			expected:    false,
			description: "should handle zero total chunks gracefully",
		},
		{
			name:        "negative chunk index",
			chunkText:   "some text",
			chunkIndex:  -1,
			totalChunks: 5,
			expected:    false,
			description: "should handle negative chunk index gracefully",
		},
		{
			name:        "chunk index beyond total",
			chunkText:   "some text",
			chunkIndex:  10,
			totalChunks: 5,
			expected:    false,
			description: "should handle chunk index beyond total gracefully",
		},
		{
			name:        "very long text with no stopwords",
			chunkText:   "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua ut enim ad minim veniam quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur",
			chunkIndex:  2,
			totalChunks: 5,
			expected:    false,
			description: "long text with no stopwords should not be extraneous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// should not panic and should return a reasonable result
			result := classifier.IsExtraneous(tt.chunkText, tt.chunkIndex, tt.totalChunks)
			if result != tt.expected {
				t.Errorf("IsExtraneous() = %v, expected %v for edge case: %s",
					result, tt.expected, tt.description)
			}
		})
	}
}
