// Package helpers provides shared CLI utility functions.
package helpers

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/salmonumbrella/threads-go/internal/iocontext"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
	"github.com/spf13/cobra"
)

func TestConfirmOrYes_WithYesFlag(t *testing.T) {
	ctx := outfmt.WithYes(context.Background(), true)
	confirmed, err := ConfirmOrYes(ctx, "Delete?")
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Error("expected confirmation with --yes flag")
	}
}

func TestConfirmOrYes_JSONMode(t *testing.T) {
	ctx := outfmt.WithFormat(context.Background(), "json")
	confirmed, err := ConfirmOrYes(ctx, "Delete?")
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Error("expected confirmation in JSON mode")
	}
}

func TestConfirmOrYes_UserSaysYes(t *testing.T) {
	var outBuf bytes.Buffer
	ctx := context.Background()
	ctx = outfmt.WithFormat(ctx, "text")
	ctx = outfmt.WithYes(ctx, false)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{
		Out:    &outBuf,
		ErrOut: &outBuf,
		In:     strings.NewReader("yes\n"),
	})

	// Mock terminal check
	oldIsTerminal := isTerminal
	isTerminal = func() bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	confirmed, err := ConfirmOrYes(ctx, "Delete?")
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Error("expected confirmation when user says yes")
	}

	// Check prompt was written
	if !strings.Contains(outBuf.String(), "Delete?") {
		t.Errorf("expected prompt in output, got: %s", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "[y/N]") {
		t.Errorf("expected [y/N] suffix in output, got: %s", outBuf.String())
	}
}

func TestConfirmOrYes_UserSaysY(t *testing.T) {
	var outBuf bytes.Buffer
	ctx := context.Background()
	ctx = outfmt.WithFormat(ctx, "text")
	ctx = outfmt.WithYes(ctx, false)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{
		Out:    &outBuf,
		ErrOut: &outBuf,
		In:     strings.NewReader("y\n"),
	})

	oldIsTerminal := isTerminal
	isTerminal = func() bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	confirmed, err := ConfirmOrYes(ctx, "Delete?")
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Error("expected confirmation when user says y")
	}
}

func TestConfirmOrYes_UserSaysYesCaseInsensitive(t *testing.T) {
	var outBuf bytes.Buffer
	ctx := context.Background()
	ctx = outfmt.WithFormat(ctx, "text")
	ctx = outfmt.WithYes(ctx, false)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{
		Out:    &outBuf,
		ErrOut: &outBuf,
		In:     strings.NewReader("YES\n"),
	})

	oldIsTerminal := isTerminal
	isTerminal = func() bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	confirmed, err := ConfirmOrYes(ctx, "Delete?")
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed {
		t.Error("expected confirmation when user says YES")
	}
}

func TestConfirmOrYes_UserSaysNo(t *testing.T) {
	var outBuf bytes.Buffer
	ctx := context.Background()
	ctx = outfmt.WithFormat(ctx, "text")
	ctx = outfmt.WithYes(ctx, false)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{
		Out:    &outBuf,
		ErrOut: &outBuf,
		In:     strings.NewReader("no\n"),
	})

	oldIsTerminal := isTerminal
	isTerminal = func() bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	confirmed, err := ConfirmOrYes(ctx, "Delete?")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed {
		t.Error("expected no confirmation when user says no")
	}
}

func TestConfirmOrYes_UserPressesEnter(t *testing.T) {
	var outBuf bytes.Buffer
	ctx := context.Background()
	ctx = outfmt.WithFormat(ctx, "text")
	ctx = outfmt.WithYes(ctx, false)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{
		Out:    &outBuf,
		ErrOut: &outBuf,
		In:     strings.NewReader("\n"),
	})

	oldIsTerminal := isTerminal
	isTerminal = func() bool { return true }
	defer func() { isTerminal = oldIsTerminal }()

	confirmed, err := ConfirmOrYes(ctx, "Delete?")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed {
		t.Error("expected no confirmation when user just presses enter")
	}
}

func TestConfirmOrYes_NotTerminal(t *testing.T) {
	ctx := context.Background()
	ctx = outfmt.WithFormat(ctx, "text")
	ctx = outfmt.WithYes(ctx, false)

	oldIsTerminal := isTerminal
	isTerminal = func() bool { return false }
	defer func() { isTerminal = oldIsTerminal }()

	_, err := ConfirmOrYes(ctx, "Delete?")
	if err == nil {
		t.Fatal("expected error when not a terminal")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("expected error message to mention --yes, got: %s", err.Error())
	}
}

func TestMustMarkRequired_Success(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("required-flag", "", "a required flag")

	// Should not panic
	MustMarkRequired(cmd, "required-flag")
}

func TestMustMarkRequired_PanicsOnMissingFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing flag")
		}
	}()

	MustMarkRequired(cmd, "nonexistent-flag")
}
