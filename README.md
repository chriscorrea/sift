# 🧑‍🍳 Sift

[![Go Version](https://img.shields.io/github/go-mod/go-version/chriscorrea/sift)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/chriscorrea/sift)](https://goreportcard.com/report/github.com/chriscorrea/sift)
[![CI](https://github.com/chriscorrea/sift/actions/workflows/push.yml/badge.svg?branch=main)](https://github.com/chriscorrea/sift/actions/workflows/push.yml)
[![Latest Release](https://img.shields.io/github/v/release/chriscorrea/sift)](https://github.com/chriscorrea/sift/releases)

**A prep tool for your text-based recipes**

`sift` is a text extraction tool for the command line. Use its search to pinpoint relevant information, or simply extract clean, structured content from URLs, files, or stdin. It's a composable tool for building data pipelines for your LLM workflows

## ✨ Highlights

- **Smart Content Extraction:** Automatically removes HTML, ads, and boilerplate to isolate the main content using Mozilla's Readability algorithm. You can also target specific elements with CSS selectors.

- **Field-Aware Search:** Pinpoint relevant information with a keyword search  that understands document structure.

- **Flexible I/O:** Process content from URLs, local files, or standard input with automatic source detection. Output formats include Markdown, plain text, or JSON.

- **Precise Output Sizing:** Control output size with token-level precision for LLM workflows, using the `cl100k_base` tokenizer, or by word and character counts.

- **Composable by Design:** Built as a native command-line tool, so you can easily pipe content in and chain with other tools to create powerful text processing workflows.

## Installation

### From Release Binaries

You can download a pre-compiled binary for your operating system from [latest releases](https://github.com/chriscorrea/sift/releases).

### Go Install
If you have a Go environment set up, you can install `sift` directly:
```bash
go install github.com/chriscorrea/sift/cmd/sift@latest
```

## Quick Start

Sift the main content from a webpage:

```bash
sift https://www.recipetineats.com/carrot-cake/
```

Target specific content with CSS selectors:

```bash
sift https://www.recipetineats.com/carrot-cake/ --selector ".wprm-recipe"
```

Find the most relevant content using keyword search (and limit to 200 tokens):
```bash
sift https://www.marcuse.org/herbert/pubs/64onedim/odmintro.html --search "technology" -t 200
```

Chain with other command line tools, such as [slop for LLMs](https://github.com/chriscorrea/slop):
```bash
sift https://www.recipetineats.com/carrot-cake/ | \
slop --yaml "build a shopping list, organized by aisle"
```

## Usage

### Flags

#### Extraction & Search
| Flag | Short | Description |
|---|---|---|
| `--search` | | Search for keywords and extract relevant context. |
| `--context-tokens` | | Token budget for smart context around search results (default is 200). |
| `--selector` | `-s` | CSS selector for content extraction. |
| `--include-all`| `-i`| Include all content without readability filtering. |

#### Output Sizing
| Flag | Short | Description |
|---|---|---|
| `--token-limit` | `-t` | Maximum number of tokens for output (effective default is 2500). |
| `--word-limit` | `-w` | Maximum number of words for output. |
| `--character-limit` | `-c` | Maximum number of characters for output. |
| `--beginning` | | Select content from the document's beginning (default). |
| `--middle` | | Select content from the document's middle, expanding outward. |
| `--end` | | Select content from the document's end, working backward. |

#### Formatting & Behavior
| Flag | Short | Description |
|---|---|---|
| `--md` | | Output in Markdown format (default). |
| `--text` | | Output in plain text format. |
| `--json` | | Output in JSON format. |

#### Other
| Flag | Short | Description |
|---|---|---|
| `--quiet`| `-q`| Suppress informational messages and progress spinners. |
| `--help` | `-h` | Show help information. |

## Contributing

Contributions and issues are welcome – please see the [issues page](https://github.com/chriscorrea/sift/issues).

## License

This project is licensed under the [BSD-3 License](LICENSE).

## Roadmap
- [x] Content fetching from multiple sources
- [x] CSS selector support
- [x] Multiple output formats (Markdown, text, JSON)
- [x] Text search with BM25 field-aware text ranking
- [ ] Content deduplication across sources
- [ ] Recursive chunking
- [ ] Streaming content processing for arbitrarily large files
- [ ] Additional tokenizer support beyon cl100k_base 
- [ ] Improve smart content extraction through zero-shot classification or other NLP approaches
- [ ] Semantic search via local ONNX embedding model (under evaluation)
