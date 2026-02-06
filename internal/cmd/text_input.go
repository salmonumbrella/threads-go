package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
)

// maxTextInputSize is the maximum number of bytes accepted from a file or stdin.
// Threads posts are limited to 500 characters, but we allow a generous 1 MiB to
// accommodate multi-byte encodings and let the API enforce the real limit.
const maxTextInputSize = 1 << 20 // 1 MiB

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

	var b []byte
	switch s {
	case "-":
		ioctx := iocontext.GetIO(ctx)
		var r io.Reader
		if ioctx != nil && ioctx.In != nil {
			r = ioctx.In
		} else {
			r = os.Stdin
		}
		var err error
		b, err = io.ReadAll(io.LimitReader(r, maxTextInputSize+1))
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
	default:
		var err error
		b, err = os.ReadFile(s) //nolint:gosec // Reading a user-supplied file path is intentional for CLI automation.
		if err != nil {
			return "", &UserFriendlyError{
				Message:    fmt.Sprintf("Failed to read file: %s", s),
				Suggestion: "Check the path or use --text-file - to read from stdin",
			}
		}
	}

	if len(b) > maxTextInputSize {
		return "", &UserFriendlyError{
			Message:    fmt.Sprintf("Input too large (%d bytes, max %d)", len(b), maxTextInputSize),
			Suggestion: "Threads posts are limited to 500 characters. Trim your input and try again",
		}
	}

	return strings.TrimRight(string(b), "\n"), nil
}
