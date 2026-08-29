package ui

import (
	"sync"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/annotate"
	"github.com/lamha-app/lamha/internal/i18n"
	"github.com/lamha-app/lamha/internal/keys"
)

var toolbarCSSOnce sync.Once

func ensureToolbarCSS() {
	toolbarCSSOnce.Do(func() {
		provider := gtk.NewCSSProvider()
		provider.LoadFromData(`
.lamha-toolbar {
  background-color: alpha(black, 0.78);
  border-radius: 14px;
  padding: 8px 14px;
  margin-top: 16px;
  color: white;
}
.lamha-toolbar button {
  color: white;
  min-width: 36px;
  min-height: 36px;
  padding: 3px;
}
.lamha-toolbar button:hover {
  background-color: alpha(white, 0.12);
}
.lamha-toolbar button:checked,
.lamha-toolbar button:active {
  background-color: alpha(white, 0.22);
  color: white;
}
.lamha-toolbar button:disabled {
  opacity: 0.35;
}
.lamha-toolbar image {
  color: white;
  -gtk-icon-style: symbolic;
  -gtk-icon-size: 28px;
}
.lamha-text-entry {
  min-width: 180px;
  padding: 6px 10px;
  border-radius: 8px;
  background-color: alpha(black, 0.72);
  color: white;
  caret-color: white;
}
.lamha-toolbar separator {
  background-color: alpha(white, 0.35);
  min-width: 1px;
}
.lamha-toolbar scale trough {
  background-color: alpha(white, 0.25);
}
.lamha-toolbar .dim-label {
  color: alpha(white, 0.85);
}
.lamha-swatch {
  min-width: 28px;
  min-height: 28px;
  padding: 2px;
  border-radius: 999px;
}
.lamha-lens,
.lamha-lens-layer {
  background: none;
  background-color: transparent;
  box-shadow: none;
  border: none;
}
.lamha-lens {
  border-radius: 999px;
}
.lamha-logo {
  min-width: 28px;
  min-height: 28px;
}
`)
		if display := gdk.DisplayGetDefault(); display != nil {
			gtk.StyleContextAddProviderForDisplay(display, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
		}
	})
}

func newToolbarIcon(icon string) *gtk.Image {
	image := gtk.NewImageFromIconName(icon)
	image.SetPixelSize(28)
	return image
}

func newIconButton(icon, tip string) *gtk.Button {
	button := gtk.NewButton()
	button.SetChild(newToolbarIcon(icon))
	button.SetTooltipText(tip)
	button.SetHasFrame(false)
	return button
}

func newIconToggle(icon, tip string) *gtk.ToggleButton {
	button := gtk.NewToggleButton()
	button.SetChild(newToolbarIcon(icon))
	button.SetTooltipText(tip)
	button.SetHasFrame(false)
	return button
}

func newColorSwatch(name string, color annotate.Color, onPick func(annotate.Color)) *gtk.Button {
	swatch := gtk.NewDrawingArea()
	swatch.SetSizeRequest(16, 16)
	swatch.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
		cr.SetSourceRGB(float64(color.R)/255, float64(color.G)/255, float64(color.B)/255)
		cr.Arc(float64(width)/2, float64(height)/2, float64(min(width, height))/2-1, 0, 6.28318)
		cr.Fill()
	})

	button := gtk.NewButton()
	button.SetChild(swatch)
	button.SetTooltipText(name)
	button.SetHasFrame(false)
	button.SetCSSClasses([]string{"lamha-swatch"})
	button.ConnectClicked(func() { onPick(color) })
	return button
}

type toolSwitcher struct {
	buttons map[annotate.Tool]*gtk.ToggleButton
}

func (s *toolSwitcher) activate(tool annotate.Tool) {
	if s == nil {
		return
	}
	if button, ok := s.buttons[tool]; ok {
		button.SetActive(true)
	}
}

func appendToolToggles(box *gtk.Box, tools []toolItem, active annotate.Tool, onPick func(annotate.Tool)) *toolSwitcher {
	switcher := &toolSwitcher{buttons: map[annotate.Tool]*gtk.ToggleButton{}}
	var group *gtk.ToggleButton
	for _, item := range tools {
		item := item
		button := newDrawnToggle(toolIcon(item.tool), item.tip)
		if group == nil {
			group = button
		} else {
			button.SetGroup(group)
		}
		button.SetActive(item.tool == active)
		button.ConnectToggled(func() {
			if button.Active() {
				onPick(item.tool)
			}
		})
		switcher.buttons[item.tool] = button
		box.Append(button)
	}
	return switcher
}

type toolItem struct {
	tip  string
	tool annotate.Tool
}

func captureTools() []toolItem {
	return []toolItem{
		{withKey(i18n.T("Select area or a mark"), keys.ToolSelect), annotate.ToolSelect},
		{withKey(i18n.T("Move marks"), keys.ToolMove), annotate.ToolMove},
		{withKey(i18n.T("Pen"), keys.ToolPen), annotate.ToolPen},
		{withKey(i18n.T("Arrow"), keys.ToolArrow), annotate.ToolArrow},
		{withKey(i18n.T("Box"), keys.ToolBox), annotate.ToolBox},
		{withKey(i18n.T("Ellipse"), keys.ToolEllipse), annotate.ToolEllipse},
		{withKey(i18n.T("Highlight"), keys.ToolHighlight), annotate.ToolHighlight},
		{withKey(i18n.T("Blur"), keys.ToolBlur), annotate.ToolBlur},
		{withKey(i18n.T("Numbered steps"), keys.ToolStep), annotate.ToolStep},
		{withKey(i18n.T("Text"), keys.ToolText), annotate.ToolText},
		{withKey(i18n.T("Magic erase brush"), keys.ToolErase), annotate.ToolMagicErase},
		{withKey(i18n.T("Area magic erase"), keys.ToolAreaErase), annotate.ToolAreaErase},
	}
}

func editorTools() []toolItem {
	return captureTools()
}
