package cmd

func itemsEnvelope(items any, paging any, nextCursor string) map[string]any {
	hasMore := nextCursor != ""
	out := map[string]any{
		"items":    items,
		"has_more": hasMore,
	}
	if nextCursor != "" {
		out["cursor"] = nextCursor
	}
	if paging != nil {
		out["paging"] = paging
	}
	return out
}
