package keys

import (
	"strings"
	"unicode"
)

// ToPortalTrigger converts a GTK accelerator to a Global Shortcuts trigger.
func ToPortalTrigger(accel string) string {
	if strings.TrimSpace(accel) == "" {
		return ""
	}
	var mods []string
	rest := accel
	for {
		start := strings.IndexByte(rest, '<')
		end := strings.IndexByte(rest, '>')
		if start != 0 || end <= start {
			break
		}
		token := strings.ToUpper(rest[start+1 : end])
		rest = rest[end+1:]
		switch token {
		case "CONTROL", "PRIMARY", "CTRL":
			mods = append(mods, "CTRL")
		case "SHIFT":
			mods = append(mods, "SHIFT")
		case "ALT", "MOD1":
			mods = append(mods, "ALT")
		case "SUPER", "META", "WIN":
			mods = append(mods, "SUPER")
		}
	}
	key := strings.ToUpper(strings.TrimSpace(rest))
	key = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, key)
	if key == "" {
		return strings.Join(mods, "+")
	}
	if len(mods) == 0 {
		return key
	}
	return strings.Join(append(mods, key), "+")
}
