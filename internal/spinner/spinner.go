// Package spinner provides a simple terminal spinner utility for indicating progress.
package spinner

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// Spinner represents a spinning progress indicator.
type Spinner struct {
	frames  []string
	delay   time.Duration
	writer  io.Writer
	active  bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	message string
	wg      sync.WaitGroup
}

// New creates a new spinner with the specified frames and delay.
// ctx allows for cancellation of the spinner goroutine.
func New(ctx context.Context, writer io.Writer, message string) *Spinner {
	spinnerCtx, cancel := context.WithCancel(ctx)
	return &Spinner{
		frames:  []string{"◜", "◠", "◝", "◞", "◡", "◟"},
		delay:   100 * time.Millisecond,
		writer:  writer,
		message: message,
		ctx:     spinnerCtx,
		cancel:  cancel,
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return // already running
	}

	s.active = true

	s.wg.Add(1)
	go s.run()
}

// Stop stops the spinner animation and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return // not running
	}

	s.active = false
	s.cancel()
	s.mu.Unlock()

	// wait for spinner goroutine to finish
	s.wg.Wait()

	// clear the spinner line with terminal control sequences
	// only clear if we're writing to a terminal (not redirected)
	if f, ok := s.writer.(*os.File); ok && isTerminal(f) {
		fmt.Fprint(s.writer, "\r\033[2K")
	} else {
		// for non-terminal output, just use carriage return
		fmt.Fprint(s.writer, "\r")
	}
}

// IsActive returns whether the spinner is currently running
func (s *Spinner) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// run is the main spinner loop.
func (s *Spinner) run() {
	defer s.wg.Done() // signal completion when goroutine exits

	frameIndex := 0
	ticker := time.NewTicker(s.delay)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			frame := s.frames[frameIndex%len(s.frames)]
			message := s.message
			s.mu.RUnlock()

			fmt.Fprintf(s.writer, "\r%s %s", frame, message)
			frameIndex++
		}
	}
}

// isTerminal helper function checks if is a terminal
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}
