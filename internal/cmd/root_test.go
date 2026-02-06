package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

func TestRootCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewRootCmd(f)

	if cmd.Use != "threads" {
		t.Errorf("expected Use=threads, got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestRootCmd_SilencesOutput(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewRootCmd(f)

	if !cmd.SilenceUsage {
		t.Error("expected SilenceUsage to be true")
	}

	if !cmd.SilenceErrors {
		t.Error("expected SilenceErrors to be true")
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewRootCmd(f)

	expectedSubs := []string{
		"auth",
		"completion",
		"config",
		"help-json",
		"insights",
		"locations",
		"me",
		"posts",
		"ratelimit",
		"replies",
		"search",
		"users",
		"version",
		"webhooks",
	}

	subcommands := cmd.Commands()
	subNames := make(map[string]bool)
	for _, sub := range subcommands {
		subNames[sub.Name()] = true
	}

	for _, expected := range expectedSubs {
		if !subNames[expected] {
			t.Errorf("missing expected subcommand: %s", expected)
		}
	}
}

func TestRootCmd_GlobalFlags(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewRootCmd(f)

	flags := []struct {
		name      string
		shorthand string
	}{
		{"account", "a"},
		{"output", "o"},
		{"color", ""},
		{"debug", ""},
		{"query", "q"},
		{"yes", "y"},
	}

	for _, f := range flags {
		flag := cmd.PersistentFlags().Lookup(f.name)
		if flag == nil {
			t.Errorf("missing global flag: %s", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("flag %s expected shorthand %q, got %q", f.name, f.shorthand, flag.Shorthand)
		}
	}
}

func TestRootCmd_OutputFlagDefaults(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewRootCmd(f)

	outputFlag := cmd.PersistentFlags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("missing output flag")
	}

	if outputFlag.DefValue != "text" {
		t.Errorf("expected output default='text', got %s", outputFlag.DefValue)
	}
}

func TestRootCmd_ColorFlagDefaults(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewRootCmd(f)

	colorFlag := cmd.PersistentFlags().Lookup("color")
	if colorFlag == nil {
		t.Fatal("missing color flag")
	}

	if colorFlag.DefValue != "auto" {
		t.Errorf("expected color default='auto', got %s", colorFlag.DefValue)
	}
}

func TestVersionCmd_Structure(t *testing.T) {
	cmd := NewVersionCmd()

	if cmd.Use != "version" {
		t.Errorf("expected Use=version, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestExecute_WithContext(t *testing.T) {
	var stdout, stderr bytes.Buffer
	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "text")

	f := newTestFactory(t)
	cmd := NewRootCmd(f)
	cmd.SetArgs([]string{"--help"})
	cmd.SetContext(ctx)

	err := ExecuteCommand(cmd, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Threads CLI") {
		t.Error("expected help output to contain 'Threads CLI'")
	}
}
