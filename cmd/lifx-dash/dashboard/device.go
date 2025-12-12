package dashboard

import (
	"fmt"
	"image/color"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	minContrastBrightness = 80
	freezeUpdatesDuration = 10 * time.Second
	modalLabelWidth       = 80
)

type deviceView struct {
	content *fyne.Container
	label   *StatusLabel
	device  *device.Device

	brightness  binding.Float
	lightCircle *canvas.Circle

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
		return ctrl.Send(d.Serial, messages.SetColor(nil, nil, &v, nil, time.Millisecond, 0))
	})

	settingsBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ColorPaletteIcon()).Resource, func() {
		hue := NewSlider("%.0f", 0, 360, 1, d.Color.Hue, func(v float64) error {
			view.setInternalColor(func(c *device.Color) { c.Hue = v })
			view.updateLightCircle()
			// Do not send a message when editing zones only.
			if view.updateIfSelectedCells() {
				return nil
			}
			return ctrl.Send(d.Serial, messages.SetColor(&v, nil, nil, nil, time.Millisecond, 0))
		})
		sat := NewSlider("%.0f%%", 0, 100, 1, d.Color.Saturation, func(v float64) error {
			view.setInternalColor(func(c *device.Color) { c.Saturation = v })
			view.updateLightCircle()
			// Do not send a message when editing zones only.
			if view.updateIfSelectedCells() {
				return nil
			}
			return ctrl.Send(d.Serial, messages.SetColor(nil, &v, nil, nil, time.Millisecond, 0))
		})
		bri := NewSlider("%.0f%%", 1, 100, 1, d.Color.Brightness, func(v float64) error {
			view.setInternalColor(func(c *device.Color) { c.Brightness = v })
			view.updateLightCircle()
			// Do not send a message when editing zones only.
			if view.updateIfSelectedCells() {
				return nil
			}
			return ctrl.Send(d.Serial, messages.SetColor(nil, nil, &v, nil, time.Millisecond, 0))
		})
		kelvin := NewSlider("%.0fK", 1500, 9000, 100, float64(d.Color.Kelvin), func(v float64) error {
			k := uint16(v)
			view.setInternalColor(func(c *device.Color) { c.Kelvin = k })
			view.updateLightCircle()
			// Do not send a message when editing zones only.
			if view.updateIfSelectedCells() {
				return nil
			}
			return ctrl.Send(d.Serial, messages.SetColor(nil, nil, nil, &k, time.Millisecond, 0))
		})

		header := container.NewCenter(widget.NewLabelWithStyle("Colour Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
		grid := container.NewVBox(
			LabelledSlider("Hue", modalLabelWidth, hue),
			LabelledSlider("Saturation", modalLabelWidth, sat),
			LabelledSlider("Brightness", modalLabelWidth, bri),
			LabelledSlider("Kelvin", modalLabelWidth, kelvin),
		)
		modalContent := container.NewVBox(
			header,
			grid,
		)

		switch d.LightType {
		case device.LightTypeSingleZone:
			view.lightCircle = canvas.NewCircle(deviceColorToRGBA(view.device))
			circleContainer := container.NewGridWrap(fyne.NewSize(100, 100), view.lightCircle)
			modalContent.Add(withTopMargin(container.NewCenter(circleContainer), 30))

		case device.LightTypeMatrix:
			if d.MatrixProperties.Width == 0 {
				break
			}
			zones := make([]packets.LightHsbk, d.MatrixProperties.Width*d.MatrixProperties.Height)
			if len(d.MatrixProperties.ChainState) > 0 {
				copy(zones, d.MatrixProperties.ChainState[0][:])
			}
			grid := newZonesGrid(parentWin, view, zones, d.MatrixProperties.Width)
			applyBtn := widget.NewButton("Apply Zones",
				func() {
					var colors [64]packets.LightHsbk
					for i := range view.cells {
						c := view.cells[i]
						colors[i] = c.ApplyColor()
					}

					ctrl.Send(d.Serial, messages.SetMatrixColors(0, 1, d.MatrixProperties.Width, colors, time.Millisecond))
				},
			)

			modalContent.Add(withTopMargin(grid, 30))
			modalContent.Add(withTopMargin(NewHItemWithSideLabel(applyBtn, newGridActionsButtons(view)), 10))

		case device.LightTypeMultiZone:
			grid := newZonesGrid(parentWin, view, d.MultizoneProperties.Zones, 8)
			applyBtn := widget.NewButton("Apply Zones",
				func() {
					colors := make([]packets.LightHsbk, len(view.cells))
					for i := range view.cells {
						c := view.cells[i]
						colors[i] = c.ApplyColor()
					}

					msgs := messages.SetMultizoneExtendedColors(0, colors, time.Millisecond)
					for _, msg := range msgs {
						ctrl.Send(d.Serial, msg)
					}
				},
			)

			modalContent.Add(withTopMargin(grid, 30))
			modalContent.Add(withTopMargin(NewHItemWithSideLabel(applyBtn, newGridActionsButtons(view)), 10))
		}

		d := dialog.NewCustom("", "Close", container.NewPadded(modalContent), parentWin)
		d.Resize(fyne.NewSize(350, 500))
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

func (v *deviceView) updateLightCircle() {
	if v.lightCircle == nil {
		return
	}
	v.lightCircle.FillColor = colorToRGBA(v.getInternalColor())
	v.lightCircle.Refresh()
}

func (v *deviceView) updateIfSelectedCells() (updated bool) {
	if len(v.cells) == 0 {
		return
	}
	color := v.getInternalColor()
	for _, c := range v.cells {
		c.SelectedColor = &color
		if c.Selected {
			c.Refresh()
			updated = true
		}
	}
	return
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
	if c.Brightness == 0 {
		return color.RGBA{}
	}

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

func newZonesGrid(parentWin fyne.Window, view *deviceView, zones []packets.LightHsbk, gridWidth int) *ZoneGrid {
	view.cells = make([]*ZoneCell, len(zones))
	grid := container.NewGridWithColumns(gridWidth)

	for i := range zones {
		color := device.NewColor(zones[i])
		cell := NewZoneCell(parentWin, &color, func() device.Color { return view.getInternalColor() })
		view.cells[i] = cell
		grid.Add(cell)
	}

	return NewZoneGrid(len(zones)/gridWidth, gridWidth, view.cells)
}

func newGridActionsButtons(view *deviceView) *fyne.Container {
	copyBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ContentCopyIcon()).Resource, func() {
		var colors []string
		for _, c := range view.cells {
			colors = append(colors, colorToLabel(c.Color))
		}
		fyne.CurrentApp().Clipboard().SetContent(fmt.Sprintf("%s", colors))
	})

	confirmSelectedBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ConfirmIcon()).Resource, func() {
		for _, c := range view.cells {
			if c.Selected {
				c.Color = c.SelectedColor
				c.Selected = false
				c.Refresh()
			} else {
				c.SelectedColor = c.Color
			}
		}
	})

	clearSelectedBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ContentUndoIcon()).Resource, func() {
		for _, c := range view.cells {
			if c.Selected {
				c.Selected = false
				c.Refresh()
			}
		}
	})

	clearGridBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.DeleteIcon()).Resource, func() {
		deviceColor := view.device.Color
		for i := range view.cells {
			c := view.cells[i]
			if c.Selected {
				c.Selected = false
			}
			c.Color = &deviceColor
			c.Refresh()
		}
	})

	return newButtonGrid(copyBtn, confirmSelectedBtn, clearSelectedBtn, clearGridBtn)
}

func newButtonGrid(buttons ...*widget.Button) *fyne.Container {
	grid := container.NewGridWithColumns(len(buttons))
	for _, b := range buttons {
		grid.Add(b)
	}
	return grid
}
func withTopMargin(content fyne.CanvasObject, px float32) fyne.CanvasObject {
	pad := canvas.NewRectangle(color.Transparent)
	pad.SetMinSize(fyne.NewSize(1, px))
	return container.NewBorder(pad, nil, nil, nil, content)
}
