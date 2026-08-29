package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/annotate"
	"github.com/lamha-app/lamha/internal/keys"
)

type keyActions struct {
	setTool    func(annotate.Tool)
	setColor   func(int)
	nudgeWidth func(float64)
	nudge      func(dx, dy float64) bool
	undo       func()
	redo       func()
	delete     func()
	duplicate  func()
	save       func()
	toggleCopy func()
	escape     func() bool
}

var relevantMods = gdk.ControlMask | gdk.ShiftMask | gdk.AltMask | gdk.SuperMask | gdk.MetaMask

func handleAnnotateKey(keyval uint, state gdk.ModifierType, actions keyActions) bool {
	ctrl := state.Has(gdk.ControlMask)
	shift := state.Has(gdk.ShiftMask)
	store := keys.Current()
	type binding struct {
		id keys.ID
		fn func() bool
	}
	if actions.nudge != nil && !ctrl {
		step := 1.0
		if shift {
			step = 10
		}
		switch keyval {
		case gdk.KEY_Left, gdk.KEY_KP_Left:
			return actions.nudge(-step, 0)
		case gdk.KEY_Right, gdk.KEY_KP_Right:
			return actions.nudge(step, 0)
		case gdk.KEY_Up, gdk.KEY_KP_Up:
			return actions.nudge(0, -step)
		case gdk.KEY_Down, gdk.KEY_KP_Down:
			return actions.nudge(0, step)
		}
	}

	for _, item := range []binding{
		{keys.Undo, func() bool { actions.undo(); return true }},
		{keys.Redo, func() bool { actions.redo(); return true }},
		{keys.Save, func() bool { actions.save(); return true }},
		{keys.ToggleCopy, func() bool {
			if actions.toggleCopy != nil {
				actions.toggleCopy()
			}
			return true
		}},
		{keys.Confirm, func() bool { actions.save(); return true }},
		{keys.Delete, func() bool { actions.delete(); return true }},
		{keys.Duplicate, func() bool {
			if actions.duplicate != nil {
				actions.duplicate()
			}
			return true
		}},
		{keys.Escape, func() bool {
			if actions.escape != nil {
				return actions.escape()
			}
			return false
		}},
		{keys.WidthDown, func() bool { actions.nudgeWidth(-1); return true }},
		{keys.WidthUp, func() bool { actions.nudgeWidth(1); return true }},
		{keys.ToolSelect, func() bool { actions.setTool(annotate.ToolSelect); return true }},
		{keys.ToolMove, func() bool { actions.setTool(annotate.ToolMove); return true }},
		{keys.ToolPen, func() bool { actions.setTool(annotate.ToolPen); return true }},
		{keys.ToolArrow, func() bool { actions.setTool(annotate.ToolArrow); return true }},
		{keys.ToolBox, func() bool { actions.setTool(annotate.ToolBox); return true }},
		{keys.ToolEllipse, func() bool { actions.setTool(annotate.ToolEllipse); return true }},
		{keys.ToolHighlight, func() bool { actions.setTool(annotate.ToolHighlight); return true }},
		{keys.ToolBlur, func() bool { actions.setTool(annotate.ToolBlur); return true }},
		{keys.ToolStep, func() bool { actions.setTool(annotate.ToolStep); return true }},
		{keys.ToolText, func() bool { actions.setTool(annotate.ToolText); return true }},
		{keys.ToolErase, func() bool { actions.setTool(annotate.ToolMagicErase); return true }},
		{keys.ToolAreaErase, func() bool { actions.setTool(annotate.ToolAreaErase); return true }},
		{keys.Color1, func() bool { actions.setColor(0); return true }},
		{keys.Color2, func() bool { actions.setColor(1); return true }},
		{keys.Color3, func() bool { actions.setColor(2); return true }},
		{keys.Color4, func() bool { actions.setColor(3); return true }},
		{keys.Color5, func() bool { actions.setColor(4); return true }},
		{keys.Color6, func() bool { actions.setColor(5); return true }},
		{keys.Color7, func() bool { actions.setColor(6); return true }},
	} {
		if matchAccel(store.Accel(item.id), keyval, state) {
			return item.fn()
		}
	}
	return false
}

func matchAccel(accel string, keyval uint, state gdk.ModifierType) bool {
	if accel == "" {
		return false
	}
	wantKey, wantMods, ok := gtk.AcceleratorParse(accel)
	if !ok || wantKey == 0 {
		return false
	}
	state &= relevantMods
	if wantMods&gdk.ShiftMask == 0 {
		state &^= gdk.ShiftMask
	}
	if !sameKey(keyval, wantKey) {
		return false
	}
	return state == wantMods
}

func sameKey(got, want uint) bool {
	if gdk.KeyvalToLower(got) == gdk.KeyvalToLower(want) {
		return true
	}
	aliases := [][2]uint{
		{gdk.KEY_Return, gdk.KEY_KP_Enter},
		{gdk.KEY_Delete, gdk.KEY_KP_Delete},
		{gdk.KEY_1, gdk.KEY_KP_1},
		{gdk.KEY_2, gdk.KEY_KP_2},
		{gdk.KEY_3, gdk.KEY_KP_3},
		{gdk.KEY_4, gdk.KEY_KP_4},
		{gdk.KEY_5, gdk.KEY_KP_5},
		{gdk.KEY_6, gdk.KEY_KP_6},
		{gdk.KEY_7, gdk.KEY_KP_7},
	}
	for _, pair := range aliases {
		if (got == pair[0] && want == pair[1]) || (got == pair[1] && want == pair[0]) {
			return true
		}
	}
	return false
}

func accelLabel(accel string) string {
	if accel == "" {
		return ""
	}
	key, mods, ok := gtk.AcceleratorParse(accel)
	if !ok || key == 0 {
		return accel
	}
	return gtk.AcceleratorGetLabel(key, mods)
}

func colorID(index int) keys.ID {
	ids := []keys.ID{keys.Color1, keys.Color2, keys.Color3, keys.Color4, keys.Color5, keys.Color6, keys.Color7}
	if index < 0 || index >= len(ids) {
		return ""
	}
	return ids[index]
}

func withKey(label string, id keys.ID) string {
	hint := accelLabel(keys.Current().Accel(id))
	if hint == "" {
		return label
	}
	return label + " (" + hint + ")"
}

func normalizeAccel(accel string) (string, bool) {
	if accel == "" {
		return "", true
	}
	key, mods, ok := gtk.AcceleratorParse(accel)
	if !ok || key == 0 {
		return "", false
	}
	return gtk.AcceleratorName(gdk.KeyvalToLower(key), mods), true
}

func accelFromEvent(keyval uint, state gdk.ModifierType) (string, bool) {
	if isModifierKey(keyval) {
		return "", false
	}
	return normalizeAccel(gtk.AcceleratorName(gdk.KeyvalToLower(keyval), state&relevantMods))
}

func isModifierKey(keyval uint) bool {
	switch keyval {
	case gdk.KEY_Shift_L, gdk.KEY_Shift_R, gdk.KEY_Control_L, gdk.KEY_Control_R,
		gdk.KEY_Alt_L, gdk.KEY_Alt_R, gdk.KEY_Meta_L, gdk.KEY_Meta_R,
		gdk.KEY_Super_L, gdk.KEY_Super_R, gdk.KEY_Hyper_L, gdk.KEY_Hyper_R,
		gdk.KEY_ISO_Level3_Shift, gdk.KEY_Mode_switch, gdk.KEY_Caps_Lock,
		gdk.KEY_Num_Lock, gdk.KEY_Scroll_Lock:
		return true
	}
	return false
}
