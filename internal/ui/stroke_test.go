package ui

import "testing"

func TestStrokePreviewGrowsWithWidth(t *testing.T) {
	_, thin := strokePreviewSize(4)
	_, mid := strokePreviewSize(24)
	wide, thick := strokePreviewSize(48)
	if mid <= thin || thick <= mid {
		t.Fatalf("preview height should grow with stroke: %d %d %d", thin, mid, thick)
	}
	if thick < 48 {
		t.Fatalf("preview height %d should fit a 48px stroke", thick)
	}
	if wide <= 48 {
		t.Fatalf("preview width %d should stay longer than the stroke so it reads as a line", wide)
	}
}
