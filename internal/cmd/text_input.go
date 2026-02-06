package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
)

// readTextFileOrStdin reads a potentially large text payload from either a file path
// or stdin. This is intentionally separate from --text to avoid breaking @mentions
// (e.g. "--text @alice" must remain literal text, not a file reference).
//
// Supported specs:
//   - "path/to/file.txt"
//   - "@path/to/file.txt" (optional convenience prefix)
//   - "-" or "@-" to read from stdin
func readTextFileOrStdin(ctx context.Context, spec string) (string, error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return "", &UserFriendlyError{
			Message:    "No --text-file provided",
			Suggestion: "Provide a file path, or use --text-file - to read from stdin",
		}
	}

	if strings.HasPrefix(s, "@") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "@"))
	}

	var r io.Reader
	switch s {
	case "-":
		ioctx := iocontext.GetIO(ctx)
		if ioctx != nil && ioctx.In != nil {
			r = ioctx.In
		} else {
			r = os.Stdin
		}
	default:
		b, err := os.ReadFile(s) //nolint:gosec // Reading a user-supplied file path is intentional for CLI automation.
		if err != nil {
			return "", &UserFriendlyError{
				Message:    fmt.Sprintf("Failed to read file: %s", s),
				Suggestion: "Check the path or use --text-file - to read from stdin",
			}
		}
		return string(b), nil
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	return string(b), nil
}
