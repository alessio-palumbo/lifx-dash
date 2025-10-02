package dashboard

import (
	"fmt"
	"image/color"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/controller"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
	"github.com/alessio-palumbo/lifxlan-go/pkg/messages"
)

type deviceView struct {
	content *fyne.Container
	label   *StatusLabel
	device  *device.Device
}

func newDeviceView(parentWin fyne.Window, ctrl *controller.Controller, d *device.Device) *deviceView {
	statusLabel := NewStatusLabel(parentWin, d)
	view := &deviceView{
		label:  statusLabel,
		device: d,
	}

	toggleBtn := widget.NewButton("Toggle", func() {
		if err := toggle(ctrl, view.device); err != nil {
			log.Println(err)
			return
		}
		// optimistic update of local copy
		view.device.PoweredOn = !view.device.PoweredOn
		view.refreshUI()
	})

	brightnessSlider := NewSlider("%.0f%%", 1, 100, 1, d.Color.Brightness, func(v float64) error {
		return ctrl.Send(d.Serial, messages.SetColor(nil, nil, &v, nil, time.Millisecond, 0))
	})

	settingsBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ColorPaletteIcon()).Resource, func() {
		hue := NewSlider("%.0f", 0, 360, 1, d.Color.Hue, func(v float64) error {
			return ctrl.Send(d.Serial, messages.SetColor(&v, nil, nil, nil, time.Millisecond, 0))
		})
		sat := NewSlider("%.0f%%", 0, 100, 1, d.Color.Saturation, func(v float64) error {
			return ctrl.Send(d.Serial, messages.SetColor(nil, &v, nil, nil, time.Millisecond, 0))
		})
		kelvin := NewSlider("%.0fK", 1500, 9000, 100, float64(d.Color.Kelvin), func(v float64) error {
			k := uint16(v)
			return ctrl.Send(d.Serial, messages.SetColor(nil, nil, nil, &k, time.Millisecond, 0))
		})

		header := container.NewCenter(widget.NewLabel("Colour Settings"))
		modalContent := container.NewVBox(
			header,
			widget.NewLabel("Hue"),
			hue,
			widget.NewLabel("Saturation"),
			sat,
			widget.NewLabel("Kelvin"),
			kelvin,
		)

		d := dialog.NewCustom("", "Close", modalContent, parentWin)
		d.Resize(fyne.NewSize(300, d.MinSize().Height))
		d.Show()
	})

	view.content = container.NewPadded(container.NewVBox(statusLabel, brightnessSlider, NewHItemWithSideLabel(toggleBtn, settingsBtn)))
	return view
}

func (v *deviceView) LastSeenAt() time.Time {
	return v.device.LastSeenAt
}

func (v *deviceView) Update(d device.Device) {
	*v.device = d
	v.refreshUI()
}

func (v *deviceView) refreshUI() {
	v.label.UpdateStatus(v.device.Label, deviceColorToRGBA(v.device))
}

func toggle(ctrl *controller.Controller, d *device.Device) error {
	if d.PoweredOn {
		return ctrl.Send(d.Serial, messages.SetPowerOff())
	}
	return ctrl.Send(d.Serial, messages.SetPowerOn())
}

func deviceColorToRGBA(d *device.Device) color.RGBA {
	if !d.PoweredOn {
		return color.RGBA{A: 255}
	}

	if d.Color.Saturation == 0 {
		r, g, b := d.Color.KelvinToRGB()
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	}
	r, g, b := d.Color.HSBToRGB()
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func deviceInfo(d *device.Device) string {
	return fmt.Sprintf("Serial: %s\n"+
		"IP: %s\n"+
		"ProductID: %d\n"+
		"Group: %s\n"+
		"Location: %s",
		d.Serial, d.Address.IP.String(), d.ProductID, d.Group, d.Location,
	)
}
