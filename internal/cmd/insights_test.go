package cmd

import (
	"testing"
)

func TestInsightsCmd_Structure(t *testing.T) {
	// insightsCmd is a package-level var
	cmd := insightsCmd

	if cmd.Use != "insights" {
		t.Errorf("expected Use=insights, got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Check subcommands
	subcommands := cmd.Commands()
	if len(subcommands) != 2 {
		t.Errorf("expected 2 subcommands (post, account), got %d", len(subcommands))
	}

	expectedSubs := map[string]bool{
		"post":    true,
		"account": true,
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

func TestInsightsPostCmd_Structure(t *testing.T) {
	cmd := insightsPostCmd

	if cmd.Use != "post [post-id]" {
		t.Errorf("expected Use='post [post-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestInsightsPostCmd_Flags(t *testing.T) {
	cmd := insightsPostCmd

	metricsFlag := cmd.Flag("metrics")
	if metricsFlag == nil {
		t.Fatal("missing metrics flag")
	}

	// Check default value contains expected metrics
	defaultMetrics := metricsFlag.DefValue
	expectedDefaults := []string{"views", "likes", "replies", "reposts"}
	for _, metric := range expectedDefaults {
		if defaultMetrics == "" {
			t.Errorf("expected default metrics to contain %s", metric)
		}
	}
}

func TestInsightsAccountCmd_Structure(t *testing.T) {
	cmd := insightsAccountCmd

	if cmd.Use != "account" {
		t.Errorf("expected Use=account, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestInsightsAccountCmd_Flags(t *testing.T) {
	cmd := insightsAccountCmd

	// Check metrics flag
	metricsFlag := cmd.Flag("metrics")
	if metricsFlag == nil {
		t.Fatal("missing metrics flag")
	}

	// Check period flag
	periodFlag := cmd.Flag("period")
	if periodFlag == nil {
		t.Fatal("missing period flag")
	}

	if periodFlag.DefValue != "lifetime" {
		t.Errorf("expected period default='lifetime', got %s", periodFlag.DefValue)
	}
}

func TestInsightsPostCmd_HasLongDescription(t *testing.T) {
	cmd := insightsPostCmd

	if cmd.Long == "" {
		t.Error("expected Long description to be set for post insights")
	}

	// Long description should mention available metrics
	if len(cmd.Long) < 50 {
		t.Error("expected comprehensive Long description for post insights")
	}
}

func TestInsightsAccountCmd_HasLongDescription(t *testing.T) {
	cmd := insightsAccountCmd

	if cmd.Long == "" {
		t.Error("expected Long description to be set for account insights")
	}

	// Long description should mention available metrics and periods
	if len(cmd.Long) < 50 {
		t.Error("expected comprehensive Long description for account insights")
	}
}
