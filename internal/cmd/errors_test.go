package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

func TestUserFriendlyError_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        *UserFriendlyError
		wantMsg    string
		wantHasSug bool
	}{
		{
			name: "with suggestion",
			err: &UserFriendlyError{
				Message:    "Something went wrong",
				Suggestion: "Try again later",
			},
			wantMsg:    "Something went wrong",
			wantHasSug: true,
		},
		{
			name: "without suggestion",
			err: &UserFriendlyError{
				Message: "Something went wrong",
			},
			wantMsg:    "Something went wrong",
			wantHasSug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if !strings.Contains(got, tt.wantMsg) {
				t.Errorf("Error() = %v, want to contain %v", got, tt.wantMsg)
			}
			hasSuggestion := strings.Contains(got, "Suggestion:")
			if hasSuggestion != tt.wantHasSug {
				t.Errorf("Error() has suggestion = %v, want %v", hasSuggestion, tt.wantHasSug)
			}
		})
	}
}

func TestUserFriendlyError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &UserFriendlyError{
		Message: "Wrapper",
		Cause:   cause,
	}

	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
}

func TestFormatError_AuthenticationError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
		wantSugg   string
	}{
		{
			name:       "expired token",
			err:        api.NewAuthenticationError(401, "Token has expired", ""),
			wantSubstr: "expired",
			wantSugg:   "threads auth refresh",
		},
		{
			name:       "invalid token",
			err:        api.NewAuthenticationError(401, "Invalid access token", ""),
			wantSubstr: "invalid",
			wantSugg:   "threads auth login",
		},
		{
			name:       "401 error",
			err:        api.NewAuthenticationError(401, "Authentication required", ""),
			wantSubstr: "Authentication required",
			wantSugg:   "threads auth login",
		},
		{
			name:       "403 error",
			err:        api.NewAuthenticationError(403, "Access denied", ""),
			wantSubstr: "permission",
			wantSugg:   "scopes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			ufErr, ok := formatted.(*UserFriendlyError)
			if !ok {
				t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
			}
			errStr := ufErr.Error()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
			if !strings.Contains(errStr, tt.wantSugg) {
				t.Errorf("Error() = %v, want suggestion to contain %v", errStr, tt.wantSugg)
			}
		})
	}
}

func TestFormatError_RateLimitError(t *testing.T) {
	err := api.NewRateLimitError(429, "Too many requests", "", 5*time.Minute)
	formatted := FormatError(err)

	ufErr, ok := formatted.(*UserFriendlyError)
	if !ok {
		t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
	}

	errStr := ufErr.Error()
	if !strings.Contains(errStr, "Rate limit") {
		t.Errorf("Error() = %v, want to contain 'Rate limit'", errStr)
	}
	if !strings.Contains(errStr, "threads ratelimit status") {
		t.Errorf("Error() = %v, want suggestion to contain 'threads ratelimit status'", errStr)
	}
}

