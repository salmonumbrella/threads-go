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
threads posts get POST_ID                               # Get post
threads posts list                                      # List posts
threads posts delete POST_ID                            # Delete post
```

### Users

```bash
threads me                      # Your profile
threads users get USER_ID       # Get user by ID
threads users lookup @username  # Lookup public profile
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
threads search "query"              # Search posts
threads search "golang" --limit 10  # With limit
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
