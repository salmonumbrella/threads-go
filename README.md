# Threads CLI

[![Go Reference](https://pkg.go.dev/badge/github.com/salmonumbrella/threads-go.svg)](https://pkg.go.dev/github.com/salmonumbrella/threads-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/salmonumbrella/threads-go)](https://goreportcard.com/report/github.com/salmonumbrella/threads-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A production-ready command-line interface for Meta's Threads API. Built on top of a comprehensive Go client library with full API coverage, OAuth 2.0 authentication, and agent-friendly design.

## Features

- **Full API Coverage**: Posts, replies, users, insights, search, and more
- **OAuth 2.0**: Browser-based authentication with long-lived tokens (60 days)
- **Secure Storage**: Credentials stored in system keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager)
- **Agent-Friendly**: Designed for automation with Claude and other AI assistants
- **Multiple Output Formats**: Text and JSON with JQ filtering support
- **Cross-Platform**: macOS, Linux, and Windows support

## Installation

### From Source

```bash
go install github.com/salmonumbrella/threads-go/cmd/threads@latest
```

### From Releases

Download the latest release for your platform from [GitHub Releases](https://github.com/salmonumbrella/threads-go/releases).

## Quick Start

### 1. Set Up Meta App Credentials

Create a Meta app at [developers.facebook.com](https://developers.facebook.com/) and configure it for Threads API access.

```bash
export THREADS_CLIENT_ID="your-client-id"
export THREADS_CLIENT_SECRET="your-client-secret"
```

### 2. Authenticate

```bash
# Browser-based OAuth flow (recommended)
threads auth login

# Or use an existing token
threads auth token YOUR_ACCESS_TOKEN
```

### 3. Start Using Threads

```bash
# View your profile
threads me

# Create a post
threads posts create --text "Hello from the CLI!"

# List your posts
threads posts list

# Search for content
threads search "machine learning"
```

## Commands

### Authentication

```bash
threads auth login          # Browser OAuth flow
threads auth token TOKEN    # Use existing token
threads auth refresh        # Refresh before expiry
threads auth status         # Show token status
threads auth list           # List accounts
threads auth remove NAME    # Remove account
```

### Posts

```bash
threads posts create --text "Hello!"                    # Text post
threads posts create --text "Check this" --image URL    # Image post
threads posts create --video URL                        # Video post
threads posts carousel --items url1,url2,url3           # Carousel (2-20 items)
threads posts quote POST_ID --text "My take"            # Quote post
threads posts repost POST_ID                            # Repost
threads posts get POST_ID                               # Get post
threads posts list                                      # List posts
threads posts delete POST_ID                            # Delete post
```

### Users

```bash
threads me                      # Your profile
threads users get USER_ID       # Get user by ID
threads users lookup @username  # Lookup public profile
threads users mentions          # Posts mentioning you
```

### Replies

```bash
threads replies list POST_ID                    # List replies
threads replies create POST_ID --text "Reply"   # Reply to post
threads replies hide REPLY_ID                   # Hide reply
threads replies unhide REPLY_ID                 # Unhide reply
threads replies conversation POST_ID            # Full thread
```

### Insights

```bash
threads insights post POST_ID                           # Post analytics
threads insights account                                # Account analytics
threads insights account --metrics views,followers_count
```

### Search

```bash
threads search "query"                           # Search posts
threads search "golang" --limit 10               # With limit
threads search "news" --media-type IMAGE         # Filter by type
threads search "tech" --since 2024-01-01         # Posts after date
```

### Locations

```bash
threads locations search "San Francisco"         # Search by name
threads locations search --lat 37.7 --lng -122.4 # Search by coords
threads locations get LOCATION_ID                # Get details
```

### Rate Limits

```bash
threads ratelimit status        # Current rate limit status
threads ratelimit publishing    # API publishing quota
```

### Shell Completion

```bash
threads completion bash         # Generate bash completions
threads completion zsh          # Generate zsh completions
threads completion fish         # Generate fish completions
threads completion powershell   # Generate PowerShell completions
```

## Global Flags

```
-a, --account string   Account to use (or THREADS_ACCOUNT)
-o, --output string    Output format: text, json (default "text")
-q, --query string     JQ filter for JSON output
-y, --yes              Skip confirmation prompts
    --limit int        Limit results
    --debug            Debug output
```

## Agent-Friendly Design

This CLI is designed for automation with AI assistants:

- **JSON output**: `--output json` for machine-readable responses
- **JQ filtering**: `--query '.data[0].id'` to extract specific fields
- **No prompts**: `--yes` to skip confirmations
- **Structured errors**: Clear error messages for programmatic handling

Example automation:

```bash
# Get post ID from JSON output
POST_ID=$(threads posts create --text "Hello" -o json | jq -r '.id')

# Get insights for the post
threads insights post $POST_ID -o json
```

## Environment Variables

```bash
THREADS_CLIENT_ID       # Meta App Client ID
THREADS_CLIENT_SECRET   # Meta App Client Secret
THREADS_REDIRECT_URI    # OAuth redirect URI
THREADS_ACCESS_TOKEN    # Access token (for token command)
THREADS_ACCOUNT         # Default account name
THREADS_OUTPUT          # Default output format
NO_COLOR                # Disable color output
```

## API Reference

CLI commands map to Threads Graph API endpoints:

| Command | API Endpoint | Description |
|---------|-------------|-------------|
| `threads me` | `GET /me` | Get authenticated user profile |
| `threads users get ID` | `GET /{user-id}` | Get user by ID |
| `threads posts create` | `POST /{user-id}/threads` + `POST /{container-id}/threads_publish` | Create and publish a post |
| `threads posts get ID` | `GET /{post-id}` | Get post details |
| `threads posts list` | `GET /{user-id}/threads` | List user's posts |
| `threads posts delete ID` | `DELETE /{post-id}` | Delete a post |
| `threads posts carousel` | `POST /{user-id}/threads` (media_type=CAROUSEL) | Create carousel post |
| `threads posts quote ID` | `POST /{user-id}/threads` (quoted_post_id) | Quote a post |
| `threads posts repost ID` | `POST /{post-id}/repost` | Repost content |
| `threads replies list ID` | `GET /{post-id}/replies` | Get replies to a post |
| `threads replies create ID` | `POST /{user-id}/threads` (reply_to_id) | Reply to a post |
| `threads replies conversation ID` | `GET /{post-id}/conversation` | Get full conversation thread |
| `threads replies hide ID` | `POST /{reply-id}/manage_reply` (hide=true) | Hide a reply |
| `threads replies unhide ID` | `POST /{reply-id}/manage_reply` (hide=false) | Unhide a reply |
| `threads insights post ID` | `GET /{post-id}/insights` | Get post analytics |
| `threads insights account` | `GET /{user-id}/threads_insights` | Get account analytics |
| `threads search QUERY` | `GET /{user-id}/threads_keyword_search` | Search posts |
| `threads locations search` | `GET /locations_search` | Search locations |
| `threads ratelimit publishing` | `GET /{user-id}/threads_publishing_limit` | Get publishing quota |
| `threads users mentions` | `GET /{user-id}/mentions` | Get posts mentioning you |

Base URL: `https://graph.threads.net`

## Troubleshooting

### Authentication Errors

**"Token expired"**
```bash
# Refresh your token (requires stored client secret)
threads auth refresh

# Or re-authenticate
threads auth login
```

**"Invalid token" or 401 errors**
- Verify your token hasn't been revoked in Meta Developer Console
- Check that your app has the required permissions
- Re-authenticate: `threads auth login`

**"Client ID and secret required"**
```bash
export THREADS_CLIENT_ID="your-app-id"
export THREADS_CLIENT_SECRET="your-app-secret"
```

### Rate Limits

**HTTP 429 or rate limit errors**
```bash
# Check current rate limit status
threads ratelimit status

# Check publishing quota
threads ratelimit publishing
```

The API has these limits (per 24-hour window):
- **Posts**: 250 posts/day
- **Replies**: 1000 replies/day
- **Deletes**: 25 deletes/day

When rate limited, wait for the reset period or reduce request frequency.

### Token Expiry

Long-lived tokens expire after **60 days**. The CLI shows warnings when tokens are expiring soon.

```bash
# Check token status
threads auth status

# Refresh before expiry
threads auth refresh
```

Set up a cron job to auto-refresh:
```bash
# Add to crontab (runs weekly)
0 0 * * 0 threads auth refresh
```

### Common Issues

**"Cannot prompt for confirmation: stdin is not a terminal"**
- Running in a non-interactive context (scripts, CI)
- Use `--yes` flag to skip prompts: `threads posts delete ID --yes`

**"Post validation failed"**
- Text exceeds 500 characters
- Carousel has fewer than 2 or more than 20 items
- Invalid media URL format

**"Container error" or media upload failures**
- Verify media URL is publicly accessible
- Check media format is supported (JPEG, PNG for images; MP4, MOV for videos)
- Video must be under 5 minutes

## Workflow Examples

### Post with Image and Get Insights

```bash
# Create an image post
POST_ID=$(threads posts create \
  --text "Check out this view!" \
  --image "https://example.com/photo.jpg" \
  --alt-text "Mountain sunset" \
  -o json | jq -r '.id')

# Wait for post to be indexed (a few seconds)
sleep 5

# Get post insights
threads insights post $POST_ID
```

### Create a Thread (Self-Replies)

```bash
# First post
POST_ID=$(threads posts create --text "Thread time! 1/3" -o json | jq -r '.id')

# Reply to create thread
threads posts create --text "More context here 2/3" --reply-to $POST_ID
threads posts create --text "And the conclusion 3/3" --reply-to $POST_ID
```

### Monitor Your Mentions

```bash
# Check mentions in JSON for scripting
threads users mentions -o json | jq '.data[] | {from: .username, text: .text}'

# Reply to a mention
threads replies create MENTION_POST_ID --text "Thanks for the mention!"
```

### Bulk Operations with JQ

```bash
# Get all post IDs from last 10 posts
threads posts list --limit 10 -o json | jq -r '.posts[].id'

# Get total views across recent posts
threads posts list --limit 10 -o json -q '[.posts[].id] | length'

# Export posts to CSV
threads posts list -o json | jq -r '.posts[] | [.id, .text, .timestamp] | @csv'
```

### Carousel Post Workflow

```bash
# Create carousel with multiple images
threads posts carousel \
  --items "https://example.com/1.jpg,https://example.com/2.jpg,https://example.com/3.jpg" \
  --text "Photo dump from my trip!" \
  --alt-text "Beach sunset" \
  --alt-text "Mountain view" \
  --alt-text "City skyline"
```

### Scheduled Posting (with cron)

```bash
#!/bin/bash
# save as ~/scripts/scheduled-post.sh

# Post at specific time via cron
threads posts create --text "Good morning! $(date +%A)"

# Check if successful
if [ $? -eq 0 ]; then
  echo "Posted at $(date)" >> ~/threads-posts.log
fi
```

```cron
# Add to crontab: post daily at 9 AM
0 9 * * * ~/scripts/scheduled-post.sh
```

## Go Library

This CLI is built on a comprehensive Go client library. You can also use the library directly:

```go
import threads "github.com/salmonumbrella/threads-go"

client, err := threads.NewClientWithToken("token", &threads.Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
})

// Create a post
post, err := client.CreateTextPost(ctx, &threads.TextPostContent{
    Text: "Hello from Go!",
})
```

See the [Go documentation](https://pkg.go.dev/github.com/salmonumbrella/threads-go) for full library usage.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Credits

- CLI built on [tirthpatell/threads-go](https://github.com/tirthpatell/threads-go) library
- Inspired by [airwallex-cli](https://github.com/salmonumbrella/airwallex-cli) patterns