func TestFormatError_ValidationError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "with field",
			err:        api.NewValidationError(400, "Invalid value", "", "text"),
			wantSubstr: "text",
		},
		{
			name:       "without field",
			err:        api.NewValidationError(400, "Validation failed", "", ""),
			wantSubstr: "Validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			ufErr, ok := formatted.(*UserFriendlyError)
			if !ok {
				t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
			}
			errStr := ufErr.Error()
			if !strings.Contains(errStr, tt.wantSubstr) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestFormatError_NetworkError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "timeout",
			err:        api.NewNetworkError(0, "Request timeout", "", true),
			wantSubstr: "timed out",
		},
		{
			name:       "dns error",
			err:        api.NewNetworkError(0, "no such host", "", false),
			wantSubstr: "DNS",
		},
		{
			name:       "temporary error",
			err:        api.NewNetworkError(0, "Temporary failure", "", true),
			wantSubstr: "transient",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			ufErr, ok := formatted.(*UserFriendlyError)
			if !ok {
				t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
			}
			errStr := ufErr.Error()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestFormatError_APIError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "server error",
			err:        api.NewAPIError(500, "Internal server error", "", "req-123"),
			wantSubstr: "server-side",
		},
		{
			name:       "not found",
			err:        api.NewAPIError(404, "Resource not found", "", ""),
			wantSubstr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			ufErr, ok := formatted.(*UserFriendlyError)
			if !ok {
				t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
			}
			errStr := ufErr.Error()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestFormatError_GenericErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "no account configured",
			err:        errors.New("no account configured"),
			wantSubstr: "threads auth login",
		},
		{
			name:       "token expired",
			err:        errors.New("token expired"),
			wantSubstr: "threads auth refresh",
		},
		{
			name:       "empty response",
			err:        errors.New("empty response from API"),
			wantSubstr: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			ufErr, ok := formatted.(*UserFriendlyError)
			if !ok {
				// Some generic errors may not be converted
				return
			}
			errStr := ufErr.Error()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestFormatError_Nil(t *testing.T) {
	if FormatError(nil) != nil {
		t.Error("FormatError(nil) should return nil")
	}
}

func TestWrapError(t *testing.T) {
	authErr := api.NewAuthenticationError(401, "Token expired", "")
	wrapped := WrapError("API call failed", authErr)

	ufErr, ok := wrapped.(*UserFriendlyError)
	if !ok {
		t.Fatalf("WrapError() did not return *UserFriendlyError, got %T", wrapped)
	}

	errStr := ufErr.Error()
	if !strings.Contains(errStr, "API call failed") {
		t.Errorf("Error() = %v, want to contain context 'API call failed'", errStr)
	}
	if !strings.Contains(errStr, "expired") {
		t.Errorf("Error() = %v, want to contain original error info", errStr)
	}
}

func TestWrapError_Nil(t *testing.T) {
	if WrapError("context", nil) != nil {
		t.Error("WrapError(context, nil) should return nil")
	}
}

func TestWrapError_PlainError(t *testing.T) {
	plainErr := errors.New("something went wrong")
	wrapped := WrapError("operation failed", plainErr)

	if wrapped == nil {
		t.Fatal("WrapError() returned nil for plain error")
	}

	errStr := wrapped.Error()
	if !strings.Contains(errStr, "operation failed") {
		t.Errorf("Error() = %v, want to contain context 'operation failed'", errStr)
	}
}

// Additional tests for better coverage

func TestFormatError_AuthenticationError_DefaultCase(t *testing.T) {
	// Test the default case (not expired, not invalid, not 401, not 403)
	err := api.NewAuthenticationError(400, "Some other auth error", "")
	formatted := FormatError(err)

	ufErr, ok := formatted.(*UserFriendlyError)
	if !ok {
		t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
	}

	errStr := ufErr.Error()
	if !strings.Contains(errStr, "Authentication error") {
		t.Errorf("Error() = %v, want to contain 'Authentication error'", errStr)
	}
	if !strings.Contains(errStr, "threads auth status") {
		t.Errorf("Error() = %v, want suggestion to contain 'threads auth status'", errStr)
	}
}

func TestFormatError_RateLimitError_NoRetryAfter(t *testing.T) {
	err := api.NewRateLimitError(429, "Too many requests", "", 0)
	formatted := FormatError(err)

	ufErr, ok := formatted.(*UserFriendlyError)
	if !ok {
		t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
	}

	errStr := ufErr.Error()
	if !strings.Contains(errStr, "Rate limit") {
		t.Errorf("Error() = %v, want to contain 'Rate limit'", errStr)
	}
	if !strings.Contains(errStr, "Wait a few minutes") {
		t.Errorf("Error() = %v, want suggestion to contain 'Wait a few minutes'", errStr)
	}
}

func TestFormatError_ValidationError_SpecificPatterns(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "text too long",
			err:        api.NewValidationError(400, "Text is too long", "", "text"),
			wantSubstr: "500 characters",
		},
		{
			name:       "invalid url",
			err:        api.NewValidationError(400, "URL is invalid", "", "url"),
			wantSubstr: "http://",
		},
		{
			name:       "media format",
			err:        api.NewValidationError(400, "Unsupported media format", "", "media"),
			wantSubstr: "JPEG",
		},
		{
			name:       "carousel items",
			err:        api.NewValidationError(400, "Carousel has too few items", "", ""),
			wantSubstr: "2-20",
		},
		{
			name:       "empty field empty message",
			err:        api.NewValidationError(400, "", "", ""),
			wantSubstr: "--help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			ufErr, ok := formatted.(*UserFriendlyError)
			if !ok {
				t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
			}
			errStr := ufErr.Error()
			if !strings.Contains(errStr, tt.wantSubstr) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestFormatError_NetworkError_AllCases(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "connection refused",
			err:        api.NewNetworkError(0, "connection refused", "", false),
			wantSubstr: "temporarily unavailable",
		},
		{
			name:       "tls error",
			err:        api.NewNetworkError(0, "tls handshake error", "", false),
			wantSubstr: "SSL/TLS",
		},
		{
			name:       "certificate error",
			err:        api.NewNetworkError(0, "certificate invalid", "", false),
			wantSubstr: "SSL/TLS",
		},
		{
			name:       "default network error",
			err:        api.NewNetworkError(0, "unknown network issue", "", false),
			wantSubstr: "internet connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			ufErr, ok := formatted.(*UserFriendlyError)
			if !ok {
				t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
			}
			errStr := ufErr.Error()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestFormatError_APIError_AllCases(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "server error without request id",
			err:        api.NewAPIError(503, "Service unavailable", "", ""),
			wantSubstr: "server-side",
		},
		{
			name:       "deleted content",
			err:        api.NewAPIError(410, "Content has been deleted", "", ""),
			wantSubstr: "no longer exists",
		},
		{
			name:       "private content",
			err:        api.NewAPIError(403, "Content is private", "", ""),
			wantSubstr: "private content",
		},
		{
			name:       "default error without request id",
			err:        api.NewAPIError(400, "Bad request", "", ""),
			wantSubstr: "problem persists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			ufErr, ok := formatted.(*UserFriendlyError)
			if !ok {
				t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
			}
			errStr := ufErr.Error()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestFormatError_GenericErrors_AllCases(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "account not found",
			err:        errors.New("account not found"),
			wantSubstr: "threads auth login",
		},
		{
			name:       "client secret not stored",
			err:        errors.New("client secret not stored"),
			wantSubstr: "client ID and secret",
		},
		{
			name:       "cannot refresh",
			err:        errors.New("cannot refresh token"),
			wantSubstr: "client ID and secret",
		},
		{
			name:       "credential store error",
			err:        errors.New("could not access credential store"),
			wantSubstr: "keychain/keyring",
		},
		{
			name:       "keyring error",
			err:        errors.New("keyring access denied"),
			wantSubstr: "keychain/keyring",
		},
		{
			name:       "context deadline exceeded",
			err:        errors.New("context deadline exceeded"),
			wantSubstr: "timed out",
		},
		{
			name:       "context canceled",
			err:        errors.New("context canceled"),
			wantSubstr: "cancelled",
		},
		{
			name:       "json error",
			err:        errors.New("json: cannot unmarshal"),
			wantSubstr: "parse",
		},
		{
			name:       "unrecognized error",
			err:        errors.New("some unknown error"),
			wantSubstr: "some unknown error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			errStr := formatted.Error()
			if !strings.Contains(strings.ToLower(errStr), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("Error() = %v, want to contain %v", errStr, tt.wantSubstr)
			}
		})
	}
}

func TestFormatError_PreservesUserFriendlyError(t *testing.T) {
	original := &UserFriendlyError{
		Message:    "already formatted message",
		Suggestion: "already formatted suggestion",
		Cause:      errors.New("underlying"),
	}
	formatted := FormatError(original)
	ufErr, ok := formatted.(*UserFriendlyError)
	if !ok {
		t.Fatalf("FormatError() did not return *UserFriendlyError, got %T", formatted)
	}
	if ufErr.Message != original.Message {
		t.Errorf("Message = %q, want %q", ufErr.Message, original.Message)
	}
	if ufErr.Suggestion != original.Suggestion {
		t.Errorf("Suggestion = %q, want %q", ufErr.Suggestion, original.Suggestion)
	}
	if ufErr != original {
		t.Error("FormatError should return the same *UserFriendlyError pointer")
	}
}

func TestFormatError_PlainTokenExpired(t *testing.T) {
	err := errors.New("token expired")
	formatted := FormatError(err)
	ufErr, ok := formatted.(*UserFriendlyError)
	if !ok {
		t.Fatalf("FormatError(\"token expired\") did not return *UserFriendlyError, got %T", formatted)
	}
	if !strings.Contains(ufErr.Message, "expired") {
		t.Errorf("Message = %q, want to contain 'expired'", ufErr.Message)
	}
	if !strings.Contains(ufErr.Suggestion, "threads auth refresh") {
		t.Errorf("Suggestion = %q, want to contain 'threads auth refresh'", ufErr.Suggestion)
	}
}

func TestWriteErrorTo_TextMode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "auth error text",
			err:        api.NewAuthenticationError(401, "Token has expired", ""),
			wantSubstr: "expired",
		},
		{
			name:       "rate limit text",
			err:        api.NewRateLimitError(429, "Too many requests", "", 5*time.Minute),
			wantSubstr: "Rate limit",
		},
		{
			name:       "nil error",
			err:        nil,
			wantSubstr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := outfmt.WithFormat(context.Background(), "text")
			WriteErrorTo(ctx, &buf, tt.err)
			got := buf.String()
			if tt.err == nil {
				if got != "" {
					t.Errorf("WriteErrorTo(nil) wrote %q, want empty", got)
				}
				return
			}
			if !strings.Contains(strings.ToLower(got), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("WriteErrorTo() = %q, want to contain %q", got, tt.wantSubstr)
			}
		})
	}
}

