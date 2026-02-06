package cmd

import (
	"fmt"
	"strings"
)

// normalizeIDArg accepts common agent shorthands:
// - "#123" means "123"
// - "<kind>:123" means "123" when kind matches expectedKind (e.g. "post:123")
//
// expectedKind is used for validation; when set, mismatched known prefixes error.
func normalizeIDArg(input string, expectedKind string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		if expectedKind == "" {
			return "", fmt.Errorf("missing ID")
		}
		return "", fmt.Errorf("missing %s ID", expectedKind)
	}

	// Common shorthand: "#123" means "123".
	s = strings.TrimPrefix(s, "#")

	if expectedKind != "" {
		if prefix, rest, ok := strings.Cut(s, ":"); ok {
			prefix = strings.ToLower(strings.TrimSpace(prefix))
			rest = strings.TrimSpace(rest)
			if rest == "" {
				return "", fmt.Errorf("invalid %s ID %q: missing value after ':'", expectedKind, input)
			}

			normalized := normalizeIDPrefix(prefix)
			if normalized != "" {
				if normalized != expectedKind {
					return "", fmt.Errorf("invalid %s ID: got %s:%s", expectedKind, normalized, rest)
				}
				s = rest
			}
		}
	}

	if s == "" {
		if expectedKind == "" {
			return "", fmt.Errorf("missing ID")
		}
		return "", fmt.Errorf("missing %s ID", expectedKind)
	}

	return s, nil
}

func normalizeIDPrefix(prefix string) string {
	switch prefix {
	case "post", "posts", "p":
		return "post"
	case "reply", "replies", "r":
		return "reply"
	case "user", "users", "u":
		return "user"
	}
	return ""
}
