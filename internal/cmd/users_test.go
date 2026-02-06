package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/salmonumbrella/threads-cli/internal/api"
	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

func TestUsersCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewUsersCmd(f)

	if cmd.Use != "users" {
		t.Errorf("expected Use=users, got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestUsersCmd_Subcommands(t *testing.T) {
	f := newTestFactory(t)
	cmd := NewUsersCmd(f)

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
	f := newTestFactory(t)
	cmd := newUsersMeCmd(f)

	if cmd.Use != "me" {
		t.Errorf("expected Use=me, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestUsersGetCmd_Structure(t *testing.T) {
	f := newTestFactory(t)
	cmd := newUsersGetCmd(f)

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
	f := newTestFactory(t)
	cmd := newUsersLookupCmd(f)

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
	f := newTestFactory(t)
	cmd := newUsersLookupCmd(f)

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestUsersMentionsCmd_Flags(t *testing.T) {
	f := newTestFactory(t)
	cmd := newUsersMentionsCmd(f)

	flags := []string{"limit", "cursor", "all", "no-hints"}
	for _, flag := range flags {
		if cmd.Flag(flag) == nil {
			t.Errorf("missing flag: %s", flag)
		}
	}

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
	f := newTestFactory(t)
	cmd := NewUsersMeCmd(f)

	if cmd.Use != "me" {
		t.Errorf("expected Use=me, got %s", cmd.Use)
	}

	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestUserToMap(t *testing.T) {
	user := &api.User{
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
	user := &api.PublicUser{
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

func TestUsersGet_AcceptsUsernameAndProfileURL(t *testing.T) {
	lookupCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/refresh_access_token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "refreshed-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}

		if r.URL.Path == "/profile_lookup" {
			lookupCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"username":            r.URL.Query().Get("username"),
				"name":                "Public User",
				"profile_picture_url": "https://example.com/public.jpg",
				"biography":           "Bio",
				"is_verified":         false,
				"follower_count":      1,
				"likes_count":         2,
				"quotes_count":        3,
				"replies_count":       4,
				"reposts_count":       5,
				"views_count":         6,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	f, io := newIntegrationTestFactory(t, server.URL)

	ctx := context.Background()
	ctx = iocontext.WithIO(ctx, io)
	ctx = outfmt.WithFormat(ctx, "json")

	cmd := newUsersGetCmd(f)
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"@publicuser"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("users get @username failed: %v", err)
	}

	if lookupCalls != 1 {
		t.Fatalf("expected 1 lookup call, got %d", lookupCalls)
	}

	// Reset buffers and call with URL form.
	io.Out.(*bytes.Buffer).Reset()
	io.ErrOut.(*bytes.Buffer).Reset()

	cmd2 := newUsersGetCmd(f)
	cmd2.SetContext(ctx)
	cmd2.SetArgs([]string{"https://www.threads.net/@publicuser"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("users get profile URL failed: %v", err)
	}

	if lookupCalls != 2 {
		t.Fatalf("expected 2 lookup calls, got %d", lookupCalls)
	}
}
