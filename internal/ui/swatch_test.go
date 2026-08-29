package ui

import (
	"testing"

	"github.com/lamha-app/lamha/internal/annotate"
)

func TestColorLumaPicksBorder(t *testing.T) {
	cases := []struct {
		color annotate.Color
		want  string
	}{
		{annotate.RGB(0xffffff), "dark"},
		{annotate.RGB(0xfacc15), "dark"},
		{annotate.RGB(0x111827), "light"},
		{annotate.RGB(0xe11d48), "neutral"},
	}
	for _, item := range cases {
		got := swatchBorderKind(item.color)
		if got != item.want {
			t.Fatalf("swatchBorderKind(%v) = %q, want %q", item.color, got, item.want)
		}
	}
}
