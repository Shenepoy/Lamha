package ui

import (
	"image"
	"testing"

	"github.com/lamha-app/lamha/internal/annotate"
)

func TestSelectionHitCornersAndMove(t *testing.T) {
	sel := image.Rect(40, 40, 200, 160)
	if got := selectionHit(sel, annotate.Point{X: 40, Y: 40}, 1); got != selHandleNW {
		t.Fatalf("corner hit = %d", got)
	}
	if got := selectionHit(sel, annotate.Point{X: 120, Y: 40}, 1); got != selHandleN {
		t.Fatalf("edge hit = %d", got)
	}
	if got := selectionHit(sel, annotate.Point{X: 120, Y: 100}, 1); got != selHandleMove {
		t.Fatalf("inside hit = %d", got)
	}
	if got := selectionHit(sel, annotate.Point{X: 10, Y: 10}, 1); got != selHandleNone {
		t.Fatalf("outside hit = %d", got)
	}
}

func TestResizeSelectionKeepsOppositeEdge(t *testing.T) {
	orig := image.Rect(40, 40, 200, 160)
	bounds := image.Rect(0, 0, 400, 300)
	got := resizeSelection(orig, selHandleSE, annotate.Point{X: 220, Y: 180}, bounds)
	if got != image.Rect(40, 40, 220, 180) {
		t.Fatalf("se resize = %v", got)
	}
	got = resizeSelection(orig, selHandleNW, annotate.Point{X: 20, Y: 30}, bounds)
	if got != image.Rect(20, 30, 200, 160) {
		t.Fatalf("nw resize = %v", got)
	}
}

func TestMoveSelectionStaysInBounds(t *testing.T) {
	orig := image.Rect(10, 10, 60, 50)
	bounds := image.Rect(0, 0, 80, 70)
	got := moveSelection(orig, 100, 100, bounds)
	if got != image.Rect(30, 30, 80, 70) {
		t.Fatalf("clamped move = %v", got)
	}
}
