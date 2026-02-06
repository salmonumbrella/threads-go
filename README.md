# Threads CLI - Social media in your terminal.

Threads in your terminal. Create posts, manage replies, view insights, search content, and automate your Threads presence.

[![Go Reference](https://pkg.go.dev/badge/github.com/salmonumbrella/threads-cli.svg)](https://pkg.go.dev/github.com/salmonumbrella/threads-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/salmonumbrella/threads-cli)](https://goreportcard.com/report/github.com/salmonumbrella/threads-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Authentication** - OAuth 2.0 with long-lived tokens (60 days), auto-refresh
- **Posts** - create text, image, video, carousel, quote posts, and reposts
- **Replies** - list, create, hide/unhide replies, view conversation threads
- **Users** - view profiles, lookup by username, check mentions
- **Insights** - post and account analytics with customizable metrics
- **Search** - keyword search with date and media type filters
- **Locations** - search by name or coordinates
- **Multiple accounts** - manage multiple Threads accounts
- **Agent-friendly** - JSON output, JQ filtering, no-prompt mode for automation

## Installation

### Homebrew

```bash
brew install salmonumbrella/tap/threads-cli
```

### From Source

```bash
go install github.com/salmonumbrella/threads-cli/cmd/threads@latest
```

### From Releases

Download the latest release for your platform from [GitHub Releases](https://github.com/salmonumbrella/threads-cli/releases).

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

### 3. Test Authentication

```bash
threads auth status
```

## Configuration

Threads CLI supports a local config file. See the current path with:

```bash
threads config path
```

Common config commands:

```bash
threads config list
threads config get output
threads config set output json
threads config set color always
```

### Account Selection

Specify the account using either a flag or environment variable:

```bash
# Via flag
threads posts list --account my-account

# Via environment
export THREADS_ACCOUNT=my-account
threads posts list
```

### Environment Variables

- `THREADS_CLIENT_ID` - Meta App Client ID
- `THREADS_CLIENT_SECRET` - Meta App Client Secret
- `THREADS_REDIRECT_URI` - OAuth redirect URI (optional)
- `THREADS_ACCESS_TOKEN` - Access token (for token command)
- `THREADS_ACCOUNT` - Default account name to use
- `THREADS_OUTPUT` - Output format: `text` (default) or `json`
- `THREADS_COLOR` - Color output: `auto` (default), `always`, `never`
- `THREADS_DEBUG` - Enable debug logging (true/false)
- `THREADS_CONFIG` - Path to config file (overrides default location)
- `NO_COLOR` - Set to any value to disable colors

## Security

### Credential Storage

Credentials are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

## Rate Limiting

The Threads API enforces rate limits per 24-hour window:
- **Posts**: 250 posts/day
- **Replies**: 1000 replies/day
- **Deletes**: 25 deletes/day

Check your current limits:

```bash
threads ratelimit status        # Current rate limit status
threads ratelimit publishing    # API publishing quota
```

When rate limited, wait for the reset period or reduce request frequency.

## Commands

### Authentication

```bash
threads auth login                     # Browser OAuth flow (recommended)
threads auth token TOKEN               # Use existing token
threads auth refresh                   # Refresh before expiry
threads auth status                    # Show token status
threads auth list                      # List configured accounts
threads auth remove NAME               # Remove account
```

### Posts

```bash
threads posts create --text "Hello!"                    # Text post
threads posts create --text "Check this" --image URL    # Image post
threads posts create --video URL                        # Video post
threads posts carousel --items url1,url2,url3           # Carousel (2-20 items)
threads posts quote POST_ID --text "My take"            # Quote post
threads posts repost POST_ID                            # Repost
threads posts get POST_ID                               # Get post details
threads posts list                                      # List your posts
threads posts delete POST_ID                            # Delete post
```

### Users

```bash
threads me                             # Your profile
threads users get USER_ID              # Get user by ID
threads users lookup @username         # Lookup public profile
threads users mentions                 # Posts mentioning you
```

### Replies

```bash
threads replies list POST_ID                    # List replies to a post
threads replies create POST_ID --text "Reply"   # Reply to post
threads replies hide REPLY_ID                   # Hide reply
threads replies unhide REPLY_ID                 # Unhide reply
threads replies conversation POST_ID            # Full conversation thread
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
threads locations search --lat 37.7 --lng -122.4 # Search by coordinates
threads locations get LOCATION_ID                # Get location details
```

## Output Formats

### Text

Human-readable output with colors and formatting:

```bash
$ threads me
Username: @johndoe
Followers: 1,234
Following: 567

$ threads posts list
ID                    TEXT                           TIMESTAMP
1234567890123456789   Hello from the CLI!            2024-01-15 10:30
9876543210987654321   Check out this photo...        2024-01-14 15:45
```

### JSON

Machine-readable output:

```bash
$ threads me --output json
{
  "id": "1234567890",
  "username": "johndoe",
  "threads_profile_picture_url": "https://...",
  "threads_biography": "..."
}
```

Data goes to stdout, errors and progress to stderr for clean piping.

## Examples

### Post with Image and Get Insights

```bash
# Create an image post
POST_ID=$(threads posts create \
  --text "Check out this view!" \
  --image "https://example.com/photo.jpg" \
  --alt-text "Mountain sunset" \
  -o json | jq -r '.id')

# Wait for post to be indexed
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

### Automation

Use `--yes` to skip confirmations and `--limit` to control result size:

```bash
# Delete a post without confirmation prompt
threads posts delete POST_ID --yes

# Get the 10 most recent posts
threads posts list --limit 10 --output json

# Pipeline: get all post IDs from last 10 posts
threads posts list --limit 10 -o json | jq -r '.posts[].id'

# Export posts to CSV
threads posts list -o json | jq -r '.posts[] | [.id, .text, .timestamp] | @csv'
```

### Switch Between Accounts

```bash
# Check primary account
threads posts list --account personal

# Check business account
threads posts list --account business

# Or set default
export THREADS_ACCOUNT=personal
threads posts list
```

### JQ Filtering

Filter JSON output with JQ expressions:

```bash
# Get only the first post ID
threads posts list --output json --query '.posts[0].id'

# Extract all post texts
threads posts list --output json --query '[.posts[].text]'

# Filter posts with images
threads posts list --output json --query '.posts[] | select(.media_type=="IMAGE")'
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

### Token Refresh Automation

Long-lived tokens expire after 60 days. Set up auto-refresh:

```bash
# Check token status
threads auth status

# Refresh before expiry
threads auth refresh
```

```cron
# Add to crontab (runs weekly)
0 0 * * 0 threads auth refresh
```

## Global Flags

All commands support these flags:

- `--account <name>`, `-a` - Account to use (overrides THREADS_ACCOUNT)
- `--output <format>`, `-o` - Output format: `text`, `json`, or `jsonl` (default: text)
- `--json` - Shortcut for `--output json`
- `--query <expr>`, `-q` - JQ filter expression for structured output (`json`/`jsonl`)
- `--yes`, `-y` - Skip confirmation prompts (useful for scripts and automation)
- `--no-prompt` - Alias for `--yes`
- `--color <mode>` - Color output: `auto`, `always`, `never`
- `--no-color` - Shortcut for `--color never`
- `--debug` - Enable debug output
- `--help` - Show help for any command
- `--version` - Show version information

Note: many list-style commands also support `--limit` (and sometimes `--cursor`) as command-specific flags.

Tip: `--output jsonl` is useful for list-style commands, emitting one JSON object per line for easy streaming and piping.

Auto-pagination:

```bash
# Stream all pages (one JSON object per line)
threads posts list --all -o jsonl

# Stream all search results
threads search "coffee" --all -o jsonl
```

## Agent Discovery

To make agents and scripts more reliable, `threads` includes a JSON help command:

```bash
threads help-json
threads help-json posts get
```

IDs also accept common shorthands like `#123` and `post:123` (depending on the command), and most post-ID arguments accept pasted permalinks like `https://www.threads.net/t/<id>`.

Agent-friendly “best match” search:

```bash
# Emit just an ID (easy to chain, no jq)
threads search "coffee" --best --emit id
```

Convenience: `threads users get @username` delegates to `threads users lookup username`.

## Shell Completions

Generate shell completions for your preferred shell:

### Bash

```bash
# macOS (Homebrew):
threads completion bash > $(brew --prefix)/etc/bash_completion.d/threads

# Linux:
threads completion bash > /etc/bash_completion.d/threads

# Or source directly in current session:
source <(threads completion bash)
```

### Zsh

```zsh
# Save to fpath:
threads completion zsh > "${fpath[1]}/_threads"

# Or add to .zshrc for auto-loading:
echo 'autoload -U compinit; compinit' >> ~/.zshrc
echo 'source <(threads completion zsh)' >> ~/.zshrc
```

### Fish

```fish
threads completion fish > ~/.config/fish/completions/threads.fish
```

### PowerShell

```powershell
# Load for current session:
threads completion powershell | Out-String | Invoke-Expression

# Or add to profile for persistence:
threads completion powershell >> $PROFILE
```

## API Reference

CLI commands map to Threads Graph API endpoints:

| Command | API Endpoint |
|---------|-------------|
| `threads me` | `GET /me` |
| `threads users get ID` | `GET /{user-id}` |
| `threads posts create` | `POST /{user-id}/threads` + `POST /{container-id}/threads_publish` |
| `threads posts get ID` | `GET /{post-id}` |
| `threads posts list` | `GET /{user-id}/threads` |
| `threads posts delete ID` | `DELETE /{post-id}` |
| `threads replies list ID` | `GET /{post-id}/replies` |
| `threads replies create ID` | `POST /{user-id}/threads` (reply_to_id) |
| `threads insights post ID` | `GET /{post-id}/insights` |
| `threads insights account` | `GET /{user-id}/threads_insights` |
| `threads search QUERY` | `GET /{user-id}/threads_keyword_search` |
| `threads locations search` | `GET /locations_search` |
| `threads ratelimit publishing` | `GET /{user-id}/threads_publishing_limit` |
| `threads users mentions` | `GET /{user-id}/mentions` |

Base URL: `https://graph.threads.net`

## Troubleshooting

### Authentication Errors

**"Token expired"**
```bash
threads auth refresh  # Requires stored client secret
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

## Go Library

This CLI is built on a comprehensive Go client library:

```go
import "github.com/salmonumbrella/threads-cli/internal/api"

client, err := api.NewClientWithToken("token", &api.Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
})

// Create a post
post, err := client.CreateTextPost(ctx, &api.TextPostContent{
    Text: "Hello from Go!",
})
```

See the [Go documentation](https://pkg.go.dev/github.com/salmonumbrella/threads-cli) for full library usage.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT

## Links

- [Threads API Documentation](https://developers.facebook.com/docs/threads)
- [Go Package Documentation](https://pkg.go.dev/github.com/salmonumbrella/threads-cli)
- [GitHub Repository](https://github.com/salmonumbrella/threads-cli)
