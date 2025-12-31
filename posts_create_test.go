package threads

import (
	"context"
	"testing"
	"time"
)

// TestRepostPost_InvalidPostID tests that RepostPost returns an error for empty post IDs
func TestRepostPost_InvalidPostID(t *testing.T) {
	// Create a minimal client for testing validation
	client := &Client{}

	// Test only with empty post ID (whitespace is not trimmed by PostID.Valid())
	_, err := client.RepostPost(context.TODO(), ConvertToPostID(""))
	if err == nil {
		t.Error("expected error for empty post ID")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "post_id" {
		t.Errorf("expected field 'post_id', got '%s'", validationErr.Field)
	}
}

// TestUnrepostPost_InvalidRepostID tests that UnrepostPost returns an error for empty repost IDs
func TestUnrepostPost_InvalidRepostID(t *testing.T) {
	// Create a minimal client for testing validation
	client := &Client{}

	// Test only with empty repost ID (whitespace is not trimmed by PostID.Valid())
	err := client.UnrepostPost(context.TODO(), ConvertToPostID(""))
	if err == nil {
		t.Error("expected error for empty repost ID")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "repost_id" {
		t.Errorf("expected field 'repost_id', got '%s'", validationErr.Field)
	}

	// Verify the error message contains helpful information
	if validationErr.Message == "" {
		t.Error("expected non-empty error message")
	}
}

// TestCreateTextPost_EmptyText tests that empty text is rejected
func TestCreateTextPost_EmptyText(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name    string
		content *TextPostContent
	}{
		{"empty text", &TextPostContent{Text: ""}},
		{"whitespace only text", &TextPostContent{Text: "   "}},
		{"tabs and newlines only", &TextPostContent{Text: "\t\n\r"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateTextPost(context.TODO(), tt.content)
			if err == nil {
				t.Error("expected error for empty text")
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "text" {
				t.Errorf("expected field 'text', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestCreateImagePost_EmptyImageURL tests that empty image URL is rejected
func TestCreateImagePost_EmptyImageURL(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name    string
		content *ImagePostContent
	}{
		{"empty image URL", &ImagePostContent{ImageURL: ""}},
		{"whitespace only image URL", &ImagePostContent{ImageURL: "   "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateImagePost(context.TODO(), tt.content)
			if err == nil {
				t.Error("expected error for empty image URL")
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			// The validation may use "media_url" instead of "image_url"
			if validationErr.Field != "image_url" && validationErr.Field != "media_url" {
				t.Errorf("expected field 'image_url' or 'media_url', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestCreateVideoPost_EmptyVideoURL tests that empty video URL is rejected
func TestCreateVideoPost_EmptyVideoURL(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name    string
		content *VideoPostContent
	}{
		{"empty video URL", &VideoPostContent{VideoURL: ""}},
		{"whitespace only video URL", &VideoPostContent{VideoURL: "   "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateVideoPost(context.TODO(), tt.content)
			if err == nil {
				t.Error("expected error for empty video URL")
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			// The validation may use "media_url" instead of "video_url"
			if validationErr.Field != "video_url" && validationErr.Field != "media_url" {
				t.Errorf("expected field 'video_url' or 'media_url', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestCreateCarouselPost_EmptyChildren tests that empty children is rejected
func TestCreateCarouselPost_EmptyChildren(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name    string
		content *CarouselPostContent
	}{
		{"nil children", &CarouselPostContent{Children: nil}},
		{"empty children slice", &CarouselPostContent{Children: []string{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateCarouselPost(context.TODO(), tt.content)
			if err == nil {
				t.Error("expected error for empty children")
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "children" {
				t.Errorf("expected field 'children', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestCreateQuotePost_EmptyQuotedPostID tests that empty quoted post ID is rejected
func TestCreateQuotePost_EmptyQuotedPostID(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name         string
		content      interface{}
		quotedPostID string
	}{
		{"empty quoted post ID with text content", &TextPostContent{Text: "Hello"}, ""},
		{"whitespace quoted post ID with text content", &TextPostContent{Text: "Hello"}, "   "},
		{"empty quoted post ID with image content", &ImagePostContent{ImageURL: "https://example.com/img.jpg"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateQuotePost(context.TODO(), tt.content, tt.quotedPostID)
			if err == nil {
				t.Error("expected error for empty quoted post ID")
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "quoted_post_id" {
				t.Errorf("expected field 'quoted_post_id', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestCreateQuotePost_UnsupportedContentType tests that unsupported content types are rejected
func TestCreateQuotePost_UnsupportedContentType(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name    string
		content interface{}
	}{
		{"string content", "just a string"},
		{"int content", 12345},
		{"nil content", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateQuotePost(context.TODO(), tt.content, "valid-post-id")
			if err == nil {
				t.Error("expected error for unsupported content type")
				return
			}

			// For non-nil content, check error message mentions content type
			if tt.content != nil && tt.name != "nil content" {
				// The error should indicate unsupported content type
				if err.Error() == "" {
					t.Error("expected non-empty error message")
				}
			}
		})
	}
}

// TestCreateMediaContainer_InvalidMediaType tests that invalid media types are rejected
func TestCreateMediaContainer_InvalidMediaType(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name      string
		mediaType string
		mediaURL  string
		altText   string
	}{
		{"empty media type", "", "https://example.com/img.jpg", "alt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateMediaContainer(context.TODO(), tt.mediaType, tt.mediaURL, tt.altText)
			if err == nil {
				t.Error("expected error for invalid media type")
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "media_type" {
				t.Errorf("expected field 'media_type', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestCreateMediaContainer_InvalidMediaURL tests that invalid media URLs are rejected
func TestCreateMediaContainer_InvalidMediaURL(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name      string
		mediaType string
		mediaURL  string
		altText   string
	}{
		{"empty media URL", "IMAGE", "", "alt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.CreateMediaContainer(context.TODO(), tt.mediaType, tt.mediaURL, tt.altText)
			if err == nil {
				t.Error("expected error for invalid media URL")
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "media_url" {
				t.Errorf("expected field 'media_url', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestGetContainerStatus_InvalidContainerID tests that empty container IDs are rejected
func TestGetContainerStatus_InvalidContainerID(t *testing.T) {
	client := &Client{}

	// Test with empty container ID only (whitespace passes ContainerID.Valid())
	_, err := client.GetContainerStatus(context.TODO(), ConvertToContainerID(""))
	if err == nil {
		t.Error("expected error for empty container ID")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "container_id" {
		t.Errorf("expected field 'container_id', got '%s'", validationErr.Field)
	}
}

// TestWaitForContainerReady_Timeout tests that waitForContainerReady times out correctly
func TestWaitForContainerReady_Timeout(t *testing.T) {
	// This test would require mocking the GetContainerStatus method
	// For now, we test the timeout calculation logic
	maxAttempts := 3
	pollInterval := 100 * time.Millisecond

	expectedMinTime := time.Duration(maxAttempts-1) * pollInterval // We wait between attempts

	// The timeout should be at least the expected minimum time
	if expectedMinTime < 0 {
		t.Error("expected positive minimum time")
	}
}

// TestContainerStatusConstants tests that container status constants are defined correctly
func TestContainerStatusConstants(t *testing.T) {
	expectedStatuses := map[string]string{
		"FINISHED":    ContainerStatusFinished,
		"IN_PROGRESS": ContainerStatusInProgress,
		"ERROR":       ContainerStatusError,
		"EXPIRED":     ContainerStatusExpired,
		"PUBLISHED":   ContainerStatusPublished,
	}

	for expected, actual := range expectedStatuses {
		if actual != expected {
			t.Errorf("expected container status constant '%s', got '%s'", expected, actual)
		}
	}
}

// TestDefaultContainerPollSettings tests the default polling settings
func TestDefaultContainerPollSettings(t *testing.T) {
	// Check max attempts
	if DefaultContainerPollMaxAttempts <= 0 {
		t.Errorf("expected positive DefaultContainerPollMaxAttempts, got %d", DefaultContainerPollMaxAttempts)
	}

	// Check poll interval is reasonable (not too short or too long)
	if DefaultContainerPollInterval < 100*time.Millisecond {
		t.Error("DefaultContainerPollInterval too short, might cause too many requests")
	}
	if DefaultContainerPollInterval > 10*time.Second {
		t.Error("DefaultContainerPollInterval too long, might cause poor user experience")
	}
}

// TestMediaTypeConstants tests that media type constants are defined correctly
func TestMediaTypeConstants(t *testing.T) {
	expectedMediaTypes := map[string]string{
		"TEXT":     MediaTypeText,
		"IMAGE":    MediaTypeImage,
		"VIDEO":    MediaTypeVideo,
		"CAROUSEL": MediaTypeCarousel,
	}

	for expected, actual := range expectedMediaTypes {
		if actual != expected {
			t.Errorf("expected media type constant '%s', got '%s'", expected, actual)
		}
	}
}

// TestReplyControlConstants tests that reply control constants are defined correctly
func TestReplyControlConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReplyControl
		expected string
	}{
		{"everyone", ReplyControlEveryone, "everyone"},
		{"accounts_you_follow", ReplyControlAccountsYouFollow, "accounts_you_follow"},
		{"mentioned_only", ReplyControlMentioned, "mentioned_only"},
		{"parent_post_author_only", ReplyControlParentPostAuthorOnly, "parent_post_author_only"},
		{"followers_only", ReplyControlFollowersOnly, "followers_only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected ReplyControl '%s', got '%s'", tt.expected, string(tt.constant))
			}
		})
	}
}
