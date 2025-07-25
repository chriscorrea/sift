// Extract provides content extraction utilities for the sift CLI tool.
// It handles extracting and processing text from various sources and formats.
package extract

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
)

// ToMarkdown extracts the main content from HTML and converts it to Markdown.
// Optional CSS selector filtering is supported.
//
// Parameters:
//   - content: io.Reader containing HTML content
//   - selector: optional CSS selector to filter content (empty string for main content extraction)
//   - includeAll: if true, skips readability extraction and converts all HTML content
//   - baseURL: optional URL for context during readability extraction (can be nil)
//
// Returns clean Markdown string or error if extraction/conversion fails.
func ToMarkdown(content io.Reader, selector string, includeAll bool, baseURL *url.URL) (string, error) {
	// if selector is specified, use it (override includeAll setting)
	if selector != "" {
		return extractWithSelector(content, selector)
	}

	// if includeAll is true, convert entire HTML without readability filtering
	if includeAll {
		return convertAllHTML(content)
	}

	// default: use go-readability to extract main content
	return extractMainContent(content, baseURL)
}

// extractMainContent uses go-readability to extract the main article content
func extractMainContent(content io.Reader, baseURL *url.URL) (string, error) {
	// use empty URL if none provided
	if baseURL == nil {
		baseURL = &url.URL{}
	}

	// parse with go-readability to extract main content directly from reader
	article, err := readability.FromReader(content, baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to extract main content: %w", err)
	}

	// convert extracted HTML to Markdown
	return convertToMarkdown(article.Content)
}

// extractWithSelector uses a CSS selector to extract specific content
func extractWithSelector(content io.Reader, selector string) (string, error) {
	// parse HTML with goquery directly from reader
	doc, err := goquery.NewDocumentFromReader(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// find elements matching the selector
	selection := doc.Find(selector)
	if selection.Length() == 0 {
		return "", fmt.Errorf("no elements found matching selector: %s", selector)
	}

	// get the HTML content of all selected elements
	var htmlParts []string
	selection.Each(func(i int, s *goquery.Selection) {
		html, err := s.Html()
		if err == nil {
			// wrap each element to preserve structure
			tagName := goquery.NodeName(s)
			htmlParts = append(htmlParts, fmt.Sprintf("<%s>%s</%s>", tagName, html, tagName))
		}
	})

	if len(htmlParts) == 0 {
		return "", fmt.Errorf("failed to extract HTML from selection")
	}

	selectedHTML := strings.Join(htmlParts, "\n")

	// convert selected HTML to Markdown
	return convertToMarkdown(selectedHTML)
}

// convertAllHTML converts all HTML content to Markdown without filtering
// TODO: Implement streaming HTML parsing to handle arbitrarily large files
func convertAllHTML(content io.Reader) (string, error) {
	// read all content from the reader for full HTML conversion
	htmlBytes, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("failed to read HTML content: %w", err)
	}

	// convert the entire HTML content to Markdown
	return convertToMarkdown(string(htmlBytes))
}

// convertToMarkdown converts HTML string to clean Markdown
func convertToMarkdown(htmlString string) (string, error) {
	// create converter with options for clean output
	converter := md.NewConverter("", true, nil)

	// add custom rule for <br> tags to produce single newlines
	converter.AddRules(
		md.Rule{
			Filter: []string{"br"},
			Replacement: func(content string, selec *goquery.Selection, opt *md.Options) *string {
				return md.String("\n")
			},
		},
	)

	// convert HTML to Markdown
	markdown, err := converter.ConvertString(htmlString)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to Markdown: %w", err)
	}

	// gentle cleanup: normalize excessive whitespace
	// preserve single and double newlines, including "  \n" patterns from <br> tags
	cleaned := markdown

	// normalize 3+ consecutive newlines to 2
	for strings.Contains(cleaned, "\n\n\n") {
		cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	}

	// only trim leading/trailing whitespace if it's truly excessive
	// preserve a single trailing newline if present and preserve line break patterns
	if strings.HasSuffix(cleaned, "\n") {
		// trim any whitespace before the final newline, but preserve meaningful patterns
		cleaned = strings.TrimRight(cleaned, " \t")
	} else {
		// no trailing newline, use gentle trimming that preserves line break patterns
		cleaned = trimSpacesOnlyFromString(cleaned)
	}

	return cleaned, nil
}

// trimSpacesOnlyFromString removes leading and trailing spaces and tabs but preserves line breaks.
// This is used to clean up markdown while maintaining intentional formatting like line breaks from <br> tags.
func trimSpacesOnlyFromString(s string) string {
	// handle empty string
	if s == "" {
		return s
	}

	// find first non-space, non-tab character
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	// find last non-space, non-tab character
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}
