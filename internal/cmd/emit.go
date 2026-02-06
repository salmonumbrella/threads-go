package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/salmonumbrella/threads-cli/internal/iocontext"
	"github.com/salmonumbrella/threads-cli/internal/outfmt"
)

type emitMode string

const (
	emitJSON emitMode = "json"
	emitID   emitMode = "id"
	emitURL  emitMode = "url"
)

func parseEmitMode(raw string) (emitMode, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch emitMode(v) {
	case emitJSON, emitID, emitURL:
		return emitMode(v), nil
	default:
		return "", &UserFriendlyError{
			Message:    fmt.Sprintf("Invalid --emit value: %s", raw),
			Suggestion: "Valid values are: json, id, url",
		}
	}
}

// emitResult emits a single command result in the requested shape.
//
// Intended for "create style" commands so agents can chain without jq.
func emitResult(ctx context.Context, io *iocontext.IO, mode emitMode, id string, url string, item any) error {
	// Machine output: keep a stable JSON wrapper for scalars.
	if outfmt.IsJSON(ctx) {
		out := outfmt.FromContext(ctx, outfmt.WithWriter(io.Out))
		switch mode {
		case emitID:
			return out.Output(map[string]any{"id": id})
		case emitURL:
			if strings.TrimSpace(url) == "" {
				return &UserFriendlyError{
					Message:    "Cannot emit url: permalink is empty",
					Suggestion: "Use --emit id or --emit json",
				}
			}
			return out.Output(map[string]any{"url": url})
		default:
			return out.Output(item)
		}
	}

	// Text output: print scalars as one-liners for easy chaining.
	switch mode {
	case emitID:
		fmt.Fprintln(io.Out, id) //nolint:errcheck // Best-effort output
		return nil
	case emitURL:
		if strings.TrimSpace(url) == "" {
			return &UserFriendlyError{
				Message:    "Cannot emit url: permalink is empty",
				Suggestion: "Use --emit id or --emit json",
			}
		}
		fmt.Fprintln(io.Out, url) //nolint:errcheck // Best-effort output
		return nil
	default:
		// Explicit JSON emit in text mode: output JSON to stdout.
		enc := json.NewEncoder(io.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(item)
	}
}
