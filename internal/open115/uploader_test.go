package open115

import (
	"errors"
	"testing"
)

func TestIsRateLimitedError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "known refresh frequently message",
			err:  errors.New("upload failed: refresh frequently"),
			want: true,
		},
		{
			name: "known 40140117 code",
			err:  errors.New("request failed: 40140117"),
			want: true,
		},
		{
			name: "wrapped empty message marker",
			err:  errors.New("sdk call failed: code: 0, message:"),
			want: true,
		},
		{
			name: "code 0 but non-empty message is not rate limit",
			err:  errors.New("sdk call failed: code: 0, message: invalid token"),
			want: false,
		},
		{
			name: "ordinary error",
			err:  errors.New("network timeout"),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsRateLimitedError(tc.err)
			if got != tc.want {
				t.Fatalf("IsRateLimitedError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
