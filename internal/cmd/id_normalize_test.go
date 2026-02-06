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
