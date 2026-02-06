package cmd

import "github.com/salmonumbrella/threads-cli/internal/api"

func pagingAfter(p api.Paging) string {
	if p.Cursors != nil && p.Cursors.After != "" {
		return p.Cursors.After
	}
	if p.After != "" {
		return p.After
	}
	return ""
}
