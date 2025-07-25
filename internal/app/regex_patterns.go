// Package app contains shared regex patterns for markdown field detection
package app

import (
	"regexp"
	"sync"
)

// regexPatterns holds compiled regex patterns for markdown field detection
type regexPatterns struct {
	headerRegex     *regexp.Regexp
	bulletListRegex *regexp.Regexp
	numberListRegex *regexp.Regexp
	codeBlockRegex  *regexp.Regexp
	inlineCodeRegex *regexp.Regexp
	boldRegex       *regexp.Regexp
	italicRegex     *regexp.Regexp
}

var (
	patterns     *regexPatterns
	patternsOnce sync.Once
)

// getRegexPatterns returns the singleton instance of compiled regex patterns
func getRegexPatterns() *regexPatterns {
	patternsOnce.Do(func() {
		patterns = &regexPatterns{
			headerRegex:     regexp.MustCompile(`^\s*#{1,6}\s+`),
			bulletListRegex: regexp.MustCompile(`^\s*[-*+]\s+`),
			numberListRegex: regexp.MustCompile(`^\s*\d+\.\s+`),
			codeBlockRegex:  regexp.MustCompile(`^\x60{3}|\x60{3}$`),
			inlineCodeRegex: regexp.MustCompile(`\x60[^\x60]+\x60`),
			boldRegex:       regexp.MustCompile(`\*\*[^*\s][^*]*[^*\s]\*\*|\*\*[^*\s]\*\*`),
			italicRegex:     regexp.MustCompile(`(?:^|[^*])\*[^*\s][^*]*[^*\s]\*(?:[^*]|$)|(?:^|[^*])\*[^*\s]\*(?:[^*]|$)`),
		}
	})
	return patterns
}
