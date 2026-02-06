package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
)

func TestReadTextFileOrStdin_File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.txt")
	// File content has trailing newlines; they should be trimmed.
	if err := os.WriteFile(p, []byte("Hello\n@alice\n\n"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := readTextFileOrStdin(context.Background(), p)
	if err != nil {
		t.Fatalf("readTextFileOrStdin(file) error: %v", err)
	}
	want := "Hello\n@alice"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	got2, err := readTextFileOrStdin(context.Background(), "@"+p)
	if err != nil {
		t.Fatalf("readTextFileOrStdin(@file) error: %v", err)
	}
	if got2 != want {
		t.Fatalf("expected %q, got %q", want, got2)
	}
}

func TestReadTextFileOrStdin_Stdin(t *testing.T) {
	io := &iocontext.IO{In: bytes.NewBufferString("from stdin\n")}
	ctx := iocontext.WithIO(context.Background(), io)

	got, err := readTextFileOrStdin(ctx, "-")
	if err != nil {
		t.Fatalf("readTextFileOrStdin(-) error: %v", err)
	}
	// Trailing newline should be trimmed.
	if got != "from stdin" {
		t.Fatalf("expected %q, got %q", "from stdin", got)
	}

	io2 := &iocontext.IO{In: bytes.NewBufferString("from @-\n")}
	ctx2 := iocontext.WithIO(context.Background(), io2)
	got2, err := readTextFileOrStdin(ctx2, "@-")
	if err != nil {
		t.Fatalf("readTextFileOrStdin(@-) error: %v", err)
	}
	if got2 != "from @-" {
		t.Fatalf("expected %q, got %q", "from @-", got2)
	}
}

func TestReadTextFileOrStdin_MissingFile(t *testing.T) {
	_, err := readTextFileOrStdin(context.Background(), "this-file-should-not-exist.txt")
	if err == nil {
		t.Fatalf("expected error")
	}
	if _, ok := err.(*UserFriendlyError); !ok {
		t.Fatalf("expected UserFriendlyError, got %T", err)
	}
}

func TestReadTextFileOrStdin_TrailingNewlineTrimmed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "newlines.txt")
	if err := os.WriteFile(p, []byte("hello world\n\n\n"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := readTextFileOrStdin(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", got)
	}
}

func TestReadTextFileOrStdin_NoContent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(p, []byte("\n\n"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := readTextFileOrStdin(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestReadTextFileOrStdin_SizeLimitFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "huge.txt")
	// Write a file just over 1 MiB.
	data := make([]byte, maxTextInputSize+1)
	for i := range data {
		data[i] = 'x'
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := readTextFileOrStdin(context.Background(), p)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	ufErr, ok := err.(*UserFriendlyError)
	if !ok {
		t.Fatalf("expected UserFriendlyError, got %T: %v", err, err)
	}
	if !strings.Contains(ufErr.Message, "too large") {
		t.Fatalf("expected 'too large' in message, got: %s", ufErr.Message)
	}
}

func TestReadTextFileOrStdin_SizeLimitStdin(t *testing.T) {
	// Create input just over 1 MiB via stdin.
	data := make([]byte, maxTextInputSize+1)
	for i := range data {
		data[i] = 'y'
	}
	io := &iocontext.IO{In: bytes.NewBuffer(data)}
	ctx := iocontext.WithIO(context.Background(), io)

	_, err := readTextFileOrStdin(ctx, "-")
	if err == nil {
		t.Fatal("expected error for oversized stdin")
	}
	ufErr, ok := err.(*UserFriendlyError)
	if !ok {
		t.Fatalf("expected UserFriendlyError, got %T: %v", err, err)
	}
	if !strings.Contains(ufErr.Message, "too large") {
		t.Fatalf("expected 'too large' in message, got: %s", ufErr.Message)
	}
}

func TestReadTextFileOrStdin_ExactSizeLimit(t *testing.T) {
	// Exactly at the limit should succeed.
	dir := t.TempDir()
	p := filepath.Join(dir, "exact.txt")
	data := make([]byte, maxTextInputSize)
	for i := range data {
		data[i] = 'z'
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := readTextFileOrStdin(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != maxTextInputSize {
		t.Fatalf("expected length %d, got %d", maxTextInputSize, len(got))
	}
}
