package cli

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0ms"},
		{"milliseconds", 523 * time.Millisecond, "523ms"},
		{"sub-second", 999 * time.Millisecond, "999ms"},
		{"one second", time.Second, "1s"},
		{"seconds", 45 * time.Second, "45s"},
		{"one minute", time.Minute, "1m"},
		{"minutes and seconds", time.Minute + 32*time.Second, "1m32s"},
		{"exact minutes", 2 * time.Minute, "2m"},
		{"large", 5*time.Minute + 12*time.Second, "5m12s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.d)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
