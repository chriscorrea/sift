package counter

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// TokenCounter implements token counting using tiktoken w/ cl100k_base encoding.
type TokenCounter struct {
	encoding *tiktoken.Tiktoken
	mu       sync.RWMutex // protects encoding access for thread safety
}

// NewTokenCounter creates a new TokenCounter w/ cl100k_base encoding
func NewTokenCounter() (Counter, error) {
	slog.Debug("Initializing TokenCounter with cl100k_base encoding")

	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cl100k_base encoding: %w", err)
	}

	return &TokenCounter{
		encoding: encoding,
	}, nil
}

// Count returns the number of tokens in the given text using cl100k_base encoding.
// This can be called concurrently
func (tc *TokenCounter) Count(text string) int {
	if text == "" {
		return 0
	}

	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// encode text to tokens (nil params mean no special tokens allowed/disallowed)
	tokens := tc.encoding.Encode(text, nil, nil)
	tokenCount := len(tokens)

	slog.Debug("Token count calculated", "textLength", len(text), "tokenCount", tokenCount)
	return tokenCount
}

// Name returns the name of this counting method (for logging and debugging).
func (tc *TokenCounter) Name() string {
	return "tokens (cl100k_base)"
}

// CreatePartialText creates a partial text up to the specified token limit.
func (tc *TokenCounter) CreatePartialText(text string, maxTokens int) string {
	if maxTokens <= 0 || text == "" {
		return ""
	}

	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// encode text to tokens
	tokens := tc.encoding.Encode(text, nil, nil)

	// if text is already within limit, return as-is
	if len(tokens) <= maxTokens {
		return text
	}

	// slice tokens to desired count
	truncatedTokens := tokens[:maxTokens]

	// decode back to text
	partialText := tc.encoding.Decode(truncatedTokens)

	slog.Debug("Created partial text", "originalTokens", len(tokens), "maxTokens", maxTokens, "resultTokens", len(truncatedTokens))
	return partialText
}
