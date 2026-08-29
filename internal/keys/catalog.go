// Package keys stores Lamha's editable keyboard shortcuts.
package keys

// ID is a stable keybind identifier stored in the user config.
type ID string

const (
	CaptureArea   ID = "capture-area"
	CaptureWindow ID = "capture-window"
	CaptureScreen ID = "capture-screen"

	ToolSelect    ID = "tool-select"
	ToolMove      ID = "tool-move"
	ToolPen       ID = "tool-pen"
	ToolArrow     ID = "tool-arrow"
	ToolBox       ID = "tool-box"
	ToolEllipse   ID = "tool-ellipse"
	ToolHighlight ID = "tool-highlight"
	ToolBlur      ID = "tool-blur"
	ToolStep      ID = "tool-step"
	ToolText      ID = "tool-text"
	ToolErase     ID = "tool-erase"
	ToolAreaErase ID = "tool-area-erase"

	Undo       ID = "undo"
	Redo       ID = "redo"
	Save       ID = "save"
	Confirm    ID = "confirm"
	ToggleCopy ID = "toggle-copy"
	Delete     ID = "delete"
	Duplicate  ID = "duplicate"
	Escape     ID = "escape"
	WidthDown  ID = "width-down"
	WidthUp    ID = "width-up"

	Color1 ID = "color-1"
	Color2 ID = "color-2"
	Color3 ID = "color-3"
	Color4 ID = "color-4"
	Color5 ID = "color-5"
	Color6 ID = "color-6"
	Color7 ID = "color-7"
)

// Binding describes one editable shortcut.
type Binding struct {
	ID      ID
	Group   string
	Label   string
	Default string
	System  bool
}

// Catalog is every shortcut the user can edit.
func Catalog() []Binding {
	return []Binding{
		{ID: CaptureArea, Group: "Capture", Label: "Capture area", Default: "<Control><Shift>A", System: true},
		{ID: CaptureWindow, Group: "Capture", Label: "Capture window", Default: "<Control><Shift>W", System: true},
		{ID: CaptureScreen, Group: "Capture", Label: "Capture screen", Default: "<Control><Shift>S", System: true},

		{ID: ToolSelect, Group: "Markup", Label: "Select", Default: "S"},
		{ID: ToolMove, Group: "Markup", Label: "Move marks", Default: "M"},
		{ID: ToolPen, Group: "Markup", Label: "Pen", Default: "P"},
		{ID: ToolArrow, Group: "Markup", Label: "Arrow", Default: "L"},
		{ID: ToolBox, Group: "Markup", Label: "Box", Default: "R"},
		{ID: ToolEllipse, Group: "Markup", Label: "Ellipse", Default: "O"},
		{ID: ToolHighlight, Group: "Markup", Label: "Highlight", Default: "H"},
		{ID: ToolBlur, Group: "Markup", Label: "Blur", Default: "B"},
		{ID: ToolStep, Group: "Markup", Label: "Numbered steps", Default: "N"},
		{ID: ToolText, Group: "Markup", Label: "Text", Default: "T"},
		{ID: ToolErase, Group: "Markup", Label: "Magic erase", Default: "E"},
		{ID: ToolAreaErase, Group: "Markup", Label: "Area erase", Default: "A"},

		{ID: Undo, Group: "Markup", Label: "Undo", Default: "<Control>z"},
		{ID: Redo, Group: "Markup", Label: "Redo", Default: "<Control>y"},
		{ID: Save, Group: "Markup", Label: "Save", Default: "<Control>s"},
		{ID: Confirm, Group: "Markup", Label: "Confirm capture", Default: "Return"},
		{ID: ToggleCopy, Group: "Markup", Label: "Toggle copy on save", Default: "<Control>c"},
		{ID: Delete, Group: "Markup", Label: "Delete selected mark", Default: "Delete"},
		{ID: Duplicate, Group: "Markup", Label: "Duplicate selected mark", Default: "<Control>d"},
		{ID: Escape, Group: "Markup", Label: "Cancel / deselect", Default: "Escape"},
		{ID: WidthDown, Group: "Markup", Label: "Smaller brush", Default: "bracketleft"},
		{ID: WidthUp, Group: "Markup", Label: "Larger brush", Default: "bracketright"},

		{ID: Color1, Group: "Markup", Label: "Color 1 (red)", Default: "1"},
		{ID: Color2, Group: "Markup", Label: "Color 2 (orange)", Default: "2"},
		{ID: Color3, Group: "Markup", Label: "Color 3 (yellow)", Default: "3"},
		{ID: Color4, Group: "Markup", Label: "Color 4 (green)", Default: "4"},
		{ID: Color5, Group: "Markup", Label: "Color 5 (blue)", Default: "5"},
		{ID: Color6, Group: "Markup", Label: "Color 6 (white)", Default: "6"},
		{ID: Color7, Group: "Markup", Label: "Color 7 (black)", Default: "7"},
	}
}

// Lookup returns the catalog entry for id.
func Lookup(id ID) (Binding, bool) {
	for _, item := range Catalog() {
		if item.ID == id {
			return item, true
		}
	}
	return Binding{}, false
}

// Groups returns catalog entries in display order, grouped by section title.
func Groups() []struct {
	Title    string
	Bindings []Binding
} {
	var out []struct {
		Title    string
		Bindings []Binding
	}
	index := map[string]int{}
	for _, item := range Catalog() {
		if i, ok := index[item.Group]; ok {
			out[i].Bindings = append(out[i].Bindings, item)
			continue
		}
		index[item.Group] = len(out)
		out = append(out, struct {
			Title    string
			Bindings []Binding
		}{Title: item.Group, Bindings: []Binding{item}})
	}
	return out
}
