package cmd

import (
	"testing"
)

func TestValidateHTTPSURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		fieldName string
		wantErr   bool
		wantMsg   string
	}{
		{
			name:      "valid https url",
			url:       "https://example.com/webhooks",
			fieldName: "Callback URL",
			wantErr:   false,
		},
		{
			name:      "http url rejected",
			url:       "http://example.com/webhooks",
			fieldName: "Callback URL",
			wantErr:   true,
			wantMsg:   "Callback URL must use HTTPS",
		},
		{
			name:      "empty url rejected",
			url:       "",
			fieldName: "Webhook URL",
			wantErr:   true,
			wantMsg:   "Webhook URL must use HTTPS",
		},
		{
			name:      "url without protocol rejected",
			url:       "example.com/webhooks",
			fieldName: "Callback URL",
			wantErr:   true,
			wantMsg:   "Callback URL must use HTTPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHTTPSURL(tt.url, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHTTPSURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				ufErr, ok := err.(*UserFriendlyError)
				if !ok {
					t.Errorf("Expected *UserFriendlyError, got %T", err)
					return
				}
				if ufErr.Message != tt.wantMsg {
					t.Errorf("ValidateHTTPSURL() message = %q, want %q", ufErr.Message, tt.wantMsg)
				}
			}
		})
	}
}
