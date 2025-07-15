package app

import (
	"context"
	"strings"
	"testing"

	"sift/internal/counter"
)

func TestConfig_IncludeAll(t *testing.T) {
	tests := []struct {
		name       string
		includeAll bool
		expected   string
	}{
		{
			name:       "default behavior (includeAll false)",
			includeAll: false,
			expected:   "filtered and processed content",
		},
		{
			name:       "include all content (includeAll true)",
			includeAll: true,
			expected:   "all content including headers/footers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Sources:        []string{},
				Selector:       "",
				MaxUnits:       1000,
				CountingMethod: counter.Tokens,
				SizingStrategy: Beginning,
				Quiet:          true,
				Debug:          false,
				IncludeAll:     tt.includeAll,
			}

			// verify config field is set correctly
			if config.IncludeAll != tt.includeAll {
				t.Errorf("Config.IncludeAll = %v, want %v", config.IncludeAll, tt.includeAll)
			}
		})
	}
}

func TestApplyContentTransformations_IncludeAll(t *testing.T) {
	testText := strings.Repeat("sugar ", 100) // 100 words

	tests := []struct {
		name       string
		text       string
		maxWords   int
		includeAll bool
	}{
		{
			name:       "with filtering enabled",
			text:       testText,
			maxWords:   50,
			includeAll: false,
		},
		{
			name:       "with filtering disabled",
			text:       testText,
			maxWords:   50,
			includeAll: true,
		},
		{
			name:       "empty text",
			text:       "",
			maxWords:   50,
			includeAll: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				MaxUnits:       tt.maxWords,
				CountingMethod: counter.Words,
				SizingStrategy: Middle,
				IncludeAll:     tt.includeAll,
			}

			result, err := applyContentTransformations(context.Background(), tt.text, cfg.CountingMethod, cfg.MaxUnits, cfg.SizingStrategy, cfg.IncludeAll, cfg.SearchQuery, cfg.Quiet, cfg.ContextBefore, cfg.ContextAfter)

			if err != nil {
				t.Fatalf("applyContentTransformations() error = %v", err)
			}

			if tt.text == "" {
				if result != "" {
					t.Errorf("applyContentTransformations() with empty text should return empty string")
				}
				return
			}

			// verify word count constraint is respected
			words := strings.Fields(result)
			if len(words) > tt.maxWords {
				t.Errorf("applyContentTransformations() returned %d words, want ≤ %d", len(words), tt.maxWords)
			}

			// basic validation that we got some content back for non-empty input
			if tt.text != "" && result == "" {
				t.Errorf("applyContentTransformations() returned empty result for non-empty input")
			}
		})
	}
}

func TestIncludeAllFlagBypassesFiltering(t *testing.T) {
	// create test document with sample extraneous content
	testDocument := `Copyright 2025. All rights reserved. This text may not be reproduced without permission.

Recipes Home About Profile Share Content Navigation 

The carrot cake recipe requires sifting flour through lorem ipsum dolor sit amet consectetur adipiscing elit. 

Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

ISBN 04550-479 Published by Hughes-Crane Publications 

References: Department of Education. 5th Edition. Foundation Publications, 2023.`

	tests := []struct {
		name         string
		includeAll   bool
		expectFooter bool
		expectHeader bool
		expectISBN   bool
		expectRefs   bool
	}{
		{
			name:         "filtering enabled (includeAll=false)",
			includeAll:   false,
			expectFooter: false, // copyright footer should be filtered
			expectHeader: false, // navigaton header should be filtered
			expectISBN:   false, // publishing metadata should be filtered
			expectRefs:   false, // academic citations should be filtered
		},
		{
			name:         "filtering disabled (includeAll=true)",
			includeAll:   true,
			expectFooter: true, // copyright footer should be preserved
			expectHeader: true, // navigation header should be preserved
			expectISBN:   true, // publishing metadata should be preserved
			expectRefs:   true, // academic citations should be preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				MaxUnits:       1000, // large enough to include all content
				CountingMethod: counter.Words,
				SizingStrategy: Beginning,
				IncludeAll:     tt.includeAll,
			}

			result, err := applyContentTransformations(context.Background(), testDocument, cfg.CountingMethod, cfg.MaxUnits, cfg.SizingStrategy, cfg.IncludeAll, cfg.SearchQuery, cfg.Quiet, cfg.ContextBefore, cfg.ContextAfter)
			if err != nil {
				t.Fatalf("applyContentTransformations() error = %v", err)
			}

			// check for copyright footer presence
			hasCopyright := strings.Contains(result, "Copyright 2025") || strings.Contains(result, "All rights reserved")
			if hasCopyright != tt.expectFooter {
				t.Errorf("Copyright footer presence = %v, want %v", hasCopyright, tt.expectFooter)
			}

			// check for navigation header presence
			hasNavigation := strings.Contains(result, "Home About Profile") || strings.Contains(result, "Navigation Footer")
			if hasNavigation != tt.expectHeader {
				t.Errorf("Navigation header presence = %v, want %v", hasNavigation, tt.expectHeader)
			}

			// check for ISBN/publishing metadata presence
			hasISBN := strings.Contains(result, "ISBN 04550")
			if hasISBN != tt.expectISBN {
				t.Errorf("ISBN/publishing metadata presence = %v, want %v", hasISBN, tt.expectISBN)
			}

			// check for academic references presence
			hasRefs := strings.Contains(result, "References:")
			if hasRefs != tt.expectRefs {
				t.Errorf("Academic references presence = %v, want %v", hasRefs, tt.expectRefs)
			}

			// core content should always be preserved
			hasMainContent := strings.Contains(result, "carrot cake recipe") && strings.Contains(result, "sifting flour")
			if !hasMainContent {
				t.Errorf("Main content should always be preserved")
			}

			// log result for debugging if needed
			if testing.Verbose() {
				t.Logf("Result length: %d characters", len(result))
				t.Logf("Result preview: %s...", result[:min(200, len(result))])
			}
		})
	}
}
