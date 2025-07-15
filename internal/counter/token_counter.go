package counter

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// TokenCounter implements token counting using tiktoken with cl100k_base encoding.
type TokenCounter struct {
	encoding *tiktoken.Tiktoken
	mu       sync.RWMutex // protects encoding access for thread safety
}

// NewTokenCounter creates a new TokenCounter with cl100k_base encoding
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
// This method is thread-safe and can be called concurrently
func (tc *TokenCounter) Count(text string) int {
	if text == "" {
		return 0
	}

	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// encode text to tokens (nil parameters mean no special tokens allowed/disallowed)
	tokens := tc.encoding.Encode(text, nil, nil)
	tokenCount := len(tokens)

	slog.Debug("Token count calculated", "textLength", len(text), "tokenCount", tokenCount)
	return tokenCount
}

// Name returns the name of this counting method (for logging and debugging).
func (tc *TokenCounter) Name() string {
	return "tokens (cl100k_base)"
}
