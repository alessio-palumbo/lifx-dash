package dashboard

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func NewSlider(labelFmt string, min, max, step, v float64, sendFunc func(v float64) error) *fyne.Container {
	sliderLabel := widget.NewLabel(fmt.Sprintf(labelFmt, v))
	slider := widget.NewSlider(min, max)
	slider.Value = v
	slider.Step = step

	// Assign to a local variable to avoid closure capturing
	cb := sendFunc

	slider.OnChanged = func(value float64) {
		sliderLabel.SetText(fmt.Sprintf(labelFmt, value))
		if err := cb(value); err != nil {
			log.Println(err)
		}
	}

	return NewHItemWithSideLabel(slider, sliderLabel)
}

func NewHItemWithSideLabel(item, label fyne.CanvasObject) *fyne.Container {
	return container.NewBorder(nil, nil, nil, label, item)
}
