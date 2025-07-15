# 🧑‍🍳 Sift

**A prep tool for your text-based recipes**

Sift is a command-line tool that extracts clean, structured text from messy sources. Feed it URLs, text files, or stdin, and `sift` automatically extracts core content.

 `sift` is a composable, pipeline-native tool designed to preprare content for large language model workflows or any pipeline requiring clean text. Use keyword search, CSS selectors, and flexible sizing strategies to refine results with precision.

## ✨ Highlights

- **Smart Content Extraction:** Automatically strips HTML tags and isolates the main content from any source using Mozilla's Readability algorithm. You can also target specific content with keyword search or CSS selectors.

- **Flexible Inputs and Outputs:** Processes content from URLs, local files, or standard input with automatic source detection, and outputs to Markdown, plain text, or JSON.

- **Precise Output Sizing:** Controls output size with token-level precision (using cl100k_base for LLM workflows), or by word and character counts.

- **Composable by Design:** Built as a native command-line tool, so you can easily pipe content in and chain with other tools to create powerful text processing workflows.

## Installation

### From Release Binaries

You can download a pre-compiled binary for your operating system from [latest releases](https://github.com/chriscorrea/sift/releases).

### Go Install
If you have a Go environment set up, you can install `sift` directly:
```bash
go install github.com/your-username/sift/cmd/sift@latest
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
sift https://www.marcuse.org/herbert/pubs/64onedim/odmintro.html --search "technology" -t 200
```


## Usage

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--selector` | `-s` | `string` | `""` | CSS selector for content extraction |
| `--search` | | `string` | `""` | Extract content via keyword search |
| `--max-tokens` | `-t` | `int` | `1000` | Maximum number of tokens for output (default method) |
| `--max-words` | `-w` | `int` | `0` | Maximum number of words for output |
| `--max-chars` | `-c` | `int` | `0` | Maximum number of characters for output |
| `--md` | | `bool` | `false` | Output in Markdown format (default) |
| `--text` | | `bool` | `false` | Output in plain text format |
| `--json` | | `bool` | `false` | Output in JSON format with chunks and links |
| `--beginning` | | `bool` | `false` | Apply size constraints from document beginning (default) |
| `--middle` | | `bool` | `false` | Apply size constraints from document middle, expanding outward |
| `--end` | | `bool` | `false` | Apply size constraints from document end, working backward |
| `--include-all` | `-i` | `bool` | `false` | Include all text without any filtering |
| `--quiet` | `-q` | `bool` | `false` | Suppress output messages |
| `--help` | `-h` | | | Show help information |


## Contributing

Contributions and issues are welcome – please see the [issues page](https://github.com/chriscorrea/sift/issues).

## License

This project is licensed under the [BSD-3 License](LICENSE).

## Roadmap
- [x] Content fetching from multiple sources
- [x] CSS selector support
- [x] Multiple output formats (Markdown, text, JSON)
- [ ] Content deduplication across sources
- [ ] Recursive chunking
- [ ] Streaming content processing for arbitrarily large files
- [ ] Additional tokenizer support beyon cl100k_base 
- [ ] Improve smart content extraction through zero-shot classification or other NLP approaches
- [ ] Semantic search via local ONNX embedding model (under evaluation)