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
  padding: 10px 14px;
  border-radius: 15px;
}
.lamha-toolbar-vertical {
  padding: 8px 6px;
}
.lamha-toolbar-vertical button {
  min-width: 32px;
  min-height: 32px;
  padding: 1px;
}
.lamha-toolbar scrolledwindow {
  background: none;
  min-width: 0;
  min-height: 0;
}
.lamha-grab {
  min-width: 18px;
  min-height: 18px;
  opacity: 0.75;
  border-radius: 8px;
}
.lamha-grab:hover {
  background-color: alpha(currentColor, 0.12);
  opacity: 1;
}
.lamha-toolbar-locked .lamha-grab {
  opacity: 0.28;
}
.lamha-capture-status,
.lamha-editor-status {
  padding: 6px 12px;
  border-radius: 10px;
}
.lamha-toolbar button {
  min-width: 36px;
  min-height: 36px;
  padding: 3px;
}
.lamha-toolbar button:disabled {
  opacity: var(--disabled-opacity, 0.5);
}
.lamha-toolbar image {
  -gtk-icon-style: symbolic;
  -gtk-icon-size: 28px;
}
.lamha-text-entry {
  min-width: 180px;
  padding: 6px 10px;
  border-radius: 8px;
}
.lamha-toolbar separator {
  min-width: 1px;
  min-height: 1px;
}
.lamha-toolbar-vertical separator {
  min-width: 0;
}
.lamha-stroke-preview {
  min-width: 0;
  min-height: 0;
  margin: 0;
}
.lamha-toolbar-vertical scale {
  min-width: 0;
  min-height: 40px;
  max-width: 18px;
  padding: 0;
}
.lamha-toolbar-horizontal scale {
  min-height: 16px;
  max-height: 18px;
}
.lamha-stroke-stepper {
  margin: 2px 0;
  padding: 2px;
  border-radius: 8px;
}
.lamha-stroke-stepper-vertical {
  min-width: 0;
}
.lamha-stroke-stepper-dark {
  background-color: alpha(currentColor, 0.12);
  border: 1px solid alpha(currentColor, 0.28);
}
.lamha-stroke-stepper-light {
  background-color: alpha(@theme_fg_color, 0.08);
  border: 1px solid @borders;
  color: @theme_fg_color;
}
.lamha-stroke-stepper-light button,
.lamha-stroke-stepper-light .lamha-stroke-value {
  color: inherit;
}
.lamha-stroke-stepper button {
  min-width: 28px;
  min-height: 28px;
  padding: 0 4px;
  border-radius: 6px;
}
.lamha-stroke-value {
  min-width: 2.2em;
  font-weight: 600;
  padding: 0 6px;
}
.lamha-step-pop contents,
.lamha-step-pop {
  padding: 4px;
}
.lamha-step-pop .lamha-stroke-stepper {
  margin: 0;
}
.lamha-swatch {
  min-width: 32px;
  min-height: 32px;
  margin: 2px 1px;
  padding: 3px;
}
.lamha-editor-chrome button {
  min-width: 36px;
  min-height: 36px;
  padding: 4px;
  border-radius: 8px;
}
.lamha-editor-chrome button:checked,
.lamha-editor-chrome button:active {
  background-color: alpha(@theme_selected_bg_color, 0.24);
}
.lamha-editor-bar {
  background-color: @theme_bg_color;
  color: @theme_fg_color;
  border-bottom: 1px solid @borders;
  padding: 4px 10px;
}
.lamha-editor-bar separator {
  min-height: 28px;
  margin: 6px 4px;
  background-color: @borders;
}
.lamha-canvas-host {
  background-color: @theme_base_color;
  color: @theme_text_color;
  border-radius: 0;
  border: none;
}
@media (prefers-contrast: more) {
  .lamha-editor-bar,
  .lamha-stroke-stepper-light,
  .lamha-stroke-stepper-dark {
    border-width: 2px;
  }
}
window.lamha-capture,
window.lamha-capture > box,
.lamha-capture-host,
.lamha-capture-canvas {
  background-color: #0d0d0e;
  background-image: none;
  box-shadow: none;
}
window.lamha-ghost {
  background-color: transparent;
  background-image: none;
  box-shadow: none;
}
window.lamha-capture scrolledwindow,
window.lamha-capture scrolledwindow > viewport,
window.lamha-capture viewport,
.lamha-toolbar scrolledwindow,
.lamha-toolbar scrolledwindow > viewport,
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
window.lamha-picker {
  background-color: transparent;
}
.lamha-picker-canvas {
  background-color: transparent;
}
`)
		if display := gdk.DisplayGetDefault(); display != nil {
			gtk.StyleContextAddProviderForDisplay(display, provider, gtk.STYLE_PROVIDER_PRIORITY_USER)
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
	unfocusable(&button.Widget)
	return button
}

func newIconToggle(icon, tip string) *gtk.ToggleButton {
	button := gtk.NewToggleButton()
	button.SetChild(newToolbarIcon(icon))
	button.SetTooltipText(tip)
	button.SetHasFrame(false)
	unfocusable(&button.Widget)
	return button
}

type colorSwitcher struct {
	selected int
	updating bool
	areas    []*gtk.DrawingArea
	buttons  []*gtk.ToggleButton
}

func (s *colorSwitcher) queueDraw() {
	if s == nil {
		return
	}
	for _, area := range s.areas {
		if area != nil {
			area.QueueDraw()
		}
	}
}

func (s *colorSwitcher) activate(index int) {
	if s == nil || index < 0 || index >= len(s.buttons) {
		return
	}
	if s.updating {
		return
	}
	s.updating = true
	s.selected = index
	for i, button := range s.buttons {
		if button.Active() != (i == index) {
			button.SetActive(i == index)
		}
		s.areas[i].QueueDraw()
	}
	s.updating = false
}

func appendColorSwatches(box *gtk.Box, active int, onPick func(int)) *colorSwitcher {
	switcher := &colorSwitcher{selected: active}
	var group *gtk.ToggleButton
	for i, item := range editorColors {
		i := i
		item := item
		area := gtk.NewDrawingArea()
		area.SetSizeRequest(22, 22)
		area.SetContentWidth(22)
		area.SetContentHeight(22)
		area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, width, height int) {
			drawColorSwatch(cr, item.color, switcher.selected == i, width, height)
		})

		button := gtk.NewToggleButton()
		button.SetChild(area)
		button.SetTooltipText(withKey(colorName(item.name), colorID(i)))
		button.SetHasFrame(false)
		button.SetVAlign(gtk.AlignCenter)
		unfocusable(&button.Widget)
		button.SetCSSClasses([]string{"lamha-swatch", "circular", "flat"})
		if group == nil {
			group = button
		} else {
			button.SetGroup(group)
		}
		button.SetActive(i == active)
		button.ConnectToggled(func() {
			if button.Active() {
				onPick(i)
			}
		})
		switcher.areas = append(switcher.areas, area)
		switcher.buttons = append(switcher.buttons, button)
		box.Append(button)
	}
	return switcher
}

func swatchBorderKind(color annotate.Color) string {
	luma := (0.299*float64(color.R) + 0.587*float64(color.G) + 0.114*float64(color.B)) / 255
	switch {
	case luma > 0.72:
		return "dark"
	case luma < 0.18:
		return "light"
	default:
		return "neutral"
	}
}

func drawColorSwatch(cr *cairo.Context, color annotate.Color, selected bool, width, height int) {
	cx := float64(width) / 2
	cy := float64(height) / 2
	radius := min(cx, cy) - 3.6

	cr.SetSourceRGB(float64(color.R)/255, float64(color.G)/255, float64(color.B)/255)
	cr.Arc(cx, cy, radius, 0, 6.283185307179586)
	cr.Fill()

	switch swatchBorderKind(color) {
	case "dark":
		if themePrefersDark() {
			cr.SetSourceRGB(0.12, 0.13, 0.15)
		} else {
			cr.SetSourceRGB(0.22, 0.23, 0.26)
		}
	case "light":
		if themePrefersDark() {
			cr.SetSourceRGB(0.82, 0.84, 0.88)
		} else {
			cr.SetSourceRGB(0.64, 0.66, 0.70)
		}
	default:
		if themePrefersDark() {
			cr.SetSourceRGBA(1, 1, 1, 0.42)
		} else {
			cr.SetSourceRGBA(0, 0, 0, 0.38)
		}
	}
	cr.SetLineWidth(1.3)
	cr.Arc(cx, cy, radius, 0, 6.283185307179586)
	cr.Stroke()

	if selected {
		cr.SetSourceRGB(0.26, 0.51, 0.96)
		cr.SetLineWidth(2.2)
		cr.Arc(cx, cy, radius+2.5, 0, 6.283185307179586)
		cr.Stroke()
	}
}

type toolSwitcher struct {
	buttons map[annotate.Tool]*gtk.ToggleButton
}

func (s *toolSwitcher) queueDraw() {
	if s == nil {
		return
	}
	for _, button := range s.buttons {
		if button == nil {
			continue
		}
		if child := button.Child(); child != nil {
			gtk.BaseWidget(child).QueueDraw()
		}
		button.QueueDraw()
	}
}

func (s *toolSwitcher) activate(tool annotate.Tool) {
	if s == nil {
		return
	}
	if button, ok := s.buttons[tool]; ok {
		button.SetActive(true)
	}
}

func appendToolToggles(box *gtk.Box, tools []toolItem, active annotate.Tool, ink iconInk, onPick func(annotate.Tool)) *toolSwitcher {
	switcher := &toolSwitcher{buttons: map[annotate.Tool]*gtk.ToggleButton{}}
	var group *gtk.ToggleButton
	for _, item := range tools {
		item := item
		button := newDrawnToggle(toolIcon(item.tool), item.tip, ink)
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
