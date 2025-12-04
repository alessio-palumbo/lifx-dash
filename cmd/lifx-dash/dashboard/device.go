package dashboard

import (
	"fmt"
	"image/color"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
	"github.com/alessio-palumbo/lifxlan-go/pkg/messages"
	"github.com/alessio-palumbo/lifxprotocol-go/gen/protocol/packets"
)

const (
	minContrastBrightness = 40
	freezeUpdatesDuration = 3 * time.Second
)

type deviceView struct {
	content *fyne.Container
	label   *StatusLabel
	device  *device.Device

	brightness binding.Float

	mu            sync.RWMutex
	internalColor *device.Color
	cells         []*ZoneCell
	freezeUntil   time.Time
}

func newDeviceView(parentWin fyne.Window, ctrl Controller, d *device.Device) *deviceView {
	statusLabel := NewStatusLabel(parentWin, d)
	view := &deviceView{
		label:         statusLabel,
		device:        d,
		internalColor: &device.Color{},
		brightness:    binding.NewFloat(),
	}
	*view.internalColor = d.Color
	view.brightness.Set(d.Color.Brightness)

	if d.Type == device.DeviceTypeSwitch {
		view.content = container.NewPadded(container.NewVBox(statusLabel))
		return view
	}

	toggleBtn := widget.NewButton("Toggle", func() {
		if err := toggle(ctrl, view.device); err != nil {
			log.Println(err)
			return
		}

		view.freezeUpdates()
		// optimistic update of local copy
		view.device.PoweredOn = !view.device.PoweredOn
		view.refreshUI()
	})

	brightnessSlider := NewSliderWithData("%.0f%%", 1, 100, 1, view.brightness, func(v float64) error {
		view.freezeUpdates()
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

		switch d.LightType {
		case device.LightTypeMatrix:
			zones := make([]packets.LightHsbk, d.MatrixProperties.Width*d.MatrixProperties.Height)
			if len(d.MatrixProperties.ChainState) > 0 {
				copy(zones, d.MatrixProperties.ChainState[0][:])
			}
			grid := newZonesGrid(parentWin, view, zones, d.MatrixProperties.Width)
			applyBtn := newApplyZonesButton(
				view.cells,
				func() {
					var colors [64]packets.LightHsbk
					for i, c := range view.cells {
						if c.Selected {
							colors[i] = c.SelectedColor.ToDeviceColor()
						}
					}

					ctrl.Send(d.Serial, messages.SetMatrixColors(0, 1, d.MatrixProperties.Width, colors, time.Millisecond))
				},
			)

			modalContent.Add(widget.NewLabel("Matrix"))
			modalContent.Add(grid)
			modalContent.Add(applyBtn)
		case device.LightTypeMultiZone:
			grid := newZonesGrid(parentWin, view, d.MultizoneProperties.Zones, 8)
			applyBtn := newApplyZonesButton(
				view.cells,
				func() {
					colors := make([]packets.LightHsbk, len(view.cells))
					for i, c := range view.cells {
						if c.Selected {
							colors[i] = c.SelectedColor.ToDeviceColor()
						}
					}
					msgs := messages.SetMultizoneExtendedColors(0, colors, time.Millisecond)
					for _, msg := range msgs {
						ctrl.Send(d.Serial, msg)
					}
				},
			)

			modalContent.Add(widget.NewLabel("Zones"))
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

func (v *deviceView) LastUpdatedAt() time.Time {
	return v.device.LastUpdatedAt
}

func (v *deviceView) Update(d device.Device) {
	*v.device = d
	v.mu.RLock()
	if v.device.LastUpdatedAt.After(v.freezeUntil) {
		v.refreshUI()
	}
	v.mu.RUnlock()
}

func (v *deviceView) refreshUI() {
	v.label.UpdateStatus(v.device.Label, deviceColorToRGBA(v.device))

	if v.brightness != nil {
		v.brightness.Set(v.device.Color.Brightness)
	}
}

// freezeUpdates prevents aggressive updates from racing with user interactions
// by momentarily avoid refreshing the UI due to state updates, which might be stale.
func (v *deviceView) freezeUpdates() {
	v.mu.Lock()
	v.freezeUntil = time.Now().Add(freezeUpdatesDuration)
	v.mu.Unlock()
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

func toggle(ctrl Controller, d *device.Device) error {
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

// colorToRGBA is used to display the color of the device in the UI.
// Brightness adjustment is performed to make sure the color is visible.
func colorToRGBA(c device.Color) color.RGBA {
	// sets the minimum brightness of the displayed color to an acceptable contrast level.
	c.Brightness = max(c.Brightness, minContrastBrightness)

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

func newZonesGrid(parentWin fyne.Window, view *deviceView, zones []packets.LightHsbk, gridWidth int) *fyne.Container {
	view.cells = make([]*ZoneCell, len(zones))
	grid := container.NewGridWithColumns(gridWidth)

	for i := range zones {
		color := device.NewColor(zones[i])
		cell := NewZoneCell(parentWin, &color, func() device.Color { return view.getInternalColor() })
		view.cells[i] = cell
		grid.Add(cell)
	}

	return grid
}
func newApplyZonesButton(cells []*ZoneCell, onTap func()) *Button {
	label := "Apply Zones"
	return NewButton(
		label,
		onTap,
		func(b *Button) {
			var colors []string
			for _, c := range cells {
				colors = append(colors, colorToLabel(c.SelectedColor))
			}
			fyne.CurrentApp().Clipboard().SetContent(fmt.Sprintf("%s", colors))
			b.SetText("Zones Copied!")
			fyne.Do(func() {
				time.Sleep(400 * time.Millisecond)
				b.SetText(label)
			})
		},
	)
}
