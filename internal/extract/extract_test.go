package extract_test

import (
	"strings"
	"testing"

	"sift/internal/extract"
)

const (
	simpleHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Test Article</title>
</head>
<body>
    <header>
        <h1>Site Header</h1>
        <nav>Navigation</nav>
    </header>
    <main>
        <article>
            <h1>Main Article Title</h1>
            <p>This is the main content of the article. It contains important information.</p>
            <p>This is a second paragraph with <strong>bold text</strong> and <em>italic text</em>.</p>
            <ul>
                <li>First list item</li>
                <li>Second list item</li>
            </ul>
        </article>
    </main>
    <aside>
        <p>This is sidebar content that should be filtered out.</p>
    </aside>
    <footer>
        <p>Footer content</p>
    </footer>
</body>
</html>`

	blogPostHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Blog Post</title>
</head>
<body>
    <div class="container">
        <header class="site-header">
            <h1>My Blog</h1>
        </header>
        <div class="content">
            <article class="blog-post">
                <h2>How to Bake the Perfect Carrot Cake</h2>
                <p class="meta">Published on July 5, 2018</p>
                <div class="post-content">
                    <p>Baking a perfect carrot cake requires <strong>sifting flour</strong> for the finest texture.</p>
                    <h3>Ingredients</h3>
                    <ul>
                        <li>2 cups flour (definitely sifted)</li>
                        <li>1 cup carrots, grated</li>
                        <li>3 eggs</li>
                    </ul>
                    <h3>Instructions</h3>
                    <ol>
                        <li>Sift the flour and mix dry ingredients together</li>
                        <li>Mix wet ingredients separately</li>
                        <li>Combine and bake at 349°F</li>
                    </ol>
                    <blockquote>
                        <p>The secret is in the sifting!</p>
                    </blockquote>
                </div>
            </article>
        </div>
        <aside class="sidebar">
            <h3>Related Posts</h3>
            <ul>
                <li><a href="#">Chocolate Cake Recipe</a></li>
                <li><a href="#">Vanilla Frosting Tips</a></li>
            </ul>
        </aside>
    </div>
</body>
</html>`

	malformedHTML = `<html>
<body>
    <div class="content">
        <h1>Unclosed Header
        <p>Paragraph without closing tag
        <div class="nested">
            <span>Some text</span>
        </div>
    </div>
</body>`
)

func TestToMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		selector    string
		expectError bool
		expectEmpty bool
		contains    []string
		notContains []string
	}{
		{
			name:        "simple HTML without selector (main content extraction)",
			html:        simpleHTML,
			selector:    "",
			expectError: false,
			contains:    []string{"Main Article Title", "main content", "bold text", "italic text", "First list item"},
			notContains: []string{"Site Header", "Navigation", "sidebar content", "Footer content"},
		},
		{
			name:        "blog post without selector",
			html:        blogPostHTML,
			selector:    "",
			expectError: false,
			contains:    []string{"How to Bake", "carrot cake", "sifting flour", "Ingredients", "Instructions"},
			notContains: []string{"My Blog", "Related Posts"},
		},
		{
			name:        "with article selector",
			html:        simpleHTML,
			selector:    "article",
			expectError: false,
			contains:    []string{"Main Article Title", "main content", "bold text", "First list item"},
			notContains: []string{"Site Header", "Navigation", "sidebar content", "Footer"},
		},
		{
			name:        "with specific class selector",
			html:        blogPostHTML,
			selector:    ".post-content",
			expectError: false,
			contains:    []string{"sifting flour", "Ingredients", "Instructions", "2 cups flour", "The secret is in the sifting"},
			notContains: []string{"How to Bake", "Published on", "My Blog", "Related Posts"},
		},
		{
			name:        "with h3 selector (multiple elements)",
			html:        blogPostHTML,
			selector:    "h3",
			expectError: false,
			contains:    []string{"Ingredients", "Instructions"},
			notContains: []string{"How to Bake", "carrot cake", "sifting flour"},
		},
		{
			name:        "with list selector",
			html:        blogPostHTML,
			selector:    "ol",
			expectError: false,
			contains:    []string{"Sift the flour", "Mix wet ingredients", "Combine and bake"},
			notContains: []string{"Ingredients", "2 cups flour"},
		},
		{
			name:        "with blockquote selector",
			html:        blogPostHTML,
			selector:    "blockquote",
			expectError: false,
			contains:    []string{"The secret is in the sifting"},
			notContains: []string{"Ingredients", "Instructions"},
		},
		{
			name:        "non-existent selector",
			html:        simpleHTML,
			selector:    ".non-existent",
			expectError: true,
		},
		{
			name:        "invalid selector",
			html:        simpleHTML,
			selector:    ">>invalid<<",
			expectError: true,
		},
		{
			name:        "malformed HTML with selector",
			html:        malformedHTML,
			selector:    ".content",
			expectError: false,
			contains:    []string{"Unclosed Header", "Paragraph without closing", "Some text"},
		},
		{
			name:        "empty HTML",
			html:        "",
			selector:    "",
			expectError: false,
			expectEmpty: true,
		},
		{
			name:        "whitespace only HTML",
			html:        "   \n\t   ",
			selector:    "",
			expectError: false,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.html)
			result, err := extract.ToMarkdown(reader, tt.selector, false, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("ToMarkdown() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ToMarkdown() unexpected error: %v", err)
			}

			if tt.expectEmpty {
				if strings.TrimSpace(result) != "" {
					t.Errorf("ToMarkdown() expected empty result but got: %q", result)
				}
				return
			}

			// Check that expected content is present
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("ToMarkdown() result should contain %q but doesn't.\nResult: %s", expected, result)
				}
			}

			// check that unwanted content is not present
			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("ToMarkdown() result should not contain %q but does.\nResult: %s", notExpected, result)
				}
			}

			// reasonableness test for valid Markdown; no tags are present
			// (...yes I know tags are allowed, but let's keep it simple and reasonable)
			if strings.TrimSpace(result) != "" {
				// should not contain raw HTML tags in the final output
				htmlTags := []string{"<div>", "<span>", "<article>", "</div>", "</span>", "</article>"}
				for _, tag := range htmlTags {
					if strings.Contains(result, tag) {
						t.Errorf("ToMarkdown() result contains raw HTML tag %q, should be converted to Markdown", tag)
					}
				}
			}
		})
	}
}

