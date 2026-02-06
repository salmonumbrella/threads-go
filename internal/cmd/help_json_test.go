package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

func TestHelpJSONCmd_Root(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewRootCmd(f)

	var stdout, stderr bytes.Buffer
	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "text")
	cmd.SetContext(ctx)

	cmd.SetArgs([]string{"help-json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help-json failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput: %s", err, stdout.String())
	}
	if payload["command_path"] != "threads" {
		t.Fatalf("expected command_path=threads, got %v", payload["command_path"])
	}
	if payload["use"] == "" {
		t.Fatalf("expected use to be set, got empty")
	}
}

func TestHelpJSONCmd_SubcommandPath(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewRootCmd(f)

	var stdout, stderr bytes.Buffer
	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "text")
	cmd.SetContext(ctx)

	cmd.SetArgs([]string{"help-json", "posts", "get"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help-json posts get failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput: %s", err, stdout.String())
	}
	if payload["command_path"] != "threads posts get" {
		t.Fatalf("expected command_path=threads posts get, got %v", payload["command_path"])
	}
	if payload["use"] == "" {
		t.Fatalf("expected use to be set, got empty")
	}
}
