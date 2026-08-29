package ui

import (
	"testing"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

func TestMatchAccel(t *testing.T) {
	if !matchAccel("<Control>z", gdk.KEY_z, gdk.ControlMask) {
		t.Fatal("Ctrl+Z did not match")
	}
	if matchAccel("<Control>z", gdk.KEY_z, 0) {
		t.Fatal("Z matched Ctrl+Z")
	}
	if !matchAccel("S", gdk.KEY_s, 0) {
		t.Fatal("S did not match")
	}
	if !matchAccel("S", gdk.KEY_S, gdk.ShiftMask) {
		t.Fatal("Shift+S did not match unbound Shift on a letter")
	}
	if !matchAccel("Return", gdk.KEY_KP_Enter, 0) {
		t.Fatal("keypad Enter did not match Return")
	}
	if matchAccel("", gdk.KEY_s, 0) {
		t.Fatal("empty accel matched")
	}
}

func TestAccelFromEventIgnoresModifiers(t *testing.T) {
	if _, ok := accelFromEvent(gdk.KEY_Control_L, gdk.ControlMask); ok {
		t.Fatal("modifier-only event should be ignored")
	}
	got, ok := accelFromEvent(gdk.KEY_a, gdk.ControlMask|gdk.ShiftMask)
	if !ok || got == "" {
		t.Fatalf("accelFromEvent() = %q, ok=%v", got, ok)
	}
}
