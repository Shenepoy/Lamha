package keys

import "testing"

func TestToPortalTrigger(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"<Control><Shift>A", "CTRL+SHIFT+A"},
		{"<Control>s", "CTRL+S"},
		{"Return", "RETURN"},
		{"", ""},
		{"<Alt>Print", "ALT+PRINT"},
	}
	for _, test := range tests {
		if got := ToPortalTrigger(test.in); got != test.want {
			t.Fatalf("ToPortalTrigger(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}
