package dashboard

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/messages"
	"github.com/alessio-palumbo/lifxlan-go/pkg/protocol"
)

type Slider struct {
	label   *widget.Label
	content *fyne.Container
}

func NewSlider(label string, v float64, sendFunc func(msg *protocol.Message) error) *Slider {
	sliderLabel := widget.NewLabel(fmt.Sprintf("%s: %.0f%%", label, v))
	slider := widget.NewSlider(1, 100)
	slider.Value = v
	slider.Step = 1

	slider.OnChanged = func(value float64) {
		sliderLabel.SetText(fmt.Sprintf("%s: %.0f%%", label, value))
		if err := sendFunc(messages.SetColor(nil, nil, &value, nil, time.Millisecond, 0)); err != nil {
			// TODO Log error
		}
	}

	return &Slider{
		label:   sliderLabel,
		content: container.NewVBox(sliderLabel, slider),
	}
}
