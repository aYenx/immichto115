package rclone

import "testing"

func TestIsProgressNoise(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"", true},
		{"   ", true},
		{"Transferred: 0 B / 0 B, -, 0 B/s, ETA -", true},
		{"Checks: 12 / 12, 100%", true},
		{"Elapsed time: 1.2s", true},
		{"Transferring: ...", true},
		{"2026/01/01 10:00:00 INFO : file.jpg: Copied (new)", false},
		{"ERROR : can't read file", false},
		{"[immichto115] 已启动 rclone copy", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := isProgressNoise(tt.line)
			if got != tt.want {
				t.Fatalf("isProgressNoise(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestRunnerIsRunning_InitiallyFalse(t *testing.T) {
	r := NewRunner()
	if r.IsRunning() {
		t.Fatal("expected IsRunning() to be false initially")
	}
}

func TestRunnerStop_NoProcess(t *testing.T) {
	r := NewRunner()
	err := r.Stop()
	if err == nil {
		t.Fatal("expected error when stopping without running process")
	}
}
