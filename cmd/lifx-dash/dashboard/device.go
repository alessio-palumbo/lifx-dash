package dashboard

import (
	"fmt"
	"image/color"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/controller"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
	"github.com/alessio-palumbo/lifxlan-go/pkg/messages"
	"github.com/alessio-palumbo/lifxprotocol-go/gen/protocol/packets"
)

type deviceView struct {
	content *fyne.Container
	label   *StatusLabel
	device  *device.Device

	mu            sync.RWMutex
	internalColor *device.Color
	cells         []*ZoneCell
}

func newDeviceView(parentWin fyne.Window, ctrl *controller.Controller, d *device.Device) *deviceView {
	statusLabel := NewStatusLabel(parentWin, d)
	view := &deviceView{
		label:         statusLabel,
		device:        d,
		internalColor: &device.Color{},
	}
	*view.internalColor = d.Color

	if d.Type == device.DeviceTypeSwitch {
		view.content = container.NewPadded(container.NewVBox(statusLabel))
		return view
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
		view.setInternalColor(func(c *device.Color) { c.Brightness = v })
		return ctrl.Send(d.Serial, messages.SetColor(nil, nil, &v, nil, time.Millisecond, 0))
	})

	settingsBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ColorPaletteIcon()).Resource, func() {
		hue := NewSlider("%.0f", 0, 360, 1, d.Color.Hue, func(v float64) error {
			view.setInternalColor(func(c *device.Color) { c.Hue = v })
			return ctrl.Send(d.Serial, messages.SetColor(&v, nil, nil, nil, time.Millisecond, 0))
		})
		sat := NewSlider("%.0f%%", 0, 100, 1, d.Color.Saturation, func(v float64) error {
			view.setInternalColor(func(c *device.Color) { c.Saturation = v })
			return ctrl.Send(d.Serial, messages.SetColor(nil, &v, nil, nil, time.Millisecond, 0))
		})
		kelvin := NewSlider("%.0fK", 1500, 9000, 100, float64(d.Color.Kelvin), func(v float64) error {
			k := uint16(v)
			view.setInternalColor(func(c *device.Color) { c.Kelvin = k })
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

		if d.LightType == device.LightTypeMatrix {
			width, height := d.MatrixProperties.Width, d.MatrixProperties.Height
			matrixSize := width * height
			view.cells = make([]*ZoneCell, matrixSize)
			grid := container.NewGridWithColumns(width)

			for i := range matrixSize {
				cell := NewZoneCell(parentWin, func() device.Color { return view.getInternalColor() })
				view.cells[i] = cell
				grid.Add(cell)
			}

			applyBtnLabel := "Apply Matrix"
			applyBtn := NewButton(
				applyBtnLabel,
				func() {
					var colors [64]packets.LightHsbk
					for i, c := range view.cells {
						if c.Selected {
							colors[i] = c.SelectedColor.ToDeviceColor()
						}
					}

					ctrl.Send(d.Serial, messages.SetMatrixColors(0, 1, width, colors, time.Millisecond))
				},
				func(b *Button) {
					var colors []string
					for _, c := range view.cells {
						colors = append(colors, colorToLabel(c.SelectedColor))
					}
					fyne.CurrentApp().Clipboard().SetContent(fmt.Sprintf("%s", colors))
					b.SetText("Matrix Copied!")
					fyne.Do(func() {
						time.Sleep(400 * time.Millisecond)
						b.SetText(applyBtnLabel)
					})
				},
			)

			modalContent.Add(widget.NewLabel("Matrix Grid"))
			modalContent.Add(grid)
			modalContent.Add(applyBtn)
		}

		d := dialog.NewCustom("", "Close", container.NewScroll(container.NewPadded(modalContent)), parentWin)
		d.Resize(fyne.NewSize(300, 500))
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

func (v *deviceView) getInternalColor() device.Color {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return *v.internalColor
}

func (v *deviceView) setInternalColor(f func(*device.Color)) {
	v.mu.Lock()
	f(v.internalColor)
	v.mu.Unlock()
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
	return colorToRGBA(d.Color)
}

func colorToRGBA(c device.Color) color.RGBA {
	if c.Saturation == 0 {
		r, g, b := c.KelvinToRGB()
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	}
	r, g, b := c.HSBToRGB()
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func deviceInfo(d *device.Device) string {
	return fmt.Sprintf("Serial: %s\n"+
		"IP: %s\n"+
		"ProductID: %d\n"+
		"Group: %s\n"+
		"Location: %s\n"+
		"RSSI: %s",
		d.Serial, d.Address.IP.String(), d.ProductID, d.Group, d.Location, d.WifiRSSI,
	)
}
