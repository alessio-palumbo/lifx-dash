package dashboard

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// AutoCompleteEntryDialog returns a modal with an embedded AutoCompleteEntryDialog.
type AutoCompleteEntryDialog struct {
	*dialog.CustomDialog
	onShow func()
}

// NewAutoCompleteEntryDialog returns an instance of AutoCompleteEntryDialog.
func NewAutoCompleteEntryDialog(win fyne.Window, placeHolder string, matcher func(string) []string, onSubmit func(string) bool) *AutoCompleteEntryDialog {
	closeBtn := widget.NewButton("Close", nil)
	entry := NewAutocompleteEntry(
		win,
		matcher,
		onSubmit,
		closeBtn,
	)
	entry.SetPlaceHolder(placeHolder)
	d := dialog.NewCustomWithoutButtons("", entry, win)
	closeBtn.OnTapped = func() { d.Dismiss() }

	return &AutoCompleteEntryDialog{CustomDialog: d, onShow: func() { win.Canvas().Focus(entry) }}
}

func (d *AutoCompleteEntryDialog) Show() {
	d.CustomDialog.Show()
	d.onShow()
}
