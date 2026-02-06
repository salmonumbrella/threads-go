package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
)

func TestReadTextFileOrStdin_File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "post.txt")
	want := "Hello\n@alice\n\n"
	if err := os.WriteFile(p, []byte(want), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := readTextFileOrStdin(context.Background(), p)
	if err != nil {
		t.Fatalf("readTextFileOrStdin(file) error: %v", err)
	}
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
	if got != "from stdin\n" {
		t.Fatalf("expected %q, got %q", "from stdin\n", got)
	}

	io2 := &iocontext.IO{In: bytes.NewBufferString("from @-\n")}
	ctx2 := iocontext.WithIO(context.Background(), io2)
	got2, err := readTextFileOrStdin(ctx2, "@-")
	if err != nil {
		t.Fatalf("readTextFileOrStdin(@-) error: %v", err)
	}
	if got2 != "from @-\n" {
		t.Fatalf("expected %q, got %q", "from @-\n", got2)
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
