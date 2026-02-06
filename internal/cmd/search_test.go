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

func TestSearchCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewSearchCmd(f)

	if cmd.Use != "search [query]" {
		t.Errorf("expected Use='search [query]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}
}

func TestSearchCmd_Flags(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewSearchCmd(f)

	flags := []string{"limit", "cursor", "media-type", "since", "until", "mode", "type", "best", "emit"}
	for _, flag := range flags {
		if cmd.Flag(flag) == nil {
			t.Errorf("missing flag: %s", flag)
		}
	}

	limitFlag := cmd.Flag("limit")
	if limitFlag.DefValue != "25" {
		t.Errorf("expected limit default=25, got %s", limitFlag.DefValue)
	}
}

func TestSearchCmd_Best_EmitID(t *testing.T) {
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

		if r.URL.Path != "/keyword_search" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":                 "p1",
					"permalink":          "https://www.threads.net/t/p1",
					"timestamp":          time.Now().UTC().Format(time.RFC3339),
					"username":           "alice",
					"media_product_type": "THREADS",
					"is_reply":           false,
				},
				{
					"id":                 "p2",
					"permalink":          "https://www.threads.net/t/p2",
					"timestamp":          time.Now().UTC().Format(time.RFC3339),
					"username":           "bob",
					"media_product_type": "THREADS",
					"is_reply":           false,
				},
			},
			"paging": map[string]any{},
		})
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)
	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	ctx = outfmt.WithFormat(ctx, "text")

	cmd := NewSearchCmd(f)
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"coffee", "--best", "--emit", "id"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("search --best --emit id failed: %v", err)
	}

	out := io.Out.(*bytes.Buffer).String()
	if out != "p1\n" {
		t.Fatalf("expected best id p1, got %q", out)
	}
}
