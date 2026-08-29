package ui

import (
	"math"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/lamha-app/lamha/internal/annotate"
	"github.com/lamha-app/lamha/internal/i18n"
)

type textDraft struct {
	active bool
	pos    annotate.Point
	text   string
	color  annotate.Color
	width  float64
}

type textInput struct {
	draft   textDraft
	entry   *gtk.Entry
	host    *gtk.Overlay
	replace int
}

func (t *textInput) active() bool {
	return t != nil && t.draft.active
}

func (t *textInput) begin(host *gtk.Overlay, view annotate.View, pos annotate.Point, color annotate.Color, width float64, onCommit func()) {
	t.start(host, view, pos, color, width, "", -1, onCommit)
}

func (t *textInput) beginReplace(host *gtk.Overlay, view annotate.View, stroke annotate.Stroke, index int, onCommit func()) {
	t.start(host, view, annotate.Point{X: stroke.X1, Y: stroke.Y1}, stroke.Color, stroke.Width, stroke.Text, index, onCommit)
}

func (t *textInput) start(host *gtk.Overlay, view annotate.View, pos annotate.Point, color annotate.Color, width float64, text string, replace int, onCommit func()) {
	if t == nil || host == nil {
		return
	}
	t.cancel()
	t.draft = textDraft{active: true, pos: pos, color: color, width: width, text: text}
	t.replace = replace
	t.host = host
	t.entry = gtk.NewEntry()
	t.entry.SetPlaceholderText(i18n.T("Type here…"))
	t.entry.SetWidthChars(16)
	t.entry.AddCSSClass("lamha-text-entry")
	t.entry.SetHAlign(gtk.AlignStart)
	t.entry.SetVAlign(gtk.AlignStart)
	if i18n.RTL() {
		t.entry.SetDirection(gtk.TextDirRTL)
	} else {
		t.entry.SetDirection(gtk.TextDirLTR)
	}
	wx, wy := view.ToWidget(pos.X, pos.Y)
	t.entry.SetMarginStart(int(math.Max(8, wx)))
	t.entry.SetMarginTop(int(math.Max(8, wy)))
	t.entry.ConnectChanged(func() {
		if t.entry != nil {
			t.draft.text = t.entry.Text()
		}
	})
	t.entry.ConnectActivate(func() {
		if onCommit != nil {
			onCommit()
		}
	})
	host.AddOverlay(t.entry)
	if text != "" {
		t.entry.SetText(text)
		t.entry.SelectRegion(0, -1)
	}
	t.entry.GrabFocus()
}

func (t *textInput) cancel() {
	if t == nil {
		return
	}
	t.hideEntry()
	t.draft = textDraft{}
	t.replace = -1
}

func (t *textInput) hideEntry() {
	if t == nil || t.entry == nil {
		return
	}
	if t.host != nil {
		t.host.RemoveOverlay(t.entry)
	}
	t.entry = nil
	t.host = nil
}

func (t *textInput) take() (stroke *annotate.Stroke, replace int) {
	if t == nil {
		return nil, -1
	}
	replace = t.replace
	return t.commit(), replace
}

func (t *textInput) commit() *annotate.Stroke {
	if t == nil || !t.draft.active {
		return nil
	}
	if t.entry != nil {
		t.draft.text = t.entry.Text()
	}
	if t.draft.text == "" {
		t.cancel()
		return nil
	}
	stroke := t.draft.stroke()
	t.cancel()
	return &stroke
}

func applyTextCommit(doc *annotate.Document, input *textInput) (selected int) {
	stroke, replace := input.take()
	if replace >= 0 {
		if stroke == nil {
			doc.Remove(replace)
			return -1
		}
		doc.Replace(replace, *stroke)
		return replace
	}
	if stroke != nil {
		doc.Add(*stroke)
		return doc.Len() - 1
	}
	return -1
}

func (t *textDraft) stroke() annotate.Stroke {
	return annotate.Stroke{
		Tool:  annotate.ToolText,
		Color: t.color,
		Width: t.width,
		X1:    t.pos.X,
		Y1:    t.pos.Y,
		Text:  t.text,
	}
}

func cursorForTool(tool annotate.Tool) string {
	switch tool {
	case annotate.ToolMove:
		return "grab"
	case annotate.ToolText:
		return "text"
	default:
		return "crosshair"
	}
}

func toolPlacesOnClick(tool annotate.Tool) bool {
	return tool == annotate.ToolStep || tool == annotate.ToolText
}
