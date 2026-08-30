package ui

import (
	"testing"

	"github.com/lamha-app/lamha/internal/prefs"
)

func TestNearestToolbarEdge(t *testing.T) {
	const w, h = 1000.0, 800.0
	cases := []struct {
		x, y float64
		want string
	}{
		{500, 10, prefs.ToolbarEdgeTop},
		{500, 790, prefs.ToolbarEdgeBottom},
		{8, 400, prefs.ToolbarEdgeLeft},
		{990, 400, prefs.ToolbarEdgeRight},
	}
	for _, tc := range cases {
		if got := nearestToolbarEdge(tc.x, tc.y, w, h); got != tc.want {
			t.Fatalf("nearestToolbarEdge(%v,%v) = %s, want %s", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestSnapEdgeFromPointer(t *testing.T) {
	if got := snapEdgeFromPointer(500, 400, 1000, 800, 72, false); got != prefs.ToolbarEdgeFloat {
		t.Fatalf("middle of screen should float, got %s", got)
	}
	if got := snapEdgeFromPointer(20, 20, 1000, 800, 72, true); got != prefs.ToolbarEdgeLeft {
		t.Fatalf("vertical bar at top-left should stay on the side, got %s", got)
	}
	if got := snapEdgeFromPointer(20, 20, 1000, 800, 72, false); got != prefs.ToolbarEdgeTop {
		t.Fatalf("horizontal bar at top-left should stay on the top, got %s", got)
	}
	if got := snapEdgeFromPointer(20, 400, 1000, 800, 72, true); got != prefs.ToolbarEdgeLeft {
		t.Fatalf("near left should snap, got %s", got)
	}
	if got := snapEdgeFromPointer(500, 10, 1000, 800, 72, false); got != prefs.ToolbarEdgeTop {
		t.Fatalf("near top should snap, got %s", got)
	}
}

func TestToolbarOriginSnapsToEdges(t *testing.T) {
	x, y := toolbarOrigin(prefs.ToolbarEdgeTop, 0.5, 0.5, 1000, 800, 200, 60, 16)
	if y != 16 || x != 16+int(0.5*(1000-200-32)) {
		t.Fatalf("top center = %d,%d", x, y)
	}
	x, y = toolbarOrigin(prefs.ToolbarEdgeRight, 0.5, 0, 1000, 800, 80, 400, 16)
	if x != 1000-80-16 || y != 16 {
		t.Fatalf("right start = %d,%d", x, y)
	}
	x, y = toolbarOrigin(prefs.ToolbarEdgeBottom, 1, 0.5, 1000, 800, 200, 60, 16)
	if y != 800-60-16 || x != 1000-200-16 {
		t.Fatalf("bottom end = %d,%d", x, y)
	}
	x, y = toolbarOrigin(prefs.ToolbarEdgeFloat, 0.5, 0.5, 1000, 800, 200, 60, 16)
	if x != 400 || y != 370 {
		t.Fatalf("float center = %d,%d", x, y)
	}
}

func TestSnapToolbarOffset(t *testing.T) {
	if got := snapToolbarOffset(0.48, 800, 72); got != 0.5 {
		t.Fatalf("near center = %v", got)
	}
	if got := snapToolbarOffset(0.2, 800, 72); got != 0.2 {
		t.Fatalf("custom offset should stay: %v", got)
	}
	if got := snapToolbarOffset(0.04, 800, 72); got != 0 {
		t.Fatalf("near start = %v", got)
	}
}

func TestClipDragOriginRightAndBottom(t *testing.T) {
	x, y := clipDragOrigin(990, 790, 1000, 800, 18, 36)
	if x != 982 || y != 764 {
		t.Fatalf("should stop when the handle would leave: %d,%d", x, y)
	}
	x, y = clipDragOrigin(-40, -20, 1000, 800, 200, 60)
	if x != 0 || y != 0 {
		t.Fatalf("origin cannot go negative: %d,%d", x, y)
	}
	x, y = clipDragOrigin(100, 100, 1000, 800, 200, 60)
	if x != 100 || y != 100 {
		t.Fatalf("in-bounds origin should stay: %d,%d", x, y)
	}
}

func TestDragToolbarVertical(t *testing.T) {
	if dragToolbarVertical(500, 400, 1000, 800, false) {
		t.Fatal("center should keep a horizontal bar")
	}
	if !dragToolbarVertical(500, 400, 1000, 800, true) {
		t.Fatal("center should keep a vertical bar")
	}
	if dragToolbarVertical(16, 400, 1000, 800, false) {
		t.Fatal("close to the left should stay horizontal until the edge")
	}
	if !dragToolbarVertical(500, 16, 1000, 800, true) {
		t.Fatal("close to the top should stay vertical until the edge")
	}
	if !dragToolbarVertical(1, 400, 1000, 800, false) {
		t.Fatal("touching the left edge should become vertical")
	}
	if dragToolbarVertical(500, 1, 1000, 800, true) {
		t.Fatal("touching the top edge should become horizontal")
	}
	if !dragToolbarVertical(1, 1, 1000, 800, true) {
		t.Fatal("corner should keep the current vertical bar")
	}
	if dragToolbarVertical(1, 1, 1000, 800, false) {
		t.Fatal("corner should keep the current horizontal bar")
	}
}

func TestSizeForToolbarOrientation(t *testing.T) {
	w, h := sizeForToolbarOrientation(true, 400, 50)
	if w != 50 || h != 400 {
		t.Fatalf("stale horizontal size on a vertical bar: %dx%d", w, h)
	}
	w, h = sizeForToolbarOrientation(false, 50, 400)
	if w != 400 || h != 50 {
		t.Fatalf("stale vertical size on a horizontal bar: %dx%d", w, h)
	}
	w, h = sizeForToolbarOrientation(true, 60, 500)
	if w != 60 || h != 500 {
		t.Fatalf("already vertical: %dx%d", w, h)
	}
}

func TestRightSnapUsesVerticalThickness(t *testing.T) {
	w, _ := sizeForToolbarOrientation(true, 400, 50)
	x := 1000 - w - 16
	if x != 934 {
		t.Fatalf("right-edge x=%d (stale width would place it at %d)", x, 1000-400-16)
	}
}

func TestToolbarOriginTopCenterTracksBarWidth(t *testing.T) {
	x1, y1 := toolbarOrigin(prefs.ToolbarEdgeTop, 0.5, 0.5, 1920, 1080, 200, 60, 16)
	x2, y2 := toolbarOrigin(prefs.ToolbarEdgeTop, 0.5, 0.5, 1920, 1080, 600, 60, 16)
	center1 := x1 + 100
	center2 := x2 + 300
	if center1 != 960 || center2 != 960 || y1 != 16 || y2 != 16 {
		t.Fatalf("top center should be the screen middle: %d/%d y=%d/%d", center1, center2, y1, y2)
	}
}

func TestPlacementCenterIsBarCenter(t *testing.T) {
	const hostW, hostH, barW, barH, margin = 1920.0, 1080.0, 480.0, 64.0, 16.0
	cx, cy := placementCenter(prefs.ToolbarEdgeTop, 0.5, 0.5, hostW, hostH, barW, barH, margin)
	if cx != hostW/2 || cy != margin+barH/2 {
		t.Fatalf("stored 0.5 should be the bar center: %v,%v", cx, cy)
	}
	ox, oy := toolbarOrigin(prefs.ToolbarEdgeTop, 0.5, 0.5, int(hostW), int(hostH), int(barW), int(barH), int(margin))
	if float64(ox)+barW/2 != cx || float64(oy) != margin {
		t.Fatalf("origin from 0.5 = %d,%d", ox, oy)
	}
	gotX, gotY := normalizeOrigin(prefs.ToolbarEdgeTop, float64(ox), float64(oy), hostW, hostH, barW, barH, margin)
	if gotX != 0.5 || gotY != 0.5 {
		t.Fatalf("centered bar should store 0.5,0.5 got %v,%v", gotX, gotY)
	}

	cx, cy = placementCenter(prefs.ToolbarEdgeFloat, 0.5, 0.5, hostW, hostH, barW, barH, margin)
	if cx != hostW/2 || cy != hostH/2 {
		t.Fatalf("float 0.5 should be the screen center: %v,%v", cx, cy)
	}
}

func TestPlausibleBarSizeRejectsFullWindowMeasure(t *testing.T) {
	if plausibleBarSize(false, 1920, 56, 1920, 1080) {
		t.Fatal("a horizontal bar as wide as the screen is not a real toolbar size")
	}
	if !plausibleBarSize(false, 560, 56, 1920, 1080) {
		t.Fatal("typical horizontal toolbar should be accepted")
	}
	if !plausibleBarSize(true, 56, 420, 1920, 1080) {
		t.Fatal("typical vertical toolbar should be accepted")
	}
	if plausibleBarSize(true, 56, 1080, 1920, 1080) {
		t.Fatal("a vertical bar as tall as the screen is not a real toolbar size")
	}
}

func TestSnapDockNormsCenter(t *testing.T) {
	x, y := snapDockNorms(prefs.ToolbarEdgeTop, 0.48, 0.2, 1000, 800, 200, 60, 16, 72)
	if x != 0.5 || y != 0.5 {
		t.Fatalf("top drop near center = %v,%v", x, y)
	}
	x, y = snapDockNorms(prefs.ToolbarEdgeFloat, 0.51, 0.49, 1000, 800, 200, 60, 16, 72)
	if x != 0.5 || y != 0.5 {
		t.Fatalf("float near center = %v,%v", x, y)
	}
}

func TestNormalizeOrigin(t *testing.T) {
	x, y := normalizeOrigin(prefs.ToolbarEdgeTop, 16, 16, 1000, 800, 200, 60, 16)
	if x != 0 {
		t.Fatalf("top start x = %v", x)
	}
	_, y = normalizeOrigin(prefs.ToolbarEdgeLeft, 16, 16+0.5*(800-60-32), 1000, 800, 80, 60, 16)
	if y < 0.49 || y > 0.51 {
		t.Fatalf("left center y = %v", y)
	}
	x, y = normalizeOrigin(prefs.ToolbarEdgeFloat, 400, 370, 1000, 800, 200, 60, 16)
	if x < 0.49 || x > 0.51 || y < 0.49 || y > 0.51 {
		t.Fatalf("float center = %v,%v", x, y)
	}
}