func TestToMarkdownFormats(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		selector  string
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:     "headers converted to markdown",
			html:     `<html><body><h1>Header 1</h1><h2>Header 2</h2><h3>Header 3</h3></body></html>`,
			selector: "body",
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "# Header 1") &&
					!strings.Contains(result, "Header 1\n=") {
					t.Errorf("H1 should be converted to Markdown header format")
				}
				if !strings.Contains(result, "## Header 2") &&
					!strings.Contains(result, "Header 2\n-") {
					t.Errorf("H2 should be converted to Markdown header format")
				}
			},
		},
		{
			name:     "lists converted to markdown",
			html:     `<html><body><ul><li>Item 1</li><li>Item 2</li></ul><ol><li>First</li><li>Second</li></ol></body></html>`,
			selector: "body",
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "- Item 1") && !strings.Contains(result, "* Item 1") {
					t.Errorf("Unordered list should be converted to Markdown format")
				}
				if !strings.Contains(result, "1. First") {
					t.Errorf("Ordered list should be converted to Markdown format")
				}
			},
		},
		{
			name:     "emphasis converted to markdown",
			html:     `<html><body><p>This is <strong>bold</strong> and <em>italic</em> text.</p></body></html>`,
			selector: "body",
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "**bold**") && !strings.Contains(result, "__bold__") {
					t.Errorf("Strong should be converted to Markdown bold format")
				}
				if !strings.Contains(result, "*italic*") && !strings.Contains(result, "_italic_") {
					t.Errorf("Em should be converted to Markdown italic format")
				}
			},
		},
		{
			name:     "blockquotes converted to markdown",
			html:     `<html><body><blockquote><p>This is a quote about sifting confectioner sugar for icing.</p></blockquote></body></html>`,
			selector: "body",
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "> This is a quote") {
					t.Errorf("Blockquote should be converted to Markdown format with > prefix")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.html)
			result, err := extract.ToMarkdown(reader, tt.selector, false, nil)

			if err != nil {
				t.Fatalf("ToMarkdown() unexpected error: %v", err)
			}

			tt.checkFunc(t, result)
		})
	}
}

func TestToMarkdownEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		selector    string
		expectError bool
		description string
	}{
		{
			name:        "only whitespace content",
			html:        `<html><body><div>   \n\t   </div></body></html>`,
			selector:    "div",
			expectError: false,
			description: "should handle whitespace-only content gracefully",
		},
		{
			name:        "nested selectors",
			html:        `<html><body><div class="outer"><div class="inner">Content</div></div></body></html>`,
			selector:    ".outer .inner",
			expectError: false,
			description: "should handle nested CSS selectors",
		},
		{
			name:        "multiple matching elements",
			html:        `<html><body><p>Para 1</p><p>Para 2</p><p>Para 3</p></body></html>`,
			selector:    "p",
			expectError: false,
			description: "should handle multiple matching elements",
		},
		{
			name:        "complex nested HTML",
			html:        `<html><body><div><article><header><h1>Title</h1></header><section><p>Content</p></section></article></div></body></html>`,
			selector:    "article",
			expectError: false,
			description: "should handle complex nested structures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.html)
			result, err := extract.ToMarkdown(reader, tt.selector, false, nil)

			if tt.expectError && err == nil {
				t.Errorf("ToMarkdown() expected error but got none for case: %s", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("ToMarkdown() unexpected error for case %s: %v", tt.description, err)
			}

			if !tt.expectError && err == nil {
				// just verify we get a result without crashing
				_ = result
			}
		})
	}
}