func TestWriteErrorTo_JSONMode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantKind string
	}{
		{
			name:     "auth error",
			err:      api.NewAuthenticationError(401, "Token has expired", ""),
			wantKind: "auth",
		},
		{
			name:     "rate limit error",
			err:      api.NewRateLimitError(429, "Too many requests", "", 5*time.Minute),
			wantKind: "rate_limit",
		},
		{
			name:     "validation error",
			err:      api.NewValidationError(400, "Invalid value", "", "text"),
			wantKind: "validation",
		},
		{
			name:     "network error",
			err:      api.NewNetworkError(0, "Request timeout", "", true),
			wantKind: "network",
		},
		{
			name:     "api error",
			err:      api.NewAPIError(500, "Internal server error", "", "req-123"),
			wantKind: "api",
		},
		{
			name:     "unknown/plain error",
			err:      errors.New("something unexpected"),
			wantKind: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := outfmt.WithFormat(context.Background(), "json")
			WriteErrorTo(ctx, &buf, tt.err)

			var envelope struct {
				Error struct {
					Message string `json:"message"`
					Kind    string `json:"kind"`
				} `json:"error"`
			}
			if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
				t.Fatalf("WriteErrorTo() produced invalid JSON: %v\nOutput: %s", err, buf.String())
			}
			if envelope.Error.Kind != tt.wantKind {
				t.Errorf("error.kind = %q, want %q", envelope.Error.Kind, tt.wantKind)
			}
			if envelope.Error.Message == "" {
				t.Error("error.message is empty")
			}
		})
	}
}

