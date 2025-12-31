package cmd

import (
	"testing"
)

func TestAuthCmd_Structure(t *testing.T) {
	// authCmd is a package-level var
	cmd := authCmd

	if cmd.Use != "auth" {
		t.Errorf("expected Use=auth, got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Check subcommands
	subcommands := cmd.Commands()
	expectedSubs := map[string]bool{
		"login":   true,
		"token":   true,
		"refresh": true,
		"status":  true,
		"list":    true,
		"remove":  true,
	}

	for _, sub := range subcommands {
		name := sub.Name()
		if !expectedSubs[name] {
			t.Errorf("unexpected subcommand: %s", name)
		}
		delete(expectedSubs, name)
	}

	for name := range expectedSubs {
		t.Errorf("missing subcommand: %s", name)
	}
}

func TestAuthLoginCmd_Structure(t *testing.T) {
	cmd := authLoginCmd

	if cmd.Use != "login" {
		t.Errorf("expected Use=login, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestAuthLoginCmd_Flags(t *testing.T) {
	cmd := authLoginCmd

	flags := []struct {
		name      string
		shorthand string
	}{
		{"name", "n"},
		{"client-id", ""},
		{"client-secret", ""},
		{"redirect-uri", ""},
		{"scopes", ""},
	}

	for _, flag := range flags {
		f := cmd.Flag(flag.name)
		if f == nil {
			t.Errorf("missing flag: %s", flag.name)
			continue
		}
		if flag.shorthand != "" && f.Shorthand != flag.shorthand {
			t.Errorf("flag %s expected shorthand %q, got %q", flag.name, flag.shorthand, f.Shorthand)
		}
	}

	// Check default for name flag
	nameFlag := cmd.Flag("name")
	if nameFlag.DefValue != "default" {
		t.Errorf("expected name default='default', got %s", nameFlag.DefValue)
	}
}

func TestAuthTokenCmd_Structure(t *testing.T) {
	cmd := authTokenCmd

	if cmd.Use != "token [access-token]" {
		t.Errorf("expected Use='token [access-token]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestAuthTokenCmd_Flags(t *testing.T) {
	cmd := authTokenCmd

	flags := []string{"name", "client-id", "client-secret"}
	for _, flag := range flags {
		if cmd.Flag(flag) == nil {
			t.Errorf("missing flag: %s", flag)
		}
	}
}

func TestAuthRefreshCmd_Structure(t *testing.T) {
	cmd := authRefreshCmd

	if cmd.Use != "refresh" {
		t.Errorf("expected Use=refresh, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestAuthStatusCmd_Structure(t *testing.T) {
	cmd := authStatusCmd

	if cmd.Use != "status" {
		t.Errorf("expected Use=status, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestAuthListCmd_Structure(t *testing.T) {
	cmd := authListCmd

	if cmd.Use != "list" {
		t.Errorf("expected Use=list, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestAuthRemoveCmd_Structure(t *testing.T) {
	cmd := authRemoveCmd

	if cmd.Use != "remove [account]" {
		t.Errorf("expected Use='remove [account]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestAuthCmd_SubcommandCount(t *testing.T) {
	cmd := authCmd
	subcommands := cmd.Commands()

	if len(subcommands) != 6 {
		t.Errorf("expected 6 subcommands, got %d", len(subcommands))
	}
}
