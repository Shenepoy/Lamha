package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/lamha-app/lamha/internal/i18n"
)

// CaptureMode is a user-facing capture action.
type CaptureMode int

const (
	CaptureNone CaptureMode = iota
	CaptureArea
	CaptureWindow
	CaptureScreen
)

var delayChoices = []struct {
	label string
	delay time.Duration
}{
	{"No delay", 0},
	{"1 second", time.Second},
	{"3 seconds", 3 * time.Second},
	{"5 seconds", 5 * time.Second},
	{"10 seconds", 10 * time.Second},
}

// ParseCaptureMode accepts CLI and desktop-action values.
func ParseCaptureMode(value string) (CaptureMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "area":
		return CaptureArea, nil
	case "window":
		return CaptureWindow, nil
	case "screen":
		return CaptureScreen, nil
	default:
		return CaptureNone, fmt.Errorf("unknown capture mode %q (want area, window, or screen)", value)
	}
}

func delayFromIndex(index uint) time.Duration {
	if int(index) >= len(delayChoices) {
		return 0
	}
	return delayChoices[index].delay
}

func delayLabels() []string {
	labels := make([]string, len(delayChoices))
	for i, choice := range delayChoices {
		labels[i] = i18n.T(choice.label)
	}
	return labels
}
