package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

func TestLocationsSearchCmd_Best_EmitID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/refresh_access_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "refreshed-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}

		if r.URL.Path != "/location_search" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":        "L1",
					"name":      "Coffee Shop",
					"address":   "123 Main St",
					"latitude":  37.0,
					"longitude": -122.0,
				},
			},
			"paging": map[string]any{},
			"meta": map[string]any{
				"generated_at": time.Now().UTC().Format(time.RFC3339),
			},
		})
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)
	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	ctx = outfmt.WithFormat(ctx, "text")

	cmd := NewLocationsCmd(f)
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"search", "coffee", "--best", "--emit", "id"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("locations search --best --emit id failed: %v", err)
	}

	out := io.Out.(*bytes.Buffer).String()
	if out != "L1\n" {
		t.Fatalf("expected best id L1, got %q", out)
	}
}
