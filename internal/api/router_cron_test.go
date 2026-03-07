package api

import "testing"

func TestNormalizeCronExpression(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty stays empty", in: "", want: ""},
		{name: "five field cron gets seconds", in: "0 2 * * *", want: "0 0 2 * * *"},
		{name: "five field cron with spaces gets trimmed", in: "  */5 * * * *  ", want: "0 */5 * * * *"},
		{name: "six field cron stays unchanged", in: "0 0 2 * * *", want: "0 0 2 * * *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCronExpression(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeCronExpression(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
