package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

// mockPost is a test type for list_helper tests
type mockPost struct {
	ID     string
	Text   string
	Status string
}

func TestNewListCommand(t *testing.T) {
	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "TEXT", "STATUS"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID, p.Text, p.Status}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items:   []mockPost{{ID: "1", Text: "Hello", Status: "PUBLISHED"}},
				HasMore: false,
			}, nil
		},
		EmptyMessage: "No posts found",
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)
	if cmd.Use != "list" {
		t.Errorf("expected Use=list, got %s", cmd.Use)
	}
	if cmd.Short != "List items" {
		t.Errorf("expected Short='List items', got %s", cmd.Short)
	}
}

func TestNewListCommand_Flags(t *testing.T) {
	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{}, nil
		},
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	// Check that --limit flag exists
	limitFlag := cmd.Flag("limit")
	if limitFlag == nil {
		t.Error("expected --limit flag to exist")
	}

	// Check that --cursor flag exists
	cursorFlag := cmd.Flag("cursor")
	if cursorFlag == nil {
		t.Error("expected --cursor flag to exist")
	}

	// Check that --no-hints flag exists
	noHintsFlag := cmd.Flag("no-hints")
	if noHintsFlag == nil {
		t.Error("expected --no-hints flag to exist")
	}
}

func TestNewListCommand_TextOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "TEXT", "STATUS"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID, p.Text, p.Status}
		},
		ColumnTypes: []outfmt.ColumnType{outfmt.ColumnID, outfmt.ColumnPlain, outfmt.ColumnStatus},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items: []mockPost{
					{ID: "1", Text: "Hello", Status: "PUBLISHED"},
					{ID: "2", Text: "World", Status: "PENDING"},
				},
				HasMore: false,
			}, nil
		},
		EmptyMessage: "No posts found",
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	// Set up IO context
	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "text")
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ID") {
		t.Error("expected output to contain header ID")
	}
	if !strings.Contains(output, "TEXT") {
		t.Error("expected output to contain header TEXT")
	}
	if !strings.Contains(output, "Hello") {
		t.Error("expected output to contain Hello")
	}
	if !strings.Contains(output, "World") {
		t.Error("expected output to contain World")
	}
}

func TestNewListCommand_JSONOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "TEXT", "STATUS"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID, p.Text, p.Status}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items: []mockPost{
					{ID: "1", Text: "Hello", Status: "PUBLISHED"},
				},
				HasMore: true,
				Cursor:  "abc123",
			}, nil
		},
		EmptyMessage: "No posts found",
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "json")
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"items"`) {
		t.Error("expected JSON output to contain items key")
	}
	if !strings.Contains(output, `"has_more"`) {
		t.Error("expected JSON output to contain has_more key")
	}
	if !strings.Contains(output, `"cursor"`) {
		t.Error("expected JSON output to contain cursor key")
	}
}

func TestNewListCommand_JSONLOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "TEXT", "STATUS"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID, p.Text, p.Status}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items: []mockPost{
					{ID: "1", Text: "Hello", Status: "PUBLISHED"},
					{ID: "2", Text: "World", Status: "ACTIVE"},
				},
				HasMore: true,
				Cursor:  "abc123",
			}, nil
		},
		EmptyMessage: "No posts found",
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "jsonl")
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()
	if strings.Contains(out, `"items"`) {
		t.Fatalf("did not expect wrapper JSON in jsonl output, got: %q", out)
	}
	if !strings.Contains(out, `"ID":"1"`) || !strings.Contains(out, `"ID":"2"`) {
		t.Fatalf("expected jsonl lines with IDs, got: %q", out)
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "More results available") || !strings.Contains(errOut, "--cursor abc123") {
		t.Fatalf("expected cursor hint on stderr, got: %q", errOut)
	}
}

func TestNewListCommand_EmptyResults_Text(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "TEXT"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID, p.Text}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items:   []mockPost{},
				HasMore: false,
			}, nil
		},
		EmptyMessage: "No posts found",
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "text")
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No posts found") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestNewListCommand_EmptyResults_JSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID", "TEXT"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID, p.Text}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items:   []mockPost{},
				HasMore: false,
			}, nil
		},
		EmptyMessage: "No posts found",
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "json")
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"items": []`) {
		t.Errorf("expected empty JSON array for items, got: %s", output)
	}
	if !strings.Contains(output, `"has_more": false`) {
		t.Errorf("expected has_more: false in JSON output, got: %s", output)
	}
}

