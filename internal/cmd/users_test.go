package cmd

import (
	"testing"

	threads "github.com/salmonumbrella/threads-go"
)

func TestUsersCmd_Structure(t *testing.T) {
	// usersCmd is a package-level var
	cmd := usersCmd

	if cmd.Use != "users" {
		t.Errorf("expected Use=users, got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestUsersCmd_Subcommands(t *testing.T) {
	// usersCmd is a package-level var
	cmd := usersCmd

	expectedSubs := map[string]bool{
		"me":       true,
		"get":      true,
		"lookup":   true,
		"mentions": true,
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

func TestUsersMeCmd_Structure(t *testing.T) {
	cmd := usersMeCmd

	if cmd.Use != "me" {
		t.Errorf("expected Use=me, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestUsersGetCmd_Structure(t *testing.T) {
	cmd := usersGetCmd

	if cmd.Use != "get [user-id]" {
		t.Errorf("expected Use='get [user-id]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestUsersLookupCmd_Structure(t *testing.T) {
	cmd := usersLookupCmd

	if cmd.Use != "lookup [username]" {
		t.Errorf("expected Use='lookup [username]', got %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator for exactly 1 arg")
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestUsersLookupCmd_HasLongDescription(t *testing.T) {
	cmd := usersLookupCmd

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestUsersMentionsCmd_Structure(t *testing.T) {
	cmd := newUsersMentionsCmd()

	if cmd.Use != "mentions" {
		t.Errorf("expected Use=mentions, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestUsersMentionsCmd_Flags(t *testing.T) {
	cmd := newUsersMentionsCmd()

	flags := []string{"limit", "cursor"}
	for _, flag := range flags {
		if cmd.Flag(flag) == nil {
			t.Errorf("missing flag: %s", flag)
		}
	}

	// Check default values
	limitFlag := cmd.Flag("limit")
	if limitFlag.DefValue != "25" {
		t.Errorf("expected limit default=25, got %s", limitFlag.DefValue)
	}

	cursorFlag := cmd.Flag("cursor")
	if cursorFlag.DefValue != "" {
		t.Errorf("expected cursor default='', got %s", cursorFlag.DefValue)
	}
}

func TestMeCmd_IsTopLevelAlias(t *testing.T) {
	// meCmd is a package-level var that should be a top-level alias for "users me"
	cmd := meCmd

	if cmd.Use != "me" {
		t.Errorf("expected Use=me, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestUserToMap(t *testing.T) {
	user := &threads.User{
		ID:            "12345",
		Username:      "testuser",
		Name:          "Test User",
		ProfilePicURL: "https://example.com/pic.jpg",
		Biography:     "Hello world",
		IsVerified:    true,
	}

	result := userToMap(user)

	if result["id"] != "12345" {
		t.Errorf("expected id=12345, got %v", result["id"])
	}
	if result["username"] != "testuser" {
		t.Errorf("expected username=testuser, got %v", result["username"])
	}
	if result["name"] != "Test User" {
		t.Errorf("expected name='Test User', got %v", result["name"])
	}
	if result["profile_pic_url"] != "https://example.com/pic.jpg" {
		t.Errorf("expected profile_pic_url to be set, got %v", result["profile_pic_url"])
	}
	if result["biography"] != "Hello world" {
		t.Errorf("expected biography='Hello world', got %v", result["biography"])
	}
	if result["is_verified"] != true {
		t.Errorf("expected is_verified=true, got %v", result["is_verified"])
	}
}

func TestPublicUserToMap(t *testing.T) {
	user := &threads.PublicUser{
		Username:          "publicuser",
		Name:              "Public User",
		ProfilePictureURL: "https://example.com/public.jpg",
		Biography:         "Public bio",
		IsVerified:        false,
		FollowerCount:     1000,
		LikesCount:        500,
		QuotesCount:       50,
		RepliesCount:      100,
		RepostsCount:      75,
		ViewsCount:        10000,
	}

	result := publicUserToMap(user)

	if result["username"] != "publicuser" {
		t.Errorf("expected username=publicuser, got %v", result["username"])
	}
	if result["name"] != "Public User" {
		t.Errorf("expected name='Public User', got %v", result["name"])
	}
	if result["follower_count"] != 1000 {
		t.Errorf("expected follower_count=1000, got %v", result["follower_count"])
	}
	if result["likes_count"] != 500 {
		t.Errorf("expected likes_count=500, got %v", result["likes_count"])
	}
	if result["quotes_count"] != 50 {
		t.Errorf("expected quotes_count=50, got %v", result["quotes_count"])
	}
	if result["replies_count"] != 100 {
		t.Errorf("expected replies_count=100, got %v", result["replies_count"])
	}
	if result["reposts_count"] != 75 {
		t.Errorf("expected reposts_count=75, got %v", result["reposts_count"])
	}
	if result["views_count"] != 10000 {
		t.Errorf("expected views_count=10000, got %v", result["views_count"])
	}
}

func TestUsersCmd_SubcommandCount(t *testing.T) {
	cmd := usersCmd
	subcommands := cmd.Commands()

	expectedCount := 4 // me, get, lookup, mentions
	if len(subcommands) != expectedCount {
		t.Errorf("expected %d subcommands, got %d", expectedCount, len(subcommands))
	}
}
