// Package i18n translates Lamha's UI from the process locale.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	mu       sync.RWMutex
	ready    bool
	override string
	arabic   bool
	messages map[string]string
)

func ensureLocale() {
	mu.Lock()
	defer mu.Unlock()
	if !ready {
		applyLocked()
	}
}

func applyLocked() {
	switch override {
	case "ar":
		arabic = true
		messages = arabicMessages
	case "en":
		arabic = false
		messages = nil
	default:
		arabic = languageIsArabic(languageTag())
		if arabic {
			messages = arabicMessages
		} else {
			messages = nil
		}
	}
	ready = true
}

// SetLanguage overrides the process language. Empty or "system" follows the OS.
func SetLanguage(code string) {
	mu.Lock()
	defer mu.Unlock()
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "ar", "ar_sa", "arabic":
		override = "ar"
	case "en", "en_us", "english":
		override = "en"
	default:
		override = ""
	}
	applyLocked()
}

// Language is the override in use: en, ar, or empty for system.
func Language() string {
	ensureLocale()
	mu.RLock()
	defer mu.RUnlock()
	return override
}

// Arabic reports whether the UI should use Arabic copy.
func Arabic() bool {
	ensureLocale()
	mu.RLock()
	defer mu.RUnlock()
	return arabic
}

// RTL reports whether widgets should flow right to left.
func RTL() bool {
	return Arabic()
}

// T translates an English UI string. Unknown keys stay in English.
func T(msg string) string {
	ensureLocale()
	mu.RLock()
	defer mu.RUnlock()
	if messages == nil {
		return msg
	}
	if out, ok := messages[msg]; ok {
		return out
	}
	return msg
}

// Tf translates then formats msg.
func Tf(msg string, args ...any) string {
	return fmt.Sprintf(T(msg), args...)
}

func languageTag() string {
	if value := firstLanguage(os.Getenv("LANGUAGE")); value != "" {
		return value
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); usableLocale(value) {
			return value
		}
	}
	return "en"
}

func firstLanguage(value string) string {
	for _, part := range strings.Split(value, ":") {
		part = strings.TrimSpace(part)
		if usableLocale(part) {
			return part
		}
	}
	return ""
}

func usableLocale(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value == "C" || strings.HasPrefix(value, "C.") || value == "POSIX" {
		return false
	}
	return true
}

func languageIsArabic(tag string) bool {
	lang := strings.ToLower(strings.SplitN(strings.SplitN(tag, ".", 2)[0], "@", 2)[0])
	return lang == "ar" || strings.HasPrefix(lang, "ar_")
}
