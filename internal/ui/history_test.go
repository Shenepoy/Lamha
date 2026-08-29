package ui

import (
	"testing"
	"time"

	"github.com/lamha-app/lamha/internal/capture"
)

func TestFormatCaptureTime(t *testing.T) {
	now := time.Date(2026, 8, 29, 10, 25, 0, 0, time.Local)
	cases := []struct {
		when time.Time
		want string
	}{
		{now.Add(-20 * time.Second), "Just now"},
		{now.Add(-12 * time.Minute), "12 minutes ago"},
		{time.Date(2026, 8, 29, 9, 4, 0, 0, time.Local), "1 hours ago"},
		{time.Date(2026, 8, 28, 18, 0, 0, 0, time.Local), "16 hours ago"},
		{time.Date(2026, 8, 28, 9, 0, 0, 0, time.Local), "Yesterday"},
		{time.Date(2026, 8, 26, 12, 0, 0, 0, time.Local), "3 days ago"},
		{time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local), "2026-08-20"},
	}
	for _, item := range cases {
		if got := formatCaptureTime(item.when, now); got != item.want {
			t.Fatalf("formatCaptureTime(%v) = %q, want %q", item.when, got, item.want)
		}
	}
}

func TestFormatCaptureMeta(t *testing.T) {
	got := formatCaptureMeta(capture.SavedCapture{Width: 1920, Height: 1080, Bytes: 1258291})
	if got != "1920×1080 • 1.2 MB" {
		t.Fatalf("formatCaptureMeta() = %q", got)
	}
	if got := formatFileSize(340); got != "340 B" {
		t.Fatalf("formatFileSize(340) = %q", got)
	}
	if got := formatFileSize(12 * 1024); got != "12.0 KB" {
		t.Fatalf("formatFileSize(12KiB) = %q", got)
	}
}
