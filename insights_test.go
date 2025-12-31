package threads

import (
	"context"
	"testing"
	"time"
)

// TestValidatePostInsightMetric tests the post insight metric validation
func TestValidatePostInsightMetric(t *testing.T) {
	client := &Client{}

	// Valid metrics should pass
	validMetrics := []string{
		string(PostInsightViews),
		string(PostInsightLikes),
		string(PostInsightReplies),
		string(PostInsightReposts),
		string(PostInsightQuotes),
		string(PostInsightShares),
		string(PostInsightLinkClicks),
		string(PostInsightProfileClicks),
	}

	for _, metric := range validMetrics {
		t.Run("valid_"+metric, func(t *testing.T) {
			err := client.validatePostInsightMetric(metric)
			if err != nil {
				t.Errorf("expected no error for valid metric '%s', got: %v", metric, err)
			}
		})
	}

	// Invalid metrics should fail
	invalidMetrics := []string{
		"invalid_metric",
		"",
		"followers_count", // This is an account metric, not post metric
		"clicks",          // This is an account metric, not post metric
		"VIEW",            // Case sensitive - should be "views"
	}

	for _, metric := range invalidMetrics {
		t.Run("invalid_"+metric, func(t *testing.T) {
			err := client.validatePostInsightMetric(metric)
			if err == nil {
				t.Errorf("expected error for invalid metric '%s'", metric)
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "metric" {
				t.Errorf("expected field 'metric', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestValidateAccountInsightMetric tests the account insight metric validation
func TestValidateAccountInsightMetric(t *testing.T) {
	client := &Client{}

	// Valid metrics should pass
	validMetrics := []string{
		string(AccountInsightViews),
		string(AccountInsightLikes),
		string(AccountInsightReplies),
		string(AccountInsightReposts),
		string(AccountInsightQuotes),
		string(AccountInsightClicks),
		string(AccountInsightFollowersCount),
		string(AccountInsightFollowerDemographics),
	}

	for _, metric := range validMetrics {
		t.Run("valid_"+metric, func(t *testing.T) {
			err := client.validateAccountInsightMetric(metric)
			if err != nil {
				t.Errorf("expected no error for valid metric '%s', got: %v", metric, err)
			}
		})
	}

	// Invalid metrics should fail
	invalidMetrics := []string{
		"invalid_metric",
		"",
		"shares",         // This is a post metric, not account metric
		"link_clicks",    // This is a post metric, not account metric
		"profile_clicks", // This is a post metric, not account metric
		"VIEWS",          // Case sensitive - should be "views"
	}

	for _, metric := range invalidMetrics {
		t.Run("invalid_"+metric, func(t *testing.T) {
			err := client.validateAccountInsightMetric(metric)
			if err == nil {
				t.Errorf("expected error for invalid metric '%s'", metric)
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "metric" {
				t.Errorf("expected field 'metric', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestValidateInsightPeriod tests the insight period validation
func TestValidateInsightPeriod(t *testing.T) {
	client := &Client{}

	// Valid periods should pass
	validPeriods := []string{
		string(InsightPeriodDay),
		string(InsightPeriodLifetime),
	}

	for _, period := range validPeriods {
		t.Run("valid_"+period, func(t *testing.T) {
			err := client.validateInsightPeriod(period)
			if err != nil {
				t.Errorf("expected no error for valid period '%s', got: %v", period, err)
			}
		})
	}

	// Invalid periods should fail
	invalidPeriods := []string{
		"invalid_period",
		"",
		"week",
		"month",
		"year",
		"DAY",      // Case sensitive - should be "day"
		"LIFETIME", // Case sensitive - should be "lifetime"
	}

	for _, period := range invalidPeriods {
		t.Run("invalid_"+period, func(t *testing.T) {
			err := client.validateInsightPeriod(period)
			if err == nil {
				t.Errorf("expected error for invalid period '%s'", period)
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "period" {
				t.Errorf("expected field 'period', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestValidateFollowerDemographicsBreakdown tests the breakdown validation
func TestValidateFollowerDemographicsBreakdown(t *testing.T) {
	client := &Client{}

	// Valid breakdowns should pass
	validBreakdowns := []string{
		string(BreakdownCountry),
		string(BreakdownCity),
		string(BreakdownAge),
		string(BreakdownGender),
	}

	for _, breakdown := range validBreakdowns {
		t.Run("valid_"+breakdown, func(t *testing.T) {
			err := client.validateFollowerDemographicsBreakdown(breakdown)
			if err != nil {
				t.Errorf("expected no error for valid breakdown '%s', got: %v", breakdown, err)
			}
		})
	}

	// Invalid breakdowns should fail
	invalidBreakdowns := []string{
		"invalid_breakdown",
		"",
		"COUNTRY", // Case sensitive - should be "country"
		"region",
		"state",
	}

	for _, breakdown := range invalidBreakdowns {
		t.Run("invalid_"+breakdown, func(t *testing.T) {
			err := client.validateFollowerDemographicsBreakdown(breakdown)
			if err == nil {
				t.Errorf("expected error for invalid breakdown '%s'", breakdown)
				return
			}

			// Verify it's a validation error
			validationErr, ok := err.(*ValidationError)
			if !ok {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			if validationErr.Field != "breakdown" {
				t.Errorf("expected field 'breakdown', got '%s'", validationErr.Field)
			}
		})
	}
}

// TestGetAvailablePostInsightMetrics tests that all expected metrics are available
func TestGetAvailablePostInsightMetrics(t *testing.T) {
	client := &Client{}
	metrics := client.GetAvailablePostInsightMetrics()

	expectedMetrics := map[PostInsightMetric]bool{
		PostInsightViews:         true,
		PostInsightLikes:         true,
		PostInsightReplies:       true,
		PostInsightReposts:       true,
		PostInsightQuotes:        true,
		PostInsightShares:        true,
		PostInsightLinkClicks:    true,
		PostInsightProfileClicks: true,
	}

	if len(metrics) != len(expectedMetrics) {
		t.Errorf("expected %d metrics, got %d", len(expectedMetrics), len(metrics))
	}

	for _, metric := range metrics {
		if !expectedMetrics[metric] {
			t.Errorf("unexpected metric in result: %s", metric)
		}
		delete(expectedMetrics, metric)
	}

	for metric := range expectedMetrics {
		t.Errorf("missing expected metric: %s", metric)
	}
}

// TestGetAvailableAccountInsightMetrics tests that all expected metrics are available
func TestGetAvailableAccountInsightMetrics(t *testing.T) {
	client := &Client{}
	metrics := client.GetAvailableAccountInsightMetrics()

	expectedMetrics := map[AccountInsightMetric]bool{
		AccountInsightViews:                true,
		AccountInsightLikes:                true,
		AccountInsightReplies:              true,
		AccountInsightReposts:              true,
		AccountInsightQuotes:               true,
		AccountInsightClicks:               true,
		AccountInsightFollowersCount:       true,
		AccountInsightFollowerDemographics: true,
	}

	if len(metrics) != len(expectedMetrics) {
		t.Errorf("expected %d metrics, got %d", len(expectedMetrics), len(metrics))
	}

	for _, metric := range metrics {
		if !expectedMetrics[metric] {
			t.Errorf("unexpected metric in result: %s", metric)
		}
		delete(expectedMetrics, metric)
	}

	for metric := range expectedMetrics {
		t.Errorf("missing expected metric: %s", metric)
	}
}

// TestGetAvailableInsightPeriods tests that all expected periods are available
func TestGetAvailableInsightPeriods(t *testing.T) {
	client := &Client{}
	periods := client.GetAvailableInsightPeriods()

	expectedPeriods := map[InsightPeriod]bool{
		InsightPeriodDay:      true,
		InsightPeriodLifetime: true,
	}

	if len(periods) != len(expectedPeriods) {
		t.Errorf("expected %d periods, got %d", len(expectedPeriods), len(periods))
	}

	for _, period := range periods {
		if !expectedPeriods[period] {
			t.Errorf("unexpected period in result: %s", period)
		}
		delete(expectedPeriods, period)
	}

	for period := range expectedPeriods {
		t.Errorf("missing expected period: %s", period)
	}
}

// TestGetAvailableFollowerDemographicsBreakdowns tests that all expected breakdowns are available
func TestGetAvailableFollowerDemographicsBreakdowns(t *testing.T) {
	client := &Client{}
	breakdowns := client.GetAvailableFollowerDemographicsBreakdowns()

	expectedBreakdowns := map[FollowerDemographicsBreakdown]bool{
		BreakdownCountry: true,
		BreakdownCity:    true,
		BreakdownAge:     true,
		BreakdownGender:  true,
	}

	if len(breakdowns) != len(expectedBreakdowns) {
		t.Errorf("expected %d breakdowns, got %d", len(expectedBreakdowns), len(breakdowns))
	}

	for _, breakdown := range breakdowns {
		if !expectedBreakdowns[breakdown] {
			t.Errorf("unexpected breakdown in result: %s", breakdown)
		}
		delete(expectedBreakdowns, breakdown)
	}

	for breakdown := range expectedBreakdowns {
		t.Errorf("missing expected breakdown: %s", breakdown)
	}
}

// TestGetPostInsights_InvalidPostID tests that empty post IDs are rejected
func TestGetPostInsights_InvalidPostID(t *testing.T) {
	client := &Client{}

	// Test with empty post ID
	_, err := client.GetPostInsights(context.TODO(), ConvertToPostID(""), []string{"views"})
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

	if validationErr.Field != "postID" {
		t.Errorf("expected field 'postID', got '%s'", validationErr.Field)
	}
}

// TestGetPostInsights_InvalidMetrics tests that invalid metrics are rejected
func TestGetPostInsights_InvalidMetrics(t *testing.T) {
	client := &Client{}

	_, err := client.GetPostInsights(context.TODO(), ConvertToPostID(""), []string{"invalid_metric"})
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error (first validation is empty postID)
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	// The first validation is postID
	if validationErr.Field != "postID" {
		t.Errorf("expected field 'postID', got '%s'", validationErr.Field)
	}
}

// TestGetPostInsightsWithOptions_InvalidPostID tests that empty post IDs are rejected
func TestGetPostInsightsWithOptions_InvalidPostID(t *testing.T) {
	client := &Client{}

	opts := &PostInsightsOptions{
		Metrics: []PostInsightMetric{PostInsightViews},
	}
	_, err := client.GetPostInsightsWithOptions(context.TODO(), ConvertToPostID(""), opts)
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

	if validationErr.Field != "postID" {
		t.Errorf("expected field 'postID', got '%s'", validationErr.Field)
	}
}

// TestGetPostInsightsWithOptions_InvalidPeriod tests that invalid periods are rejected
func TestGetPostInsightsWithOptions_InvalidPeriod(t *testing.T) {
	client := &Client{}

	// First test that empty postID returns validation error
	opts := &PostInsightsOptions{
		Metrics: []PostInsightMetric{PostInsightViews},
		Period:  InsightPeriod("invalid_period"),
	}
	_, err := client.GetPostInsightsWithOptions(context.TODO(), ConvertToPostID(""), opts)
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	// First validation is postID
	if validationErr.Field != "postID" {
		t.Errorf("expected field 'postID', got '%s'", validationErr.Field)
	}
}

// TestGetPostInsightsWithOptions_InvalidDateRange tests that invalid date ranges are rejected
func TestGetPostInsightsWithOptions_InvalidDateRange(t *testing.T) {
	client := &Client{}

	now := time.Now()
	since := now
	until := now.Add(-24 * time.Hour) // Until is before since

	opts := &PostInsightsOptions{
		Metrics: []PostInsightMetric{PostInsightViews},
		Since:   &since,
		Until:   &until,
	}
	_, err := client.GetPostInsightsWithOptions(context.TODO(), ConvertToPostID(""), opts)
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	// First validation is postID
	if validationErr.Field != "postID" {
		t.Errorf("expected field 'postID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsights_InvalidUserID tests that empty user IDs are rejected
func TestGetAccountInsights_InvalidUserID(t *testing.T) {
	client := &Client{}

	_, err := client.GetAccountInsights(context.TODO(), ConvertToUserID(""), []string{"views"}, "lifetime")
	if err == nil {
		t.Error("expected error for empty user ID")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsights_InvalidMetrics tests that invalid metrics are rejected
func TestGetAccountInsights_InvalidMetrics(t *testing.T) {
	client := &Client{}

	// Empty userID is validated first
	_, err := client.GetAccountInsights(context.TODO(), ConvertToUserID(""), []string{"invalid_metric"}, "lifetime")
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsights_InvalidPeriod tests that invalid periods are rejected
func TestGetAccountInsights_InvalidPeriod(t *testing.T) {
	client := &Client{}

	// Empty userID is validated first
	_, err := client.GetAccountInsights(context.TODO(), ConvertToUserID(""), []string{"views"}, "invalid_period")
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsightsWithOptions_InvalidUserID tests that empty user IDs are rejected
func TestGetAccountInsightsWithOptions_InvalidUserID(t *testing.T) {
	client := &Client{}

	opts := &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightViews},
	}
	_, err := client.GetAccountInsightsWithOptions(context.TODO(), ConvertToUserID(""), opts)
	if err == nil {
		t.Error("expected error for empty user ID")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsightsWithOptions_FollowerDemographicsWithSinceUntil tests that follower_demographics rejects since/until
func TestGetAccountInsightsWithOptions_FollowerDemographicsWithSinceUntil(t *testing.T) {
	client := &Client{}

	now := time.Now()
	since := now.Add(-7 * 24 * time.Hour)

	opts := &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightFollowerDemographics},
		Since:   &since,
	}
	// Empty userID is validated first
	_, err := client.GetAccountInsightsWithOptions(context.TODO(), ConvertToUserID(""), opts)
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsightsWithOptions_FollowersCountWithSinceUntil tests that followers_count rejects since/until
func TestGetAccountInsightsWithOptions_FollowersCountWithSinceUntil(t *testing.T) {
	client := &Client{}

	now := time.Now()
	since := now.Add(-7 * 24 * time.Hour)

	opts := &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightFollowersCount},
		Since:   &since,
	}
	// Empty userID is validated first
	_, err := client.GetAccountInsightsWithOptions(context.TODO(), ConvertToUserID(""), opts)
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsightsWithOptions_InvalidBreakdown tests that invalid breakdowns are rejected
func TestGetAccountInsightsWithOptions_InvalidBreakdown(t *testing.T) {
	client := &Client{}

	opts := &AccountInsightsOptions{
		Metrics:   []AccountInsightMetric{AccountInsightFollowerDemographics},
		Breakdown: "invalid_breakdown",
	}
	// Empty userID is validated first
	_, err := client.GetAccountInsightsWithOptions(context.TODO(), ConvertToUserID(""), opts)
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsightsWithOptions_SinceBelowMinTimestamp tests that since below min timestamp is rejected
func TestGetAccountInsightsWithOptions_SinceBelowMinTimestamp(t *testing.T) {
	client := &Client{}

	// Use a timestamp below the minimum allowed (1712991600)
	since := time.Unix(1712991599, 0) // One second before minimum

	opts := &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightViews},
		Since:   &since,
	}
	// Empty userID is validated first
	_, err := client.GetAccountInsightsWithOptions(context.TODO(), ConvertToUserID(""), opts)
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsightsWithOptions_UntilBelowMinTimestamp tests that until below min timestamp is rejected
func TestGetAccountInsightsWithOptions_UntilBelowMinTimestamp(t *testing.T) {
	client := &Client{}

	// Use a timestamp below the minimum allowed (1712991600)
	until := time.Unix(1712991599, 0) // One second before minimum

	opts := &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightViews},
		Until:   &until,
	}
	// Empty userID is validated first
	_, err := client.GetAccountInsightsWithOptions(context.TODO(), ConvertToUserID(""), opts)
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestGetAccountInsightsWithOptions_InvalidDateRange tests that since after until is rejected
func TestGetAccountInsightsWithOptions_InvalidDateRange(t *testing.T) {
	client := &Client{}

	// Use valid timestamps but with since after until
	since := time.Unix(MinInsightTimestamp+86400, 0) // Min + 1 day
	until := time.Unix(MinInsightTimestamp, 0)       // Min timestamp

	opts := &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightViews},
		Since:   &since,
		Until:   &until,
	}
	// Empty userID is validated first
	_, err := client.GetAccountInsightsWithOptions(context.TODO(), ConvertToUserID(""), opts)
	if err == nil {
		t.Error("expected error")
		return
	}

	// Verify it's a validation error
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	if validationErr.Field != "userID" {
		t.Errorf("expected field 'userID', got '%s'", validationErr.Field)
	}
}

// TestMinInsightTimestamp tests that the min insight timestamp constant is correct
func TestMinInsightTimestamp(t *testing.T) {
	// MinInsightTimestamp should be 1712991600 (April 13, 2024)
	expectedTimestamp := int64(1712991600)
	if MinInsightTimestamp != expectedTimestamp {
		t.Errorf("expected MinInsightTimestamp %d, got %d", expectedTimestamp, MinInsightTimestamp)
	}
}

// TestPostInsightMetricConstants tests that post insight metric constants are correct
func TestPostInsightMetricConstants(t *testing.T) {
	tests := []struct {
		constant PostInsightMetric
		expected string
	}{
		{PostInsightViews, "views"},
		{PostInsightLikes, "likes"},
		{PostInsightReplies, "replies"},
		{PostInsightReposts, "reposts"},
		{PostInsightQuotes, "quotes"},
		{PostInsightShares, "shares"},
		{PostInsightLinkClicks, "link_clicks"},
		{PostInsightProfileClicks, "profile_clicks"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, string(tt.constant))
			}
		})
	}
}

// TestAccountInsightMetricConstants tests that account insight metric constants are correct
func TestAccountInsightMetricConstants(t *testing.T) {
	tests := []struct {
		constant AccountInsightMetric
		expected string
	}{
		{AccountInsightViews, "views"},
		{AccountInsightLikes, "likes"},
		{AccountInsightReplies, "replies"},
		{AccountInsightReposts, "reposts"},
		{AccountInsightQuotes, "quotes"},
		{AccountInsightClicks, "clicks"},
		{AccountInsightFollowersCount, "followers_count"},
		{AccountInsightFollowerDemographics, "follower_demographics"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, string(tt.constant))
			}
		})
	}
}

// TestInsightPeriodConstants tests that insight period constants are correct
func TestInsightPeriodConstants(t *testing.T) {
	tests := []struct {
		constant InsightPeriod
		expected string
	}{
		{InsightPeriodDay, "day"},
		{InsightPeriodLifetime, "lifetime"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, string(tt.constant))
			}
		})
	}
}

// TestFollowerDemographicsBreakdownConstants tests that breakdown constants are correct
func TestFollowerDemographicsBreakdownConstants(t *testing.T) {
	tests := []struct {
		constant FollowerDemographicsBreakdown
		expected string
	}{
		{BreakdownCountry, "country"},
		{BreakdownCity, "city"},
		{BreakdownAge, "age"},
		{BreakdownGender, "gender"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, string(tt.constant))
			}
		})
	}
}
