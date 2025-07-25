package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/chriscorrea/sift/internal/app"
	"github.com/chriscorrea/sift/internal/counter"

	"github.com/spf13/cobra"
)

// buildConfig constructs an app.Config from command flags and arguments
func buildConfig(cmd *cobra.Command, args []string) (app.Config, error) {
	// get flag values
	selector, _ := cmd.Flags().GetString("selector")
	tokenLimit, _ := cmd.Flags().GetInt("token-limit")
	wordLimit, _ := cmd.Flags().GetInt("word-limit")
	charLimit, _ := cmd.Flags().GetInt("character-limit")
	search, _ := cmd.Flags().GetString("search")
	mdFlag, _ := cmd.Flags().GetBool("md")
	textFlag, _ := cmd.Flags().GetBool("text")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	beginningFlag, _ := cmd.Flags().GetBool("beginning")
	middleFlag, _ := cmd.Flags().GetBool("middle")
	endFlag, _ := cmd.Flags().GetBool("end")
	quiet, _ := cmd.Flags().GetBool("quiet")
	debug, _ := cmd.Flags().GetBool("debug")
	includeAll, _ := cmd.Flags().GetBool("include-all")

	//TODO: configurable http timeout, ...

	// determine counting method and max units
	var countingMethod counter.CountingMethod
	var maxUnits int
	switch {
	case tokenLimit > 0:
		countingMethod = counter.Tokens
		maxUnits = tokenLimit
	case wordLimit > 0:
		countingMethod = counter.Words
		maxUnits = wordLimit
	case charLimit > 0:
		countingMethod = counter.Characters
		maxUnits = charLimit
	default:
		// only apply default limits when no search query is specified
		// search queries should return only relevant results, not fill to a limit
		if search == "" {
			// default to 2500 tokens for non-search scenarios
			maxUnits = 2500
			countingMethod = counter.Tokens
		} else {
			// search without explicit limits: no size constraint
			maxUnits = 0
			countingMethod = counter.Tokens // still need a counting method for chunking
		}
	}

	// determine output format
	var outputFormat app.OutputFormat
	switch {
	case textFlag:
		outputFormat = app.Text
	case jsonFlag:
		outputFormat = app.JSON
	case mdFlag:
		outputFormat = app.Markdown
	default:
		outputFormat = app.Markdown // default if no format flag
	}

	// determine sizing strategy
	var sizingStrategy app.SizingStrategy
	switch {
	case middleFlag:
		sizingStrategy = app.Middle
	case endFlag:
		sizingStrategy = app.End
	case beginningFlag:
		sizingStrategy = app.Beginning
	default:
		sizingStrategy = app.Beginning // default when no flag
	}

	// use positional arguments as sources with smart detection
	var sources []string
	if len(args) == 0 {
		// no arguments provided - use stdin
		sources = append(sources, "-")
	} else {
		// use provided arguments as sources (auto-detection handled in fetch.go)
		sources = args
	}

	// get smart context configuration
	contextTokens, _ := cmd.Flags().GetInt("context-tokens")

	// use smart context if context-tokens flag is present (even without value, which uses default)
	useSmartContext := cmd.Flags().Changed("context-tokens")

	// apply default of 200 tokens if flag is present but no value specified
	if useSmartContext && contextTokens == 0 {
		contextTokens = 200
	}

	// return constructed config
	return app.Config{
		Sources:         sources,
		Selector:        selector,
		MaxUnits:        maxUnits,
		CountingMethod:  countingMethod,
		SizingStrategy:  sizingStrategy,
		SearchQuery:     search,
		OutputFormat:    outputFormat,
		ContextBefore:   1, // default: 1 chunk before search results
		ContextAfter:    2, // default: 2 chunks after search results
		ContextUnits:    contextTokens,
		UseSmartContext: useSmartContext,
		Quiet:           quiet,
		Debug:           debug,
		IncludeAll:      includeAll,
	}, nil
}

// setupLogger configures the default slog logger based on debug mode
func setupLogger(debug bool) {
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else {
		level = slog.LevelError
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

var rootCmd = &cobra.Command{
	Use:   "sift [sources...]",
	Short: "A CLI tool for text content extraction",
	Long: `Sift is a command-line tool that extracts clean, structured text from messy sources. Sources may include URLs, local files, or standard input.

Examples:
  sift https://example.com
  sift file.txt document.html
  cat content.txt | sift`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// build config from flags and arguments
		config, err := buildConfig(cmd, args)
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		// configure logging pending debug flag
		setupLogger(config.Debug)

		// create context with signal handling for graceful shutdown
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		// run the app!
		result, err := app.Run(ctx, config)
		if err != nil {
			return fmt.Errorf("sift failed: %w", err)
		}

		// Output the result
		fmt.Print(result)

		return nil
	},
}

func init() {
	rootCmd.Flags().StringP("selector", "s", "", "CSS selector or extraction pattern")

	// limit flags
	rootCmd.Flags().IntP("token-limit", "t", 0, "Limit output to number of tokens (default: 1000)")
	rootCmd.Flags().IntP("word-limit", "w", 0, "Limit output to number of words")
	rootCmd.Flags().IntP("character-limit", "c", 0, "Limit output to number of characters")

	// limit flags are mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("token-limit", "word-limit", "character-limit")

	// search functionality
	rootCmd.Flags().String("search", "", "Search for keyword(s)")
	rootCmd.Flags().Int("context-tokens", 0, "Set token budget for smart context around search results (default: 200 when flag is used)")

	// output format flags (see also 'configure mutually exclusive flag groups' below)
	rootCmd.Flags().Bool("md", false, "Output in Markdown format (default)")
	rootCmd.Flags().Bool("text", false, "Output in plain text format")
	rootCmd.Flags().Bool("json", false, "Output in JSON format")

	// output format flags are mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("md", "text", "json")

	// sizing strategy flags (see also 'configure mutually exclusive flag groups' below)
	rootCmd.Flags().Bool("beginning", false, "Apply size constraints from the beginning of the document (default)")
	rootCmd.Flags().Bool("middle", false, "Apply size constraints from the middle of the document, expanding outward")
	rootCmd.Flags().Bool("end", false, "Apply size constraints from the end of the document, working backward")

	// sizing strategy flags are mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("beginning", "middle", "end")

	// other flags
	rootCmd.Flags().BoolP("quiet", "q", false, "Suppress output messages")
	rootCmd.Flags().BoolP("debug", "D", false, "Enable debug logging")
	_ = rootCmd.Flags().MarkHidden("debug")
	rootCmd.Flags().BoolP("include-all", "i", false, "Include all content without readability filtering")

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
