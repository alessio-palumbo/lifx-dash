package dashboard

import (
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	// maximum rows shown before scrolling kicks in
	maxVisible = 5

	maxHistory = 50
)

var initialSelectedIdx = -1

// AutocompleteEntry combines a widget.Entry with a floating suggestion box showing
// a list of Matcher-returned suggestions against the word being typed.
type AutocompleteEntry struct {
	widget.Entry
	refocusEntry func()

	Matcher     func(partial string) []string
	OnSubmitted func(text string) (cacheText bool)

	children       []fyne.CanvasObject
	commandHistory *CommandHistory

	mu     sync.Mutex
	items  []string
	selIdx int
	shown  bool

	// one label per suggestion, reused across updates
	labels        []*TapLabel
	vbox          *fyne.Container
	scroll        *container.Scroll
	listContainer *fyne.Container
}

// NewAutocompleteEntry initialises an AutocompleteEntry with the given matcher and onSubmitted callbacks.
// If any children are passed, they appended to the entry. If the suggestion box is active, this will appear
// in front of the children.
func NewAutocompleteEntry(win fyne.Window, matcher func(string) []string, onSubmitted func(string) (cache bool), children ...fyne.CanvasObject) *AutocompleteEntry {
	e := &AutocompleteEntry{Matcher: matcher, OnSubmitted: onSubmitted}
	e.ExtendBaseWidget(e)
	e.refocusEntry = func() { win.Canvas().Focus(e) }
	e.commandHistory = NewCommandHistory(maxHistory)
	e.children = children
	e.OnChanged = e.onChanged

	e.vbox = container.NewVBox()
	e.scroll = container.NewVScroll(e.vbox)

	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.StrokeColor = theme.Color(theme.ColorNameSeparator)
	bg.StrokeWidth = 1
	bg.CornerRadius = theme.SelectionRadiusSize()

	e.listContainer = container.NewStack(bg, e.scroll)
	e.listContainer.Hide()

	return e
}

func (e *AutocompleteEntry) CreateRenderer() fyne.WidgetRenderer {
	e.ExtendBaseWidget(e)
	return &autocompleteEntryRenderer{e: e, inner: e.Entry.CreateRenderer()}
}

func (e *AutocompleteEntry) AcceptsTab() bool { return true }

func (e *AutocompleteEntry) TypedKey(ev *fyne.KeyEvent) {
	e.mu.Lock()
	open := e.shown
	e.mu.Unlock()

	switch ev.Name {
	case fyne.KeyDown:
		if open {
			e.moveSelection(+1)
			return
		}
		if e.Text == "" || e.commandHistory.Active() {
			if next, ok := e.commandHistory.Next(); ok {
				e.setTextAndCursor(next)
			}
		}
	case fyne.KeyUp:
		if open {
			e.moveSelection(-1)
			return
		}
		if e.Text == "" || e.commandHistory.Active() {
			if prev, ok := e.commandHistory.Prev(); ok {
				e.setTextAndCursor(prev)
			}
		}
	case fyne.KeyTab:
		if open {
			if driver, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
				if driver.CurrentKeyModifiers()&fyne.KeyModifierShift != 0 {
					e.moveSelection(-1)
					return
				}
			}

			e.moveSelection(+1)
		}
	case fyne.KeyReturn, fyne.KeyEnter:
		if open {
			if word := e.selectedWord(); word != "" {
				e.acceptSuggestion(word)
				return
			}
		}
		if e.OnSubmitted != nil {
			if cacheCmd := e.OnSubmitted(e.Text); cacheCmd {
				e.commandHistory.Add(e.Text)
			}
			e.SetText("")
		}
	case fyne.KeyEscape:
		if open {
			e.hideList()
		}
	default:
		e.Entry.TypedKey(ev)
	}

}

func (e *AutocompleteEntry) setTextAndCursor(text string) {
	e.SetText(text)
	e.CursorRow = 0
	e.CursorColumn = len([]rune(text))
	e.Refresh()
}

func (e *AutocompleteEntry) selectedWord() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.selIdx >= 0 && e.selIdx < len(e.items) {
		return e.items[e.selIdx]
	}
	return ""
}

func (e *AutocompleteEntry) moveSelection(delta int) {
	e.mu.Lock()
	n := len(e.items)
	if n == 0 {
		e.mu.Unlock()
		return
	}
	e.selIdx = (e.selIdx + delta + n) % n
	sel := e.selIdx
	e.mu.Unlock()
	e.updateHighlight(sel)
	e.scrollToRow(sel)
}

func (e *AutocompleteEntry) onChanged(text string) {
	if e.commandHistory.Active() {
		e.hideList()
		return
	}

	partial := lastWord(text)
	if partial == "" {
		e.hideList()
		return
	}
	matches := e.Matcher(partial)
	if len(matches) == 0 || (len(matches) == 1 && matches[0] == partial) {
		e.hideList()
		return
	}

	e.mu.Lock()
	e.shown = true
	e.items = matches
	e.selIdx = initialSelectedIdx
	e.mu.Unlock()

	e.rebuildLabels(matches)
	e.listContainer.Show()
}

func (e *AutocompleteEntry) hideList() {
	e.mu.Lock()
	e.shown = false
	e.items = nil
	e.selIdx = initialSelectedIdx
	e.mu.Unlock()
	e.listContainer.Hide()
}

func (e *AutocompleteEntry) acceptSuggestion(word string) {
	e.hideList()
	replaced := replaceLastWord(e.Text, word)
	e.SetText(replaced)
	e.OnChanged = e.onChanged
	e.CursorColumn = len([]rune(replaced))
	e.Refresh()
	e.refocusEntry()
}

