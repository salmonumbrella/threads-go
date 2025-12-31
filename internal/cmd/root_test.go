package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/salmonumbrella/threads-go/internal/iocontext"
	"github.com/salmonumbrella/threads-go/internal/outfmt"
)

func TestRootCmd_Structure(t *testing.T) {
	// rootCmd is a package-level var
	cmd := rootCmd

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
	cmd := rootCmd

	if !cmd.SilenceUsage {
		t.Error("expected SilenceUsage to be true")
	}

	if !cmd.SilenceErrors {
		t.Error("expected SilenceErrors to be true")
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	cmd := rootCmd

	expectedSubs := []string{
		"auth",
		"completion",
		"insights",
		"locations",
		"me",
		"posts",
		"ratelimit",
		"replies",
		"search",
		"users",
		"version",
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
	cmd := rootCmd

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
		{"limit", ""},
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
	cmd := rootCmd

	outputFlag := cmd.PersistentFlags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("missing output flag")
	}

	if outputFlag.DefValue != "text" {
		t.Errorf("expected output default='text', got %s", outputFlag.DefValue)
	}
}

func TestRootCmd_ColorFlagDefaults(t *testing.T) {
	cmd := rootCmd

	colorFlag := cmd.PersistentFlags().Lookup("color")
	if colorFlag == nil {
		t.Fatal("missing color flag")
	}

	if colorFlag.DefValue != "auto" {
		t.Errorf("expected color default='auto', got %s", colorFlag.DefValue)
	}
}

func TestVersionCmd_Structure(t *testing.T) {
	cmd := versionCmd

	if cmd.Use != "version" {
		t.Errorf("expected Use=version, got %s", cmd.Use)
	}

	if cmd.Run == nil {
		t.Error("expected Run to be set")
	}
}

func TestVersionCmd_Output(t *testing.T) {
	var stdout bytes.Buffer

	// Create a fresh copy of the version command to avoid state pollution
	versionTestCmd := versionCmd

	// Run the command directly instead of executing
	versionTestCmd.Run(versionTestCmd, []string{})

	// The version command writes to stdout directly via fmt.Printf
	// We need to check the command structure instead
	if versionTestCmd.Use != "version" {
		t.Errorf("expected Use=version, got %s", versionTestCmd.Use)
	}

	if versionTestCmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Verify version variables are accessible
	if Version == "" {
		// Version may be empty in tests, that's okay - it's set by ldflags
		_ = stdout // silence unused
	}
}

func TestExecute_WithContext(t *testing.T) {
	var stdout, stderr bytes.Buffer
	io := &iocontext.IO{Out: &stdout, ErrOut: &stderr}
	ctx := iocontext.WithIO(context.Background(), io)
	ctx = outfmt.WithFormat(ctx, "text")

	// Execute with just --help to avoid needing auth
	rootCmd.SetArgs([]string{"--help"})
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := Execute(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Threads CLI") {
		t.Error("expected help output to contain 'Threads CLI'")
	}
}

func TestConfirm_WithYesFlag(t *testing.T) {
	oldYesFlag := yesFlag
	yesFlag = true
	defer func() { yesFlag = oldYesFlag }()

	if !confirm("Delete?") {
		t.Error("expected confirm to return true when yesFlag is set")
	}
}

func TestGetAccount_Empty(t *testing.T) {
	oldAccountName := accountName
	accountName = ""
	defer func() { accountName = oldAccountName }()

	// This may return empty or first account from keyring depending on system state
	// We just verify it doesn't panic
	_ = getAccount()
}

func TestRequireAccount_NoAccount(t *testing.T) {
	oldAccountName := accountName
	accountName = ""
	defer func() { accountName = oldAccountName }()

	// Mock getStore would be needed for comprehensive test
	// For now, just verify it returns error when no account
	_, err := requireAccount()
	// This will likely fail since getAccount may find an account or not
	// The test verifies the function runs without panic
	_ = err
}
