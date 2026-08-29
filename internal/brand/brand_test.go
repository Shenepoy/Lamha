package brand

import (
	"bytes"
	"testing"
)

func TestLogoSVG(t *testing.T) {
	if !bytes.Contains(SVG, []byte("<svg")) {
		t.Fatal("embedded logo is not SVG")
	}
	if !bytes.Contains(SVG, []byte("#7C2AA8")) {
		t.Fatal("embedded logo is missing the brand fill")
	}
	if Name != "io.github.lamha.Lamha" {
		t.Fatalf("Name = %q", Name)
	}
	if PanelName != "io.github.lamha.Lamha-panel" {
		t.Fatalf("PanelName = %q", PanelName)
	}
}

func TestTrayPixbufFillsSlot(t *testing.T) {
	pb := TrayPixbuf(128)
	if pb == nil {
		t.Skip("SVG pixbuf loader is unavailable")
	}
	if pb.Width() != 128 || pb.Height() != 128 {
		t.Fatalf("TrayPixbuf(128) = %dx%d", pb.Width(), pb.Height())
	}
	full := Pixbuf(128)
	if full == nil {
		return
	}
	cropped := cropOpaque(full)
	if cropped == nil {
		t.Fatal("logo crop did not tighten the padded SVG")
	}
	if cropped.Width() >= full.Width()-4 {
		t.Fatalf("crop %d is not tighter than %d", cropped.Width(), full.Width())
	}
}
