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
