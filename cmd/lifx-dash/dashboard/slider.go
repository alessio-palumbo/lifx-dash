package dashboard

import (
	"fmt"
	"log"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
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

func NewSliderWithData(labelFmt string, min, max, step float64, v binding.Float, sendFunc func(v float64) error) *fyne.Container {
	sliderLabel := widget.NewLabel(fmt.Sprintf(labelFmt, 0))
	value, _ := v.Get()
	sliderLabel.SetText(fmt.Sprintf(labelFmt, float64(value)))

	slider := widget.NewSliderWithData(min, max, v)
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

func NewSliderWithEntry(labelFmt string, min, max, step, v float64, sendFunc func(v float64) error) *fyne.Container {
	entry := NewEntry(50)
	entry.SetText(fmt.Sprintf(labelFmt, v))
	entry.SetMinRowsVisible(1)

	slider := widget.NewSlider(min, max)
	slider.Value = v
	slider.Step = step

	// Assign to a local variable to avoid closure capturing
	cb := sendFunc

	slider.OnChanged = func(value float64) {
		log.Println("setting again")
		entry.SetText(fmt.Sprintf(labelFmt, value))
		if err := cb(value); err != nil {
			log.Println(err)
		}
	}

	entry.OnSubmitted = func(s string) {
		v, err := strconv.Atoi(s)
		if err == nil {
			// Clamp
			if v < int(slider.Min) {
				v = int(slider.Min)
			}
			if v > int(slider.Max) {
				v = int(slider.Max)
			}
			slider.SetValue(float64(v))
		}

		// Resync entry from slider
		entry.SetText(fmt.Sprintf(labelFmt, slider.Value))
	}

	return NewHItemWithSideLabel(slider, entry)
}

func NewHItemWithSideLabel(item, label fyne.CanvasObject) *fyne.Container {
	return container.NewBorder(nil, nil, nil, label, item)
}

func LabelledSlider(label string, labelWidth int, slider *fyne.Container) *fyne.Container {
	fixedWidthLabel := container.NewGridWrap(fyne.NewSize(float32(labelWidth), 0), widget.NewLabel(label))
	return container.NewBorder(
		nil, nil,
		fixedWidthLabel, nil,
		slider,
	)
}

type SelectEntry struct {
	widget.Entry
	minWidth float32
}

func NewEntry(minWidth float32) *SelectEntry {
	e := &SelectEntry{minWidth: minWidth}
	e.ExtendBaseWidget(e)
	return e
}

func (e *SelectEntry) FocusGained() {
	e.Entry.FocusGained()
	// Emulate a double click event and select all text on focus.
	e.DoubleTapped(nil)
}

func (e *SelectEntry) FocusLost() {
	e.Entry.FocusLost()
	// Emulate a double click event and select all text on focus.
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	log.Println("LOst")
}

func (e *SelectEntry) MinSize() fyne.Size {
	min := e.Entry.MinSize()
	if min.Width < e.minWidth {
		min.Width = e.minWidth
	}
	return min
}