// rebuildLabels reuses existing label widgets where possible to avoid
// allocating new objects on every keystroke. Only creates new labels when
// the match count grows beyond what we already have.
func (e *AutocompleteEntry) rebuildLabels(items []string) {
	// Grow the label pool if needed.
	for len(e.labels) < len(items) {
		e.labels = append(e.labels, NewTapLabel(nil))
	}

	// Update text, tap handler, and rebuild vbox objects.
	objs := make([]fyne.CanvasObject, len(items))
	for i, text := range items {
		l := e.labels[i]
		l.SetText(text)
		l.OnTap = func() { e.acceptSuggestion(text) }
		objs[i] = l
	}
	e.vbox.Objects = objs
	e.vbox.Refresh()
	e.updateHighlight(initialSelectedIdx)
	e.scroll.ScrollToTop()
}

// updateHighlight repaints labels to reflect the newly selected row.
func (e *AutocompleteEntry) updateHighlight(sel int) {
	e.mu.Lock()
	n := len(e.items)
	e.mu.Unlock()
	for i := 0; i < n && i < len(e.labels); i++ {
		if i == sel {
			e.labels[i].Importance = widget.HighImportance
		} else {
			e.labels[i].Importance = widget.MediumImportance
		}
		e.labels[i].Refresh()
	}
}

// scrollToRow scrolls the viewport so row sel is fully visible.
// Row height is constant (single-line labels), so the offset is exact.
func (e *AutocompleteEntry) scrollToRow(sel int) {
	rowH := e.rowHeight()
	top := rowH * float32(sel)
	bot := top + rowH
	viewH := e.scroll.Size().Height
	off := e.scroll.Offset.Y

	if top < off {
		e.scroll.ScrollToOffset(fyne.NewPos(0, top))
	} else if bot > off+viewH {
		e.scroll.ScrollToOffset(fyne.NewPos(0, bot-viewH))
	}
}

// rowHeight returns the pixel height of one label row inside the VBox.
// All rows are single-line labels so this is constant.
func (e *AutocompleteEntry) rowHeight() float32 {
	return widget.NewLabel("M").MinSize().Height + theme.Padding()
}

func (e *AutocompleteEntry) cursorPos() fyne.Position {
	row := e.CursorRow
	col := e.CursorColumn
	lines := strings.Split(e.Text, "\n")

	wordStartCol := col
	if row < len(lines) {
		runes := []rune(lines[row])
		if wordStartCol > len(runes) {
			wordStartCol = len(runes)
		}
		for wordStartCol > 0 && !isWordSep(runes[wordStartCol-1]) {
			wordStartCol--
		}
	}

	var prefix string
	if row < len(lines) {
		prefix = string([]rune(lines[row])[:wordStartCol])
	}

	style := fyne.TextStyle{}
	tw := fyne.MeasureText(prefix, theme.TextSize(), style).Width
	rowH := fyne.MeasureText("Mg", theme.TextSize(), style).Height + theme.InnerPadding()

	return fyne.NewPos(tw, e.MinSize().Height+float32(row)*rowH)
}

type autocompleteEntryRenderer struct {
	e     *AutocompleteEntry
	inner fyne.WidgetRenderer
}

func (r *autocompleteEntryRenderer) Layout(size fyne.Size) {
	entryHeight := r.inner.MinSize().Height
	r.inner.Layout(fyne.NewSize(size.Width, entryHeight))
	// Move the objects created by the entry renderer
	for _, o := range r.inner.Objects() {
		o.Move(fyne.NewPos(0, 0))
	}

	padding := theme.Padding()
	y := entryHeight + padding
	for _, child := range r.e.children {
		csize := child.MinSize()
		// Position children below entry
		child.Move(fyne.NewPos((size.Width-csize.Width)/2, y))
		child.Resize(csize)

		y += csize.Height + padding
	}

	r.positionList()
}

func (r *autocompleteEntryRenderer) positionList() {
	r.e.mu.Lock()
	shown := r.e.shown
	n := len(r.e.items)
	r.e.mu.Unlock()

	if !shown || n == 0 {
		r.e.listContainer.Hide()
		return
	}

	rowH := r.e.rowHeight()
	visible := min(n, maxVisible)
	r.e.listContainer.Move(r.e.cursorPos())
	r.e.listContainer.Resize(fyne.NewSize(200, rowH*float32(visible)))
}

func (r *autocompleteEntryRenderer) MinSize() fyne.Size { return r.inner.MinSize() }

func (r *autocompleteEntryRenderer) Refresh() {
	r.inner.Refresh()
	r.positionList()
}

func (r *autocompleteEntryRenderer) Destroy() { r.inner.Destroy() }

func (r *autocompleteEntryRenderer) Objects() []fyne.CanvasObject {
	return append(append(r.inner.Objects(), r.e.children...), r.e.listContainer)
}

func isWordSep(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }

func lastWord(s string) string {
	trimmed := strings.TrimRight(s, " \t")
	if trimmed == "" {
		return ""
	}
	if idx := strings.LastIndexAny(trimmed, " \t\n"); idx >= 0 {
		return trimmed[idx+1:]
	}
	return trimmed
}

func replaceLastWord(text, replacement string) string {
	trimmed := strings.TrimRight(text, " \t")
	if idx := strings.LastIndexAny(trimmed, " \t\n"); idx >= 0 {
		return trimmed[:idx+1] + replacement + " "
	}
	return replacement + " "
}
