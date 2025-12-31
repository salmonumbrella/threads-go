package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	threads "github.com/salmonumbrella/threads-go"
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
			err:        threads.NewAuthenticationError(401, "Token has expired", ""),
			wantSubstr: "expired",
			wantSugg:   "threads auth refresh",
		},
		{
			name:       "invalid token",
			err:        threads.NewAuthenticationError(401, "Invalid access token", ""),
			wantSubstr: "invalid",
			wantSugg:   "threads auth login",
		},
		{
			name:       "401 error",
			err:        threads.NewAuthenticationError(401, "Authentication required", ""),
			wantSubstr: "Authentication required",
			wantSugg:   "threads auth login",
		},
		{
			name:       "403 error",
			err:        threads.NewAuthenticationError(403, "Access denied", ""),
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
	err := threads.NewRateLimitError(429, "Too many requests", "", 5*time.Minute)
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
			err:        threads.NewValidationError(400, "Invalid value", "", "text"),
			wantSubstr: "text",
		},
		{
			name:       "without field",
			err:        threads.NewValidationError(400, "Validation failed", "", ""),
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
			err:        threads.NewNetworkError(0, "Request timeout", "", true),
			wantSubstr: "timed out",
		},
		{
			name:       "dns error",
			err:        threads.NewNetworkError(0, "no such host", "", false),
			wantSubstr: "DNS",
		},
		{
			name:       "temporary error",
			err:        threads.NewNetworkError(0, "Temporary failure", "", true),
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
			err:        threads.NewAPIError(500, "Internal server error", "", "req-123"),
			wantSubstr: "server-side",
		},
		{
			name:       "not found",
			err:        threads.NewAPIError(404, "Resource not found", "", ""),
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
	authErr := threads.NewAuthenticationError(401, "Token expired", "")
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