func TestNewListCommand_FetchError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	expectedErr := errors.New("API error")
	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{}, expectedErr
		},
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "API error") {
		t.Errorf("expected API error, got: %v", err)
	}
}

func TestNewListCommand_ClientError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	expectedErr := errors.New("client initialization error")
	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{}, nil
		},
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, expectedErr
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "client initialization error") {
		t.Errorf("expected client initialization error, got: %v", err)
	}
}

func TestNewListCommand_LimitAndCursor(t *testing.T) {
	var capturedLimit int
	var capturedCursor string

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			capturedLimit = limit
			capturedCursor = cursor
			return ListResult[mockPost]{
				Items: []mockPost{{ID: "1"}},
			}, nil
		},
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	var stdout, stderr bytes.Buffer
	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--limit", "50", "--cursor", "next_page_cursor"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedLimit != 50 {
		t.Errorf("expected limit=50, got %d", capturedLimit)
	}
	if capturedCursor != "next_page_cursor" {
		t.Errorf("expected cursor='next_page_cursor', got %s", capturedCursor)
	}
}

func TestNewListCommand_HasMoreHint(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items:   []mockPost{{ID: "1"}},
				HasMore: true,
				Cursor:  "next_cursor_123",
			}, nil
		},
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "text")
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The pagination hint should be on stderr
	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "next_cursor_123") {
		t.Errorf("expected pagination hint with cursor on stderr, got: %s", stderrOutput)
	}
}

func TestNewListCommand_LimitMax(t *testing.T) {
	var capturedLimit int

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			capturedLimit = limit
			return ListResult[mockPost]{Items: []mockPost{{ID: "1"}}}, nil
		},
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	var stdout, stderr bytes.Buffer
	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--limit", "200"}) // Exceeds max of 100

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Limit should be capped at 100
	if capturedLimit != 100 {
		t.Errorf("expected limit to be capped at 100, got %d", capturedLimit)
	}
}

func TestListResult_Generic(t *testing.T) {
	// Test that ListResult works with different types
	type customType struct {
		Name string
		Age  int
	}

	result := ListResult[customType]{
		Items: []customType{
			{Name: "Alice", Age: 30},
			{Name: "Bob", Age: 25},
		},
		HasMore: true,
		Cursor:  "page2",
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Name != "Alice" {
		t.Errorf("expected first item name to be Alice, got %s", result.Items[0].Name)
	}
	if !result.HasMore {
		t.Error("expected HasMore to be true")
	}
	if result.Cursor != "page2" {
		t.Errorf("expected Cursor to be page2, got %s", result.Cursor)
	}
}

func TestNewListCommand_NoHints_Text(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID}
		},
		ColumnTypes: []outfmt.ColumnType{outfmt.ColumnID},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items:   []mockPost{{ID: "1"}},
				HasMore: true,
				Cursor:  "next_cursor",
			}, nil
		},
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "text")
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--no-hints"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stderr.String(), "More results") {
		t.Fatalf("expected no hint on stderr with --no-hints, got: %q", stderr.String())
	}
}

func TestNewListCommand_NoHints_JSONL(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cfg := ListConfig[mockPost]{
		Use:     "list",
		Short:   "List items",
		Headers: []string{"ID"},
		RowFunc: func(p mockPost) []string {
			return []string{p.ID}
		},
		Fetch: func(ctx context.Context, client *api.Client, cursor string, limit int) (ListResult[mockPost], error) {
			return ListResult[mockPost]{
				Items:   []mockPost{{ID: "1"}},
				HasMore: true,
				Cursor:  "next_cursor",
			}, nil
		},
	}

	getClient := func(ctx context.Context) (*api.Client, error) {
		return nil, nil
	}

	cmd := NewListCommand(cfg, getClient)

	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "jsonl")
	cmd.SetContext(ctx)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--no-hints"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stderr.String(), "More results") {
		t.Fatalf("expected no hint on stderr with --no-hints in JSONL mode, got: %q", stderr.String())
	}
}
