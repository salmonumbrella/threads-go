package cmd

import (
	"fmt"
	"net/url"
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

	// Pasted URLs (permalinks): try to extract a post/reply/user ID.
	// We only attempt URL extraction when an expected kind is provided.
	if expectedKind != "" && strings.Contains(s, "://") {
		id, kind, ok := extractIDFromURL(s)
		if ok {
			// Map URL kind to our expected kinds.
			if kind != "" && kind != expectedKind {
				return "", fmt.Errorf("invalid %s ID: URL is for %s, expected %s", expectedKind, kind, expectedKind)
			}
			return id, nil
		}
		return "", fmt.Errorf("invalid %s ID: could not extract ID from URL", expectedKind)
	}

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
	case "location", "loc", "l":
		return "location"
	}
	return ""
}

// extractIDFromURL extracts a resource ID from a pasted URL.
// It supports common Threads-like patterns:
// - https://.../t/<id>
// - https://.../@<user>/post/<id>
// - https://.../post/<id>
//
// It returns (id, kind, ok). kind is currently "post" (and may be extended later).
func extractIDFromURL(raw string) (string, string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", "", false
	}

	// Query param fallbacks – map param key to the appropriate kind.
	q := u.Query()
	for _, qp := range []struct {
		key  string
		kind string
	}{
		{"post_id", "post"},
		{"reply_id", "reply"},
		{"id", ""}, // generic – no kind assertion
	} {
		if v := strings.TrimSpace(q.Get(qp.key)); v != "" {
			return v, qp.kind, true
		}
	}

	// Path scanning
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "", "", false
	}
	segs := strings.Split(path, "/")

	// Common: /t/<id>
	for i := 0; i < len(segs)-1; i++ {
		if segs[i] == "t" || segs[i] == "post" || segs[i] == "p" {
			id := strings.TrimSpace(segs[i+1])
			if id != "" {
				return id, "post", true
			}
		}
	}

	// Threads profile style: /@user/post/<id>
	for i := 0; i < len(segs)-2; i++ {
		if strings.HasPrefix(segs[i], "@") && (segs[i+1] == "post" || segs[i+1] == "p") {
			id := strings.TrimSpace(segs[i+2])
			if id != "" {
				return id, "post", true
			}
		}
	}

	return "", "", false
}

// extractUsernameFromURL extracts a Threads username from a profile URL like:
// - https://www.threads.net/@username
// - https://www.threads.net/@username/post/<id>
func extractUsernameFromURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", false
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "", false
	}
	segs := strings.Split(path, "/")
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if strings.HasPrefix(seg, "@") && len(seg) > 1 {
			return strings.TrimPrefix(seg, "@"), true
		}
	}
	return "", false
}
