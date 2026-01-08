package cmd

import (
	"fmt"
	"strings"
)

// ValidateHTTPSURL validates that a URL uses HTTPS protocol.
// Returns a UserFriendlyError if validation fails.
func ValidateHTTPSURL(url, fieldName string) error {
	if !strings.HasPrefix(url, "https://") {
		return &UserFriendlyError{
			Message:    fmt.Sprintf("%s must use HTTPS", fieldName),
			Suggestion: "Use a URL starting with https://",
		}
	}
	return nil
}
