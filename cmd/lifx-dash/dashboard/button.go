package dashboard

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type Button struct {
	widget.Button
	tappedSecondary func(*Button)
}

func NewButton(label string, tapped func(), tappedSecondary func(*Button)) *Button {
	b := &Button{}
	b.ExtendBaseWidget(b)
	b.Text = label
	b.OnTapped = tapped
	b.tappedSecondary = tappedSecondary
	return b
}

func (b *Button) TappedSecondary(*fyne.PointEvent) {
	if b.tappedSecondary != nil {
		b.tappedSecondary(b)
	}
}
