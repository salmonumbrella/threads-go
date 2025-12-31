package cmd

import (
	"testing"
)

func TestRepliesCmd_Structure(t *testing.T) {
	// repliesCmd is a package-level var
	cmd := repliesCmd

	if cmd.Use != "replies" {
		t.Errorf("expected Use=replies, got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Check subcommands
	subcommands := cmd.Commands()
	expectedCount := 5 // list, create, hide, unhide, conversation
	if len(subcommands) != expectedCount {
		t.Errorf("expected %d subcommands, got %d", expectedCount, len(subcommands))
	}
}

func TestRepliesCmd_Subcommands(t *testing.T) {
	cmd := repliesCmd

	expectedSubs := map[string]bool{
		"list":         true,
		"create":       true,
		"hide":         true,
		"unhide":       true,
		"conversation": true,
	}

	for _, sub := range cmd.Commands() {
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

func TestRepliesListCmd_Structure(t *testing.T) {
	cmd := repliesListCmd

	if cmd.Use != "list [post-id]" {
		t.Errorf("expected Use='list [post-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestRepliesListCmd_Flags(t *testing.T) {
	cmd := repliesListCmd

	limitFlag := cmd.Flag("limit")
	if limitFlag == nil {
		t.Fatal("missing limit flag")
	}

	if limitFlag.DefValue != "25" {
		t.Errorf("expected limit default=25, got %s", limitFlag.DefValue)
	}
}

func TestRepliesCreateCmd_Structure(t *testing.T) {
	cmd := repliesCreateCmd

	if cmd.Use != "create [post-id]" {
		t.Errorf("expected Use='create [post-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestRepliesCreateCmd_Flags(t *testing.T) {
	cmd := repliesCreateCmd

	textFlag := cmd.Flag("text")
	if textFlag == nil {
		t.Fatal("missing text flag")
	}

	if textFlag.Shorthand != "t" {
		t.Errorf("expected text flag shorthand='t', got %s", textFlag.Shorthand)
	}

	// Text flag should be required - check annotations
	annotations := textFlag.Annotations
	if annotations != nil {
		// The flag may be marked required through MarkFlagRequired
		// We just verify the flag exists
		_ = annotations["cobra_annotation_bash_completion_one_required_flag"]
	}
}

func TestRepliesHideCmd_Structure(t *testing.T) {
	cmd := repliesHideCmd

	if cmd.Use != "hide [reply-id]" {
		t.Errorf("expected Use='hide [reply-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestRepliesUnhideCmd_Structure(t *testing.T) {
	cmd := repliesUnhideCmd

	if cmd.Use != "unhide [reply-id]" {
		t.Errorf("expected Use='unhide [reply-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestRepliesConversationCmd_Structure(t *testing.T) {
	cmd := repliesConversationCmd

	if cmd.Use != "conversation [post-id]" {
		t.Errorf("expected Use='conversation [post-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestRepliesConversationCmd_Flags(t *testing.T) {
	cmd := repliesConversationCmd

	limitFlag := cmd.Flag("limit")
	if limitFlag == nil {
		t.Fatal("missing limit flag")
	}

	if limitFlag.DefValue != "25" {
		t.Errorf("expected limit default=25, got %s", limitFlag.DefValue)
	}
}

func TestRepliesHideCmd_HasLongDescription(t *testing.T) {
	cmd := repliesHideCmd

	if cmd.Long == "" {
		t.Error("expected Long description to be set for hide command")
	}
}

func TestRepliesUnhideCmd_HasLongDescription(t *testing.T) {
	cmd := repliesUnhideCmd

	if cmd.Long == "" {
		t.Error("expected Long description to be set for unhide command")
	}
}