func TestWriteErrorTo_JSONMode_Nil(t *testing.T) {
	var buf bytes.Buffer
	ctx := outfmt.WithFormat(context.Background(), "json")
	WriteErrorTo(ctx, &buf, nil)
	if buf.Len() != 0 {
		t.Errorf("WriteErrorTo(nil) wrote %q in JSON mode, want empty", buf.String())
	}
}

func TestWrapError_UserFriendlyError(t *testing.T) {
	original := &UserFriendlyError{
		Message:    "original message",
		Suggestion: "original suggestion",
		Cause:      errors.New("root cause"),
	}
	wrapped := WrapError("context prefix", original)
	ufErr, ok := wrapped.(*UserFriendlyError)
	if !ok {
		t.Fatalf("WrapError() did not return *UserFriendlyError, got %T", wrapped)
	}
	if !strings.HasPrefix(ufErr.Message, "context prefix") {
		t.Errorf("Message = %q, want prefix 'context prefix'", ufErr.Message)
	}
	if !strings.Contains(ufErr.Message, "original message") {
		t.Errorf("Message = %q, want to contain 'original message'", ufErr.Message)
	}
	if ufErr.Suggestion != "original suggestion" {
		t.Errorf("Suggestion = %q, want %q", ufErr.Suggestion, "original suggestion")
	}
}

func TestWrapError_PlainErrorFmtStyle(t *testing.T) {
	plain := errors.New("something went wrong")
	wrapped := WrapError("operation failed", plain)
	if wrapped == nil {
		t.Fatal("WrapError() returned nil")
	}
	got := wrapped.Error()
	want := fmt.Sprintf("operation failed: %s", plain.Error())
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
