// Package cmd provides CLI command implementations with user-friendly error handling.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// UserFriendlyError wraps an error with a user-friendly message and optional suggestion.
type UserFriendlyError struct {
	Message    string
	Suggestion string
	Cause      error
}

func (e *UserFriendlyError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s\n\nSuggestion: %s", e.Message, e.Suggestion)
	}
	return e.Message
}

func (e *UserFriendlyError) Unwrap() error {
	return e.Cause
}

type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`

	// Kind is a stable, coarse classification that agents can use for branching.
	Kind string `json:"kind,omitempty"` // auth|rate_limit|validation|network|api|unknown

	// API details, when available.
	Code      int    `json:"code,omitempty"`
	Type      string `json:"type,omitempty"`
	Field     string `json:"field,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Temporary bool   `json:"temporary,omitempty"`

	// RetryAfterSeconds is set for rate limit errors when present.
	RetryAfterSeconds int64 `json:"retry_after_seconds,omitempty"`
}

// WriteErrorTo writes a formatted error to w.
//
// In JSON output mode, this emits a structured JSON error object to avoid
// contaminating stdout pipelines with plain-text diagnostics.
func WriteErrorTo(ctx context.Context, w io.Writer, err error) {
	if err == nil {
		return
	}

	formatted := FormatError(err)
	if !outfmt.IsJSON(ctx) {
		fmt.Fprintln(w, formatted.Error()) //nolint:errcheck // Best-effort output
		return
	}

	_ = writeErrorJSONTo(ctx, w, formatted) // Best-effort output in error paths.
}

func writeErrorJSONTo(ctx context.Context, w io.Writer, formatted error) error {
	root := formatted
	var uf *UserFriendlyError
	if errors.As(formatted, &uf) {
		if uf.Cause != nil {
			root = uf.Cause
		}
	}

	payload := errorPayload{
		Message: formatted.Error(),
	}
	if uf != nil {
		payload.Message = uf.Message
		payload.Suggestion = uf.Suggestion
	}

	// Enrich with typed API errors if we can.
	var authErr *api.AuthenticationError
	var rateErr *api.RateLimitError
	var valErr *api.ValidationError
	var netErr *api.NetworkError
	var apiErr *api.APIError

	switch {
	case errors.As(root, &authErr):
		payload.Kind = "auth"
		payload.Code = authErr.Code
		payload.Type = authErr.Type
	case errors.As(root, &rateErr):
		payload.Kind = "rate_limit"
		payload.Code = rateErr.Code
		payload.Type = rateErr.Type
		payload.RetryAfterSeconds = int64(rateErr.RetryAfter.Seconds())
	case errors.As(root, &valErr):
		payload.Kind = "validation"
		payload.Code = valErr.Code
		payload.Type = valErr.Type
		payload.Field = valErr.Field
	case errors.As(root, &netErr):
		payload.Kind = "network"
		payload.Code = netErr.Code
		payload.Type = netErr.Type
		payload.Temporary = netErr.Temporary
	case errors.As(root, &apiErr):
		payload.Kind = "api"
		payload.Code = apiErr.Code
		payload.Type = apiErr.Type
		payload.RequestID = apiErr.RequestID
	default:
		payload.Kind = "unknown"
	}

	enc := json.NewEncoder(w)
	if !outfmt.IsJSONL(ctx) {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(errorEnvelope{Error: payload})
}

// FormatError converts API errors and common errors into user-friendly messages
// with actionable suggestions. This should be called on errors before returning
// them to the user.
func FormatError(err error) error {
	if err == nil {
		return nil
	}

	// Preserve already formatted errors.
	var ufErr *UserFriendlyError
	if errors.As(err, &ufErr) {
		return ufErr
	}

	// Check for authentication errors
	var authErr *api.AuthenticationError
	if errors.As(err, &authErr) {
		return formatAuthError(authErr)
	}

	// Check for rate limit errors
	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return formatRateLimitError(rateLimitErr)
	}

	// Check for validation errors
	var validationErr *api.ValidationError
	if errors.As(err, &validationErr) {
		return formatValidationError(validationErr)
	}

	// Check for network errors
	var networkErr *api.NetworkError
	if errors.As(err, &networkErr) {
		return formatNetworkError(networkErr)
	}

	// Check for API errors
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return formatAPIError(apiErr)
	}

	// Check for common error patterns in the message
	errMsg := err.Error()
	return formatGenericError(errMsg, err)
}

