package ui

import "testing"

func TestParseCaptureMode(t *testing.T) {
	tests := []struct {
		in      string
		want    CaptureMode
		wantErr bool
	}{
		{in: "area", want: CaptureArea},
		{in: " Window ", want: CaptureWindow},
		{in: "SCREEN", want: CaptureScreen},
		{in: "monitor", wantErr: true},
	}

	for _, test := range tests {
		got, err := ParseCaptureMode(test.in)
		if test.wantErr {
			if err == nil {
				t.Fatalf("ParseCaptureMode(%q) error = nil, want an error", test.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseCaptureMode(%q) error = %v", test.in, err)
		}
		if got != test.want {
			t.Fatalf("ParseCaptureMode(%q) = %v, want %v", test.in, got, test.want)
		}
	}
}

func TestDelayFromIndex(t *testing.T) {
	if delayFromIndex(0) != 0 {
		t.Fatal("delayFromIndex(0) should be no delay")
	}
	if delayFromIndex(2) != delayChoices[2].delay {
		t.Fatalf("delayFromIndex(2) = %s", delayFromIndex(2))
	}
	if delayFromIndex(99) != 0 {
		t.Fatal("delayFromIndex should ignore unknown indexes")
	}
}

func TestCaptureWindowPlanWhenMainWindowIsOpen(t *testing.T) {
	plan := captureWindowPlanFor(true)
	if !plan.restoreMain {
		t.Fatal("capture should restore an originally visible main window")
	}
	if !plan.hideBeforeGrab {
		t.Fatal("capture should hide the main window before grabbing the desktop")
	}
	if !plan.dedicatedOverlay {
		t.Fatal("capture should not fullscreen the main application window")
	}
}

func TestCaptureWindowPlanWhenMainWindowIsHidden(t *testing.T) {
	plan := captureWindowPlanFor(false)
	if plan.restoreMain {
		t.Fatal("background capture should leave the main window hidden")
	}
	if plan.hideBeforeGrab {
		t.Fatal("background capture should not wait for an already hidden window")
	}
	if !plan.dedicatedOverlay {
		t.Fatal("capture overlay should always use its own window")
	}
}
