// Package helpers provides shared CLI utility functions.
package helpers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/salmonumbrella/threads-go/internal/iocontext"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"golang.org/x/term"
)

// isTerminal checks if stdin is a terminal (mockable for tests)
var isTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// ConfirmOrYes prompts for confirmation unless --yes flag is set.
// Returns true if confirmed, false otherwise.
// Returns error if stdin is not a terminal and --yes is not set.
func ConfirmOrYes(ctx context.Context, prompt string) (bool, error) {
	// Skip confirmation if --yes flag is set
	if outfmt.GetYes(ctx) {
		return true, nil
	}

	// Skip confirmation in JSON mode (scripts expect non-interactive)
	if outfmt.IsJSON(ctx) {
		return true, nil
	}

	// Require terminal for interactive confirmation
	if !isTerminal() {
		return false, fmt.Errorf("stdin is not a terminal; use --yes to confirm non-interactively")
	}

	// Get IO from context for testability
	io := iocontext.GetIO(ctx)

	// Write prompt to stderr
	fmt.Fprintf(io.ErrOut, "%s [y/N]: ", prompt)

	// Read response from stdin
	reader := bufio.NewReader(io.In)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	// Normalize response
	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes", nil
}

// FlagMarker is the interface for commands that can mark flags as required.
type FlagMarker interface {
	MarkFlagRequired(name string) error
}

// MustMarkRequired marks a flag as required, panicking if the flag doesn't exist.
// This is appropriate because a missing flag is a programmer error, not a runtime error.
func MustMarkRequired(cmd FlagMarker, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Sprintf("failed to mark flag %q as required: %v", name, err))
	}
}
