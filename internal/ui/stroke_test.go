package ui

import "testing"

func TestStrokePreviewStaysFixed(t *testing.T) {
	thinW, thinH := strokePreviewSize(true, false)
	thickW, thickH := strokePreviewSize(true, false)
	if thinW != thickW || thinH != thickH {
		t.Fatalf("compact preview must not depend on stroke width: %d×%d vs %d×%d", thinW, thinH, thickW, thickH)
	}
	if thinW != compactPreviewLength || thinH != compactPreviewBreadth {
		t.Fatalf("compact horizontal preview = %d×%d", thinW, thinH)
	}
	vw, vh := strokePreviewSize(true, true)
	if vw != verticalPreviewSize || vh != verticalPreviewSize {
		t.Fatalf("compact vertical preview = %d×%d", vw, vh)
	}
	if vw > 32 || vh > 36 {
		t.Fatalf("vertical preview must not widen the toolbar: %d×%d", vw, vh)
	}
	ew, eh := strokePreviewSize(false, false)
	if ew != editorPreviewLength || eh != editorPreviewBreadth {
		t.Fatalf("editor preview = %d×%d", ew, eh)
	}
}