func formatAuthError(err *api.AuthenticationError) *UserFriendlyError {
	var msg string
	var suggestion string

	switch {
	case strings.Contains(strings.ToLower(err.Message), "expired"):
		msg = "Your access token has expired"
		suggestion = "Run 'threads auth refresh' to get a new token, or 'threads auth login' to re-authenticate"

	case strings.Contains(strings.ToLower(err.Message), "invalid"):
		msg = "Your access token is invalid"
		suggestion = "Run 'threads auth login' to re-authenticate with Threads"

	case err.Code == 401:
		msg = "Authentication required"
		suggestion = "Run 'threads auth login' to authenticate with Threads"

	case err.Code == 403:
		msg = "Access denied - you may not have permission for this action"
		suggestion = "Check that your token has the required scopes. Run 'threads auth login --scopes <scopes>' to request additional permissions"

	default:
		msg = fmt.Sprintf("Authentication error: %s", err.Message)
		suggestion = "Run 'threads auth status' to check your authentication, or 'threads auth login' to re-authenticate"
	}

	return &UserFriendlyError{
		Message:    msg,
		Suggestion: suggestion,
		Cause:      err,
	}
}

func formatRateLimitError(err *api.RateLimitError) *UserFriendlyError {
	msg := "Rate limit exceeded"
	suggestion := ""

	if err.RetryAfter > 0 {
		msg = fmt.Sprintf("Rate limit exceeded - try again in %s", err.RetryAfter.String())
		suggestion = fmt.Sprintf("Wait %s before making another request. Run 'threads ratelimit status' to check your current limits", err.RetryAfter.String())
	} else {
		suggestion = "Wait a few minutes before retrying. Run 'threads ratelimit status' to check your current limits"
	}

	return &UserFriendlyError{
		Message:    msg,
		Suggestion: suggestion,
		Cause:      err,
	}
}

func formatValidationError(err *api.ValidationError) *UserFriendlyError {
	msg := "Invalid input"
	suggestion := ""

	if err.Field != "" {
		msg = fmt.Sprintf("Invalid value for '%s'", err.Field)
		suggestion = fmt.Sprintf("Check the value provided for '%s' and try again", err.Field)
	} else if err.Message != "" {
		msg = fmt.Sprintf("Validation error: %s", err.Message)
		suggestion = "Check your input and try again. Use --help for usage information"
	} else {
		suggestion = "Check your input values and try again. Use --help for usage information"
	}

	// Extract more specific suggestions from common validation errors
	lowerMsg := strings.ToLower(err.Message)
	switch {
	case strings.Contains(lowerMsg, "text") && strings.Contains(lowerMsg, "long"):
		suggestion = "Post text exceeds the maximum length (500 characters). Shorten your text and try again"
	case strings.Contains(lowerMsg, "url") && strings.Contains(lowerMsg, "invalid"):
		suggestion = "Provide a valid URL (must start with http:// or https://)"
	case strings.Contains(lowerMsg, "media") && strings.Contains(lowerMsg, "format"):
		suggestion = "Use a supported media format (JPEG, PNG for images; MP4 for videos)"
	case strings.Contains(lowerMsg, "carousel") && strings.Contains(lowerMsg, "items"):
		suggestion = "Carousel posts require 2-20 media items"
	}

	return &UserFriendlyError{
		Message:    msg,
		Suggestion: suggestion,
		Cause:      err,
	}
}

func formatNetworkError(err *api.NetworkError) *UserFriendlyError {
	var msg string
	var suggestion string

	switch {
	case strings.Contains(strings.ToLower(err.Message), "timeout"):
		msg = "Request timed out"
		suggestion = "Check your internet connection and try again. The Threads API may be experiencing slowdowns"

	case strings.Contains(strings.ToLower(err.Message), "no such host"),
		strings.Contains(strings.ToLower(err.Message), "dns"):
		msg = "Could not reach the Threads API"
		suggestion = "Check your internet connection and DNS settings"

	case strings.Contains(strings.ToLower(err.Message), "connection refused"):
		msg = "Connection refused by the server"
		suggestion = "The Threads API may be temporarily unavailable. Try again in a few minutes"

	case strings.Contains(strings.ToLower(err.Message), "tls"),
		strings.Contains(strings.ToLower(err.Message), "certificate"):
		msg = "Secure connection failed"
		suggestion = "There may be a problem with your network's SSL/TLS certificates"

	case err.Temporary:
		msg = "Temporary network error"
		suggestion = "This is usually a transient issue. Try again in a moment"

	default:
		msg = fmt.Sprintf("Network error: %s", err.Message)
		suggestion = "Check your internet connection and try again"
	}

	return &UserFriendlyError{
		Message:    msg,
		Suggestion: suggestion,
		Cause:      err,
	}
}

