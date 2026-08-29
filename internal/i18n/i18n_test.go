package i18n

import "testing"

func TestLanguageIsArabic(t *testing.T) {
	cases := map[string]bool{
		"ar":           true,
		"ar_SA.UTF-8":  true,
		"ar_EG":        true,
		"en_US.UTF-8":  false,
		"C":            false,
		"C.UTF-8":      false,
		"fr_FR.UTF-8":  false,
	}
	for tag, want := range cases {
		if got := languageIsArabic(tag); got != want {
			t.Fatalf("languageIsArabic(%q) = %v, want %v", tag, got, want)
		}
	}
}

func TestArabicMessagesCoverCommonUI(t *testing.T) {
	for _, key := range []string{"Move marks", "Pen", "Arrow", "Ellipse", "Text", "Capture area", "Area", "Recent", "Main menu", "Copy path", "Delete"} {
		if arabicMessages[key] == "" {
			t.Fatalf("missing Arabic for %q", key)
		}
	}
}
