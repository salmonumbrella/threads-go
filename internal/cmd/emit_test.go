package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

func TestParseEmitMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    emitMode
		wantErr bool
	}{
		{name: "json lowercase", input: "json", want: emitJSON},
		{name: "id lowercase", input: "id", want: emitID},
		{name: "url lowercase", input: "url", want: emitURL},
		{name: "JSON uppercase", input: "JSON", want: emitJSON},
		{name: "Id mixed case", input: "Id", want: emitID},
		{name: "URL uppercase", input: "URL", want: emitURL},
		{name: "whitespace padded", input: " json ", want: emitJSON},
		{name: "invalid value", input: "invalid", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEmitMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseEmitMode(%q) = %v, want error", tt.input, got)
				}
				var ufErr *UserFriendlyError
				if ok := isUserFriendlyError(err, &ufErr); !ok {
					t.Errorf("parseEmitMode(%q) error is %T, want *UserFriendlyError", tt.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEmitMode(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseEmitMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// isUserFriendlyError checks if err is *UserFriendlyError and sets target.
func isUserFriendlyError(err error, target **UserFriendlyError) bool {
	uf, ok := err.(*UserFriendlyError)
	if ok && target != nil {
		*target = uf
	}
	return ok
}

func TestEmitResult_JSONMode(t *testing.T) {
	type sampleItem struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	}

	tests := []struct {
		name    string
		mode    emitMode
		id      string
		url     string
		item    any
		wantKey string // top-level JSON key to check
		wantVal string // expected value for that key
		wantErr bool
	}{
		{
			name:    "emitID outputs id",
			mode:    emitID,
			id:      "test-id-123",
			url:     "https://example.com/post/123",
			item:    sampleItem{ID: "test-id-123", Text: "hello"},
			wantKey: "id",
			wantVal: "test-id-123",
		},
		{
			name:    "emitURL outputs url",
			mode:    emitURL,
			id:      "test-id-123",
			url:     "https://example.com/post/123",
			item:    sampleItem{ID: "test-id-123", Text: "hello"},
			wantKey: "url",
			wantVal: "https://example.com/post/123",
		},
		{
			name:    "emitURL with empty url errors",
			mode:    emitURL,
			id:      "test-id-123",
			url:     "",
			item:    sampleItem{ID: "test-id-123", Text: "hello"},
			wantErr: true,
		},
		{
			name:    "emitJSON outputs full item",
			mode:    emitJSON,
			id:      "test-id-123",
			url:     "https://example.com/post/123",
			item:    sampleItem{ID: "test-id-123", Text: "hello"},
			wantKey: "text",
			wantVal: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := outfmt.WithFormat(context.Background(), "json")
			io := &iocontext.IO{Out: &buf, ErrOut: &bytes.Buffer{}}

			err := emitResult(ctx, io, tt.mode, tt.id, tt.url, tt.item)
			if tt.wantErr {
				if err == nil {
					t.Fatal("emitResult() = nil, want error")
				}
				var ufErr *UserFriendlyError
				if !isUserFriendlyError(err, &ufErr) {
					t.Errorf("error is %T, want *UserFriendlyError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("emitResult() error = %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
				t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
			}
			val, ok := parsed[tt.wantKey]
			if !ok {
				t.Fatalf("JSON output missing key %q: %s", tt.wantKey, buf.String())
			}
			if valStr, ok := val.(string); !ok || valStr != tt.wantVal {
				t.Errorf("JSON[%q] = %v, want %q", tt.wantKey, val, tt.wantVal)
			}
		})
	}
}

func TestEmitResult_TextMode(t *testing.T) {
	type sampleItem struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	}

	tests := []struct {
		name       string
		mode       emitMode
		id         string
		url        string
		item       any
		wantSubstr string
		wantErr    bool
	}{
		{
			name:       "emitID outputs just the id",
			mode:       emitID,
			id:         "abc-123",
			url:        "https://example.com",
			item:       sampleItem{ID: "abc-123", Text: "hello"},
			wantSubstr: "abc-123",
		},
		{
			name:       "emitURL outputs just the url",
			mode:       emitURL,
			id:         "abc-123",
			url:        "https://example.com/post/123",
			item:       sampleItem{ID: "abc-123", Text: "hello"},
			wantSubstr: "https://example.com/post/123",
		},
		{
			name:    "emitURL with empty url errors",
			mode:    emitURL,
			id:      "abc-123",
			url:     "  ",
			item:    sampleItem{ID: "abc-123", Text: "hello"},
			wantErr: true,
		},
		{
			name:       "emitJSON outputs pretty-printed JSON",
			mode:       emitJSON,
			id:         "abc-123",
			url:        "https://example.com",
			item:       sampleItem{ID: "abc-123", Text: "hello"},
			wantSubstr: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := outfmt.WithFormat(context.Background(), "text")
			io := &iocontext.IO{Out: &buf, ErrOut: &bytes.Buffer{}}

			err := emitResult(ctx, io, tt.mode, tt.id, tt.url, tt.item)
			if tt.wantErr {
				if err == nil {
					t.Fatal("emitResult() = nil, want error")
				}
				var ufErr *UserFriendlyError
				if !isUserFriendlyError(err, &ufErr) {
					t.Errorf("error is %T, want *UserFriendlyError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("emitResult() error = %v", err)
			}

			got := buf.String()
			if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("output = %q, want to contain %q", got, tt.wantSubstr)
			}

			// Text scalar modes should output a single clean line
			if tt.mode == emitID || tt.mode == emitURL {
				trimmed := strings.TrimSpace(got)
				if trimmed != tt.wantSubstr {
					t.Errorf("text output = %q, want exactly %q", trimmed, tt.wantSubstr)
				}
			}

			// emitJSON in text mode should produce valid JSON
			if tt.mode == emitJSON {
				var parsed map[string]any
				if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
					t.Errorf("emitJSON text mode output is not valid JSON: %v\nOutput: %s", err, got)
				}
			}
		})
	}
}
