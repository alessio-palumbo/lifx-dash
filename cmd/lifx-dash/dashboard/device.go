package dashboard

import (
	"fmt"
	"image/color"
	"log"
	"math"
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
	v.device = &d
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
		r, g, b := KelvinToRGB(int(d.Color.Kelvin))
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	}
	r, g, b := HSBToRGB(d.Color.Hue, d.Color.Saturation, d.Color.Brightness)
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

func HSBToRGB(h, s, b float64) (int, int, int) {
	s, b = s/100, b/100
	if s == 0.0 {
		return int(b * 255), int(b * 255), int(b * 255)
	}

	h = math.Mod(h, 360)
	hi := math.Floor(h / 60)
	f := h/60 - hi
	p := b * (1 - s)
	q := b * (1 - f*s)
	t := b * (1 - (1-f)*s)

	switch int(hi) {
	case 0:
		return int(b * 255), int(t * 255), int(p * 255)
	case 1:
		return int(q * 255), int(b * 255), int(p * 255)
	case 2:
		return int(p * 255), int(b * 255), int(t * 255)
	case 3:
		return int(p * 255), int(q * 255), int(b * 255)
	case 4:
		return int(t * 255), int(p * 255), int(b * 255)
	case 5:
		return int(b * 255), int(p * 255), int(q * 255)
	}

	return 0, 0, 0
}

// KelvinToRGB converts a color temperature in Kelvin to an RGB color.
// It uses a standard approximation suitable for many applications,
// but accuracy is best between 1000K and 40000K.
func KelvinToRGB(kelvin int) (r, g, b int) {
	temp := int(math.Round(float64(kelvin) / 100.0))

	// Red
	if temp <= 66 {
		r = 255
	} else {
		r = temp - 60
		r = int(329.698727446 * math.Pow(float64(r), -0.1332047592))
		r = min(max(r, 0), 255)
	}

	// Green
	if temp <= 66 {
		g = temp
		g = int(99.4708025861*math.Log(float64(g)) - 161.1195681661)
		g = min(max(g, 0), 255)
	} else {
		g = temp - 60
		g = int(288.1221695283 * math.Pow(float64(g), -0.0755148492))
		g = min(max(g, 0), 255)
	}

	// Blue
	if temp >= 66 {
		b = 255
	} else if temp <= 19 {
		b = 0
	} else {
		b = temp - 10
		b = int(138.5177312231*math.Log(float64(b)) - 305.0447927307)
		b = min(max(b, 0), 255)
	}

	return int(r), int(g), int(b)
}
