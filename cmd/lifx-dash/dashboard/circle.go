package dashboard

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type ClickableCircle struct {
	widget.BaseWidget
	Circle   *canvas.Circle
	OnTapped func()
}

func NewClickableCircle(c *canvas.Circle, tapped func()) *ClickableCircle {
	w := &ClickableCircle{
		Circle:   c,
		OnTapped: tapped,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *ClickableCircle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.Circle)
}

func (w *ClickableCircle) Tapped(*fyne.PointEvent) {
	if w.OnTapped != nil {
		w.OnTapped()
	}
}
