// Package counter provides text counting functionality for the sift CLI tool.
//
// This package implements various text counting strategies including token counting
// (using OpenAI's tiktoken), word counting, and character counting. The default
// strategy uses token counting with the cl100k_base encoding, which is compatible
// with OpenAI's GPT models.
//
// Usage Example:
//
//	counter := counter.NewTokenCounter()
//	count := counter.Count("Hello, world!")
//	// Returns the number of tokens in the text
//
// The package supports multiple counting methods through the Counter interface,
// making it easy to switch between different counting strategies based on user
// preferences or specific requirements.
package counter

// Counter defines the interface for different text counting strategies.
type Counter interface {
	// Count returns the number of units (tokens, words, or characters) in given text.
	Count(text string) int

	// Name returns a human-readable name for this counting method (for logging)
	Name() string
}

// CountingMethod represents the different available counting strategies.
type CountingMethod int

const (
	// Tokens uses tiktoken with cl100k_base encoding (default)
	Tokens CountingMethod = iota
	// Words counts words using whitespace splitting
	Words
	// Characters counts individual characters including whitespace
	Characters
)

// String returns the string representation of the counting method.
func (cm CountingMethod) String() string {
	switch cm {
	case Tokens:
		return "tokens"
	case Words:
		return "words"
	case Characters:
		return "characters"
	default:
		return "unknown"
	}
}

// NewCounter creates a new Counter instance based on the specified method.
// This functions as a factory; it returns concrete Counter types,
// providing a single, simple entry point for to get a counter instance.
// Returns an error if the counter cannot be initialized (e.g., tiktoken encoding fails).
func NewCounter(method CountingMethod) (Counter, error) {
	switch method {
	case Tokens:
		return NewTokenCounter()
	case Words:
		return NewWordCounter(), nil
	case Characters:
		return NewCharCounter(), nil
	default:
		return NewTokenCounter() // fallback to default
	}
}