func formatAPIError(err *api.APIError) *UserFriendlyError {
	var msg string
	var suggestion string

	switch {
	case err.Code >= 500 && err.Code < 600:
		msg = "The Threads API is experiencing issues"
		if err.RequestID != "" {
			suggestion = fmt.Sprintf("This is a server-side issue. Try again later. Request ID: %s", err.RequestID)
		} else {
			suggestion = "This is a server-side issue. Try again in a few minutes"
		}

	case strings.Contains(strings.ToLower(err.Message), "not found"):
		msg = "Resource not found"
		suggestion = "Check that the ID you provided is correct and that the resource exists"

	case strings.Contains(strings.ToLower(err.Message), "deleted"):
		msg = "This content has been deleted"
		suggestion = "The post, reply, or user you're looking for no longer exists"

	case strings.Contains(strings.ToLower(err.Message), "private"):
		msg = "This content is private"
		suggestion = "You cannot access private content from other users"

	default:
		msg = fmt.Sprintf("API error: %s", err.Message)
		if err.RequestID != "" {
			suggestion = fmt.Sprintf("If this persists, report the issue with request ID: %s", err.RequestID)
		} else {
			suggestion = "Try again. If the problem persists, check the Threads API status"
		}
	}

	return &UserFriendlyError{
		Message:    msg,
		Suggestion: suggestion,
		Cause:      err,
	}
}

func formatGenericError(errMsg string, originalErr error) error {
	lowerMsg := strings.ToLower(errMsg)

	// Common CLI-level error patterns
	switch {
	case strings.Contains(lowerMsg, "no account configured"),
		strings.Contains(lowerMsg, "account not found"):
		return &UserFriendlyError{
			Message:    "No Threads account configured",
			Suggestion: "Run 'threads auth login' to authenticate with your Threads account",
			Cause:      originalErr,
		}

	case strings.Contains(lowerMsg, "token expired"):
		return &UserFriendlyError{
			Message:    "Your access token has expired",
			Suggestion: "Run 'threads auth refresh' to get a new token, or 'threads auth login' to re-authenticate",
			Cause:      originalErr,
		}

	case strings.Contains(lowerMsg, "client secret not stored"),
		strings.Contains(lowerMsg, "cannot refresh"):
		return &UserFriendlyError{
			Message:    "Cannot refresh token - missing client credentials",
			Suggestion: "Run 'threads auth login' with your client ID and secret to enable token refresh",
			Cause:      originalErr,
		}

	case strings.Contains(lowerMsg, "credential store"),
		strings.Contains(lowerMsg, "keyring"):
		return &UserFriendlyError{
			Message:    "Could not access the credential store",
			Suggestion: "Ensure you have keychain/keyring access. On Linux, you may need to install libsecret",
			Cause:      originalErr,
		}

	case strings.Contains(lowerMsg, "context deadline exceeded"),
		strings.Contains(lowerMsg, "context canceled"):
		return &UserFriendlyError{
			Message:    "Operation timed out or was cancelled",
			Suggestion: "Try again. For large operations, consider using smaller batch sizes",
			Cause:      originalErr,
		}

	case strings.Contains(lowerMsg, "empty response"):
		return &UserFriendlyError{
			Message:    "Received empty response from the API",
			Suggestion: "The API may be experiencing issues. Try again in a moment",
			Cause:      originalErr,
		}

	case strings.Contains(lowerMsg, "json"):
		return &UserFriendlyError{
			Message:    "Failed to parse API response",
			Suggestion: "The API response was malformed. This may be a temporary issue - try again",
			Cause:      originalErr,
		}
	}

	// Return original error if no specific handling
	return originalErr
}

// WrapError wraps an error with context while preserving the ability to format it.
// Use this instead of fmt.Errorf when you want to add context but still get
// user-friendly error formatting.
func WrapError(context string, err error) error {
	if err == nil {
		return nil
	}

	// First format the underlying error
	formatted := FormatError(err)

	// If it's already a user-friendly error, prepend the context
	if ufErr, ok := formatted.(*UserFriendlyError); ok {
		return &UserFriendlyError{
			Message:    fmt.Sprintf("%s: %s", context, ufErr.Message),
			Suggestion: ufErr.Suggestion,
			Cause:      ufErr.Cause,
		}
	}

	// Otherwise wrap normally
	return fmt.Errorf("%s: %w", context, formatted)
}
