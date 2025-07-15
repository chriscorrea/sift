package spinner

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewSpinner(t *testing.T) {
	var buf bytes.Buffer
	message := "Loading..."

	spinner := New(context.Background(), &buf, message)

	if spinner == nil {
		t.Fatal("New() returned nil")
	}

	if spinner.message != message {
		t.Errorf("Expected message %q, got %q", message, spinner.message)
	}

	if len(spinner.frames) != 6 {
		t.Errorf("Expected 6 frames, got %d", len(spinner.frames))
	}

	expectedFrames := []string{"◜", "◠", "◝", "◞", "◡", "◟"}
	for i, frame := range spinner.frames {
		if frame != expectedFrames[i] {
			t.Errorf("Expected frame %d to be %q, got %q", i, expectedFrames[i], frame)
		}
	}
}

func TestSpinnerStartStop(t *testing.T) {
	var buf bytes.Buffer
	spinner := New(context.Background(), &buf, "Testing...")

	// initially not active
	if spinner.IsActive() {
		t.Error("Spinner should not be active initially")
	}

	// start spinner
	spinner.Start()

	if !spinner.IsActive() {
		t.Error("Spinner should be active after Start()")
	}

	// allow some time for spinner to run
	time.Sleep(150 * time.Millisecond)

	// stop spinner
	spinner.Stop()

	if spinner.IsActive() {
		t.Error("Spinner should not be active after Stop()")
	}

	// check that we are writing something to the buffer
	if buf.Len() == 0 {
		t.Error("Expected output to be written to buffer")
	}

	// check that the buffer contains spinner frames
	output := buf.String()
	hasSpinnerFrame := false
	for _, frame := range []string{"◜", "◠", "◝", "◞", "◡", "◟"} {
		if strings.Contains(output, frame) {
			hasSpinnerFrame = true
			break
		}
	}

	if !hasSpinnerFrame {
		t.Error("Expected spinner frames in output")
	}
}

func TestSpinnerUpdateMessage(t *testing.T) {
	var buf bytes.Buffer
	spinner := New(context.Background(), &buf, "Initial message")

	newMessage := "Updated message"
	spinner.UpdateMessage(newMessage)

	if spinner.message != newMessage {
		t.Errorf("Expected message %q, got %q", newMessage, spinner.message)
	}
}

func TestSpinnerDoubleStart(t *testing.T) {
	var buf bytes.Buffer
	spinner := New(context.Background(), &buf, "Testing...")

	// start the spinner
	spinner.Start()

	if !spinner.IsActive() {
		t.Error("Spinner should be active after first Start()")
	}

	// start it again; should not cause any issues
	spinner.Start()

	if !spinner.IsActive() {
		t.Error("Spinner should still be active after second Start()")
	}

	// clean up
	spinner.Stop()
}

func TestSpinnerDoubleStop(t *testing.T) {
	var buf bytes.Buffer
	spinner := New(context.Background(), &buf, "Testing...")

	// start and stop spinner
	spinner.Start()
	spinner.Stop()

	if spinner.IsActive() {
		t.Error("Spinner should not be active after Stop()")
	}

	// stop again - should not cause issues
	spinner.Stop()

	if spinner.IsActive() {
		t.Error("Spinner should still not be active after second Stop()")
	}
}

func TestSpinnerStopWithoutStart(t *testing.T) {
	var buf bytes.Buffer
	spinner := New(context.Background(), &buf, "Testing...")

	// stop without starting - should not cause issues
	spinner.Stop()

	if spinner.IsActive() {
		t.Error("Spinner should not be active after Stop() without Start()")
	}
}

func TestSpinnerOutput(t *testing.T) {
	var buf bytes.Buffer
	spinner := New(context.Background(), &buf, "Processing...")

	spinner.Start()

	// let this run for a bit
	time.Sleep(333 * time.Millisecond)

	spinner.Stop()

	output := buf.String()

	// check that the message appears in the output
	if !strings.Contains(output, "Processing...") {
		t.Error("Expected message to appear in output")
	}

	// check that the output ends with carriage return (for non-terminal output)
	if !strings.HasSuffix(output, "\r") {
		t.Error("Expected output to end with carriage return")
	}
}
