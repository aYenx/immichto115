package api

import "testing"

func TestNormalizeRemoteDir(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty becomes root", in: "", want: "/"},
		{name: "spaces become root", in: "   ", want: "/"},
		{name: "relative path gets leading slash", in: "immich-backup", want: "/immich-backup"},
		{name: "duplicate slashes are cleaned", in: "//albums///2026//", want: "/albums/2026"},
		{name: "dot path becomes root", in: ".", want: "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeRemoteDir(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeRemoteDir(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
