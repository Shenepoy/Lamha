// Package i18n translates Lamha's UI from the process locale.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	once     sync.Once
	arabic   bool
	messages map[string]string
)

func initLocale() {
	once.Do(func() {
		arabic = languageIsArabic(languageTag())
		if arabic {
			messages = arabicMessages
		}
	})
}

// Arabic reports whether the UI should use Arabic copy.
func Arabic() bool {
	initLocale()
	return arabic
}

// RTL reports whether widgets should flow right to left.
func RTL() bool {
	return Arabic()
}

// T translates an English UI string. Unknown keys stay in English.
func T(msg string) string {
	initLocale()
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
