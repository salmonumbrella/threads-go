package cmd

import (
	"strings"
	"testing"
)

func TestNormalizeIDArg(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		kind      string
		want      string
		wantErr   bool
		errSubstr string
	}{
		{name: "plain", input: "123", kind: "post", want: "123"},
		{name: "hash", input: "#123", kind: "post", want: "123"},
		{name: "prefixed ok", input: "post:123", kind: "post", want: "123"},
		{name: "prefixed ok alias", input: "p:123", kind: "post", want: "123"},
		{name: "prefixed mismatch", input: "reply:123", kind: "post", wantErr: true, errSubstr: "invalid post ID"},
		{name: "missing after colon", input: "post:", kind: "post", wantErr: true, errSubstr: "missing value"},
		{name: "missing", input: "   ", kind: "post", wantErr: true, errSubstr: "missing post ID"},
		{name: "unknown prefix ignored", input: "foo:bar", kind: "post", want: "foo:bar"},
		{name: "url t", input: "https://www.threads.net/t/123", kind: "post", want: "123"},
		{name: "url t other host", input: "https://api.net/t/12345", kind: "post", want: "12345"},
		{name: "url profile post", input: "https://www.threads.net/@alice/post/999", kind: "post", want: "999"},
		{name: "url query id", input: "https://example.com/whatever?id=777", kind: "post", want: "777"},
		// reply_id URL returns kind "reply", so it works when expected kind is "reply"
		{name: "url reply_id ok", input: "https://example.com/x?reply_id=888", kind: "reply", want: "888"},
		// reply_id URL returns kind "reply", which mismatches expected "post"
		{name: "url reply_id mismatch", input: "https://example.com/x?reply_id=888", kind: "post", wantErr: true, errSubstr: "URL is for reply"},
		// generic ?id= param returns no kind assertion, works for any expected kind
		{name: "url generic id reply", input: "https://example.com/x?id=999", kind: "reply", want: "999"},
		// location prefixes
		{name: "location prefix", input: "loc:42", kind: "location", want: "42"},
		{name: "location prefix full", input: "location:42", kind: "location", want: "42"},
		{name: "location prefix l", input: "l:42", kind: "location", want: "42"},
		{name: "location prefix mismatch", input: "post:42", kind: "location", wantErr: true, errSubstr: "invalid location ID"},
		{name: "location hash", input: "#42", kind: "location", want: "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeIDArg(tt.input, tt.kind)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeIDArg() err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errSubstr != "" && err != nil && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("expected error to contain %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if got != tt.want {
				t.Fatalf("normalizeIDArg()=%q want=%q", got, tt.want)
			}
		})
	}
}
