package dashboard

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
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
	"github.com/alessio-palumbo/lifxlan-go/pkg/protocol"
	"github.com/alessio-palumbo/lifxprotocol-go/gen/protocol/packets"
)

const (
	gridMinBrightness = 50.0
	gridMaxBrightness = 100.0
	gridGamma         = 2.2

	freezeUpdatesDuration = 2 * time.Second
	modalLabelWidth       = 90

	unsetColorValue = -1
)

var emptyLightState = packets.LightHsbk{}

type deviceView struct {
	content   *fyne.Container
	label     *StatusLabel
	device    *device.Device
	parentWin fyne.Window

	brightness  binding.Float
	lightCircle *canvas.Circle

	settingsBtn *widget.Button

	mu            sync.RWMutex
	internalColor *device.Color
	// zoneSelectionColor defaults to -1 for any unselected field allowing
	// changing only brightness or saturation or hue for the selected cells.
	zoneSelectionColor    *device.Color
	zoneSetPackets        func(i int, colors []packets.LightHsbk) []*protocol.Message
	selectedEffect        *EffectDescriptor
	selectedEffectStopper func()
	grids                 []*ZoneGrid
	activeGrid            int
	freezeUntil           time.Time
}

func newDeviceView(parentWin fyne.Window, ctrl Controller, d *device.Device) *deviceView {
	statusLabel := NewStatusLabel(parentWin, d)
	view := &deviceView{
		label:              statusLabel,
		device:             d,
		parentWin:          parentWin,
		internalColor:      &device.Color{},
		zoneSelectionColor: &device.Color{Hue: unsetColorValue, Saturation: unsetColorValue, Brightness: unsetColorValue},
		brightness:         binding.NewFloat(),
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

	brightnessSlider := NewSliderWithUIBinding("%.0f%%", 1, 100, 1, d.Color.Brightness, view.brightness, func(v float64) error {
		view.freezeUpdates()
		return ctrl.Send(d.Serial, messages.SetColor(nil, nil, &v, nil, time.Millisecond, 0))
	})

	view.settingsBtn = widget.NewButtonWithIcon("", widget.NewIcon(theme.ColorPaletteIcon()).Resource, func() {
		sliders := container.NewVBox()
		if view.device.ColorProperties.HasColor {
			hue := NewSliderWithEntry("%.0f", 0, 360, 1, d.Color.Hue, func(v float64) error {
				view.setInternalColor(func(c *device.Color) { c.Hue = v })
				view.updateLightCircle()
				// Do not send a message when editing zones only.
				if view.updateIfSelectedCells() {
					return nil
				}
				return ctrl.Send(d.Serial, messages.SetColor(&v, nil, nil, nil, time.Millisecond, 0))
			})
			sliders.Add(LabelledSlider("Hue", modalLabelWidth, hue))

			sat := NewSliderWithEntry("%.0f", 0, 100, 1, d.Color.Saturation, func(v float64) error {
				view.setInternalColor(func(c *device.Color) { c.Saturation = v })
				view.updateLightCircle()
				// Do not send a message when editing zones only.
				if view.updateIfSelectedCells() {
					return nil
				}
				return ctrl.Send(d.Serial, messages.SetColor(nil, &v, nil, nil, time.Millisecond, 0))
			})
			sliders.Add(LabelledSlider("Saturation", modalLabelWidth, sat))
		}

		bri := NewSliderWithEntry("%.0f", 1, 100, 1, d.Color.Brightness, func(v float64) error {
			view.setInternalColor(func(c *device.Color) { c.Brightness = v })
			view.updateLightCircle()
			// Do not send a message when editing zones only.
			if view.updateIfSelectedCells() {
				return nil
			}
			return ctrl.Send(d.Serial, messages.SetColor(nil, nil, &v, nil, time.Millisecond, 0))
		})
		sliders.Add(LabelledSlider("Brightness", modalLabelWidth, bri))

		kMin, kMax := float64(d.ColorProperties.TemperatureRange.Min), float64(d.ColorProperties.TemperatureRange.Max)
		kValue := float64(d.Color.Kelvin)
		// Handle devices with fixed kelvin which returns incorrect kelvin color.
		if kMin == kMax {
			kValue = kMin
		}
		kelvin := NewSliderWithEntry("%.0f", kMin, kMax, 100, kValue, func(v float64) error {
			k := uint16(v)
			view.setInternalColor(func(c *device.Color) { c.Kelvin = k })
			view.updateLightCircle()
			// Do not send a message when editing zones only.
			if view.updateIfSelectedCells() {
				return nil
			}
			return ctrl.Send(d.Serial, messages.SetColor(nil, nil, nil, &k, time.Millisecond, 0))
		})
		sliders.Add(LabelledSlider("Kelvin", modalLabelWidth, kelvin))

		modalContent := container.NewVBox(
			container.NewCenter(widget.NewLabelWithStyle("Colour Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
			sliders,
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

			chainBox := container.NewVBox()
			view.grids = make([]*ZoneGrid, len(d.MatrixProperties.ChainZones))
			for i := range d.MatrixProperties.ChainZones {
				zones := make([]packets.LightHsbk, d.MatrixProperties.NZones)
				copy(zones, d.MatrixProperties.ChainZones[i])
				view.grids[i] = NewZoneGrid(view, zones, d.MatrixProperties.Width)
			}

			chainBox.Add(withTopMargin(view.grids[view.activeGrid], 30))
			modalContent.Add(chainBox)

			// For chain devices add buttons to select active grid.
			if len(view.grids) > 1 {
				tileSelector := container.NewHBox()
				buttons := make([]*widget.Button, len(view.grids))
				for i := range view.grids {
					btn := widget.NewButton(fmt.Sprintf("%d", i+1), func() {
						view.activeGrid = i
						chainBox.Objects = []fyne.CanvasObject{
							withTopMargin(view.grids[view.activeGrid], 30),
						}
						chainBox.Refresh()
						for bi, b := range buttons {
							if view.activeGrid == bi {
								b.Disable()
							} else {
								b.Enable()
							}
						}
					})
					if view.activeGrid == i {
						btn.Disable()
					}
					buttons[i] = btn
					tileSelector.Add(btn)
				}
				modalContent.Add(withTopMargin(container.NewCenter(tileSelector), 10))
			}

			view.zoneSetPackets = func(i int, colors []packets.LightHsbk) []*protocol.Message {
				return messages.SetMatrixColorsFromSlice(i, 1, d.MatrixProperties.Width, colors, time.Millisecond)
			}

			applyBtn := widget.NewButton("Apply Zones",
				func() {
					// Prevent brightness-only updates triggered by the device last update.
					// The brightness in this case is set in the individual pixels, so it does not
					// correspond to the general brightness of the device.
					view.freezeUpdates()
					view.applyZones(ctrl, view.grids)
				},
			)

			modalContent.Add(withTopMargin(NewHItemWithSideLabel(applyBtn, newGridActionsButtons(view)), 10))

		case device.LightTypeMultiZone:
			grid := NewZoneGrid(view, d.MultizoneProperties.Zones, 8)
			view.grids = []*ZoneGrid{grid}
			view.zoneSetPackets = func(i int, colors []packets.LightHsbk) []*protocol.Message {
				return messages.SetMultizoneExtendedColors(0, colors, time.Millisecond)
			}
			applyBtn := widget.NewButton("Apply Zones", func() { view.applyZones(ctrl, view.grids) })

			modalContent.Add(withTopMargin(grid, 30))
			modalContent.Add(withTopMargin(NewHItemWithSideLabel(applyBtn, newGridActionsButtons(view)), 10))
		}

		if effects := availableEffectsForDevice(view.device); len(effects) > 0 {
			effectBtn := newEffectButton(ctrl, view, effects)
			modalContent.Add(withTopBottomMargin(effectBtn, 10))
		}

		d := dialog.NewCustom("", "Close", container.NewPadded(modalContent), parentWin)
		d.Resize(fyne.NewSize(350, 500))
		d.Show()
	})

	// Disable settings until zones have been loaded.
	switch view.device.LightType {
	case device.LightTypeMatrix, device.LightTypeMultiZone:
		view.enableSettingsIfReady()
	}

	view.content = container.NewPadded(container.NewVBox(statusLabel, brightnessSlider, NewHItemWithSideLabel(toggleBtn, view.settingsBtn)))
	return view
}

func (v *deviceView) applyZones(ctrl Controller, grids []*ZoneGrid) {
	for i, g := range grids {
		colors := make([]packets.LightHsbk, len(g.Cells))
		for j, c := range g.Cells {
			colors[j] = c.HSBK()
		}
		for _, m := range v.zoneSetPackets(i, colors) {
			ctrl.Send(v.device.Serial, m)
		}
	}
}

func newEffectButton(ctrl Controller, view *deviceView, effects []EffectDescriptor) *widget.Button {
	return widget.NewButtonWithIcon("", widget.NewIcon(theme.VisibilityIcon()).Resource, func() {
		var params any
		paramsBox := container.NewVBox()
		selectBtn := widget.NewSelect(effectLabels(effects), func(label string) {
			view.selectedEffect = new(EffectDescriptor)
			*view.selectedEffect = effectByLabel[label]
			params = view.selectedEffect.NewParams()
			paramsBox.Objects = nil
			if view.selectedEffect.ParamsUI != nil {
				if ui := view.selectedEffect.ParamsUI(view, params); ui != nil {
					paramsBox.Add(widget.NewLabelWithStyle("Parameters", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
					paramsBox.Add(ui)
				}
			}
			paramsBox.Refresh()
		})
		if view.selectedEffect != nil {
			selectBtn.SetSelected(view.selectedEffect.Label)
		} else if len(effects) > 0 {
			selectBtn.SetSelected(effects[0].Label)
		}

		sendFunc := func(msg *protocol.Message) error {
			return ctrl.Send(view.device.Serial, msg)
		}
		gridsBackup := view.grids
		playBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.MediaPlayIcon()).Resource, func() {
			view.mu.Lock()
			if view.selectedEffect != nil {
				// Make sure any running effect is stopped and its goroutine released.
				if view.selectedEffectStopper != nil {
					view.selectedEffectStopper()
				}
				view.selectedEffectStopper = view.selectedEffect.Play(sendFunc, view.device, params)
			}
			view.mu.Unlock()
		})

		stopBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.MediaStopIcon()).Resource, func() {
			view.mu.Lock()
			if view.selectedEffect != nil && view.selectedEffectStopper != nil {
				view.selectedEffectStopper()
				view.selectedEffectStopper = nil
				view.applyZones(ctrl, gridsBackup)
			}
			view.mu.Unlock()
		})
		content := container.NewVBox(
			container.NewCenter(widget.NewLabelWithStyle("Effect", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
			selectBtn,

			widget.NewSeparator(),

			paramsBox,

			withTopMargin(container.NewCenter(container.NewHBox(
				playBtn,
				stopBtn,
			)), 10),
		)

		effectModal := dialog.NewCustom("", "Close", container.NewPadded(content), view.parentWin)
		effectModal.Resize(fyne.NewSize(350, 400))
		effectModal.Show()

	})
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
	if v.settingsBtn != nil && v.settingsBtn.Disabled() {
		v.enableSettingsIfReady()
	}
	v.mu.RUnlock()
}

func (v *deviceView) refreshUI() {
	v.label.UpdateStatus(v.device.Label, deviceColorToRGBA(v.device))

	if v.brightness != nil {
		v.brightness.Set(v.device.Color.Brightness)
	}
}

func (v *deviceView) enableSettingsIfReady() {
	switch v.device.LightType {
	case device.LightTypeMatrix:
		if len(v.device.MatrixProperties.ChainZones) < 1 {
			v.settingsBtn.Disable()
			return
		}
		var isSet bool
		for i := range v.device.MatrixProperties.ChainZones[0] {
			if v.device.MatrixProperties.ChainZones[0][i] != emptyLightState {
				isSet = true
				break
			}
		}
		if !isSet {
			v.settingsBtn.Disable()
			return
		}
	case device.LightTypeMultiZone:
		if len(v.device.MultizoneProperties.Zones) == 0 {
			v.settingsBtn.Disable()
			return
		}
	}
	v.settingsBtn.Enable()
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

func (v *deviceView) getZoneSelectionColor() device.Color {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return *v.zoneSelectionColor
}

func (v *deviceView) setInternalColor(f func(*device.Color)) {
	v.mu.Lock()
	f(v.internalColor)
	f(v.zoneSelectionColor)
	v.mu.Unlock()
}

func (v *deviceView) updateLightCircle() {
	if v.lightCircle == nil {
		return
	}
	v.lightCircle.FillColor = colorToRGBA(v.getInternalColor())
	v.lightCircle.Refresh()
}

// updateIfSelectedCells sets the individual selected color for each zone
// according to the current zone selection color.
func (v *deviceView) updateIfSelectedCells() (updated bool) {
	if len(v.grids) == 0 || len(v.grids[0].Cells) == 0 {
		return
	}
	color := v.getZoneSelectionColor()
	for _, g := range v.grids {
		for _, c := range g.Cells {
			updatedColor := updateColorFromZoneSelectionColor(c.SelectedColor, &color)
			c.SelectedColor = &updatedColor
			if c.Selected {
				c.Refresh()
				updated = true
			}
		}
	}
	return
}

func updateColorFromZoneSelectionColor(c, sc *device.Color) device.Color {
	updated := device.Color{
		Hue:        c.Hue,
		Saturation: c.Saturation,
		Brightness: c.Brightness,
		Kelvin:     c.Kelvin,
	}
	if sc.Hue != unsetColorValue {
		updated.Hue = sc.Hue
	}
	if sc.Saturation != unsetColorValue {
		updated.Saturation = sc.Saturation
	}
	if sc.Brightness != unsetColorValue {
		updated.Brightness = sc.Brightness
	}
	if sc.Kelvin != 0 {
		updated.Kelvin = sc.Kelvin
	}
	return updated
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

	c.Brightness = mapBrightnessForUI(c.Brightness)

	if c.Saturation == 0 {
		r, g, b := c.KelvinToRGB()
		// Apply brightness scale after Kelvin conversion.
		scale := c.Brightness / 100.0
		return color.RGBA{R: uint8(float64(r) * scale), G: uint8(float64(g) * scale), B: uint8(float64(b) * scale), A: 255}
	}
	r, g, b := c.HSBToRGB()
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func mapBrightnessForUI(b float64) float64 {
	t := b / 100.0
	t = math.Pow(t, 1.0/gridGamma)

	return gridMinBrightness + t*(gridMaxBrightness-gridMinBrightness)
}

func deviceInfo(d *device.Device) string {
	return fmt.Sprintf("Serial: %s\n"+
		"IP: %s\n"+
		"ProductID: %d\n"+
		"FW Version: %s\n"+
		"Group: %s\n"+
		"Location: %s\n"+
		"RSSI: %s",
		d.Serial, d.Address.IP.String(), d.ProductID, d.FirmwareVersion, d.Group, d.Location, d.WifiRSSI,
	)
}

func newGridActionsButtons(view *deviceView) *fyne.Container {
	copyBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ContentCopyIcon()).Resource, func() {
		var colors []externalColor
		for _, g := range view.grids {
			for _, c := range g.Cells {
				colors = append(colors, colorToExternal(c.Color))
			}
		}
		b, _ := json.Marshal(colors)
		fyne.CurrentApp().Clipboard().SetContent(string(b))
	})

	confirmSelectedBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ConfirmIcon()).Resource, func() {
		for _, g := range view.grids {
			for _, c := range g.Cells {
				// Set prev state to the current state before applying the new color to selected cells.
				c.PrevState.Last = c.Color
				if c.Selected {
					c.Color = c.SelectedColor
				}
				c.ClearSelection()
			}
		}
	})

	clearSelectedBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.CancelIcon()).Resource, func() {
		for _, g := range view.grids {
			for _, c := range g.Cells {
				c.ClearSelection()
			}
		}
	})

	undoBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ContentUndoIcon()).Resource, func() {
		for _, g := range view.grids {
			for _, c := range g.Cells {
				c.ResetLast()
			}
		}
	})

	clearGridBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.DeleteIcon()).Resource, func() {
		for _, g := range view.grids {
			for _, c := range g.Cells {
				c.ResetInitial()
			}
		}
	})

	rotateGridBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.ViewRefreshIcon()).Resource, func() {
		switch view.device.LightType {
		case device.LightTypeMatrix:
			view.grids[view.activeGrid].Rotate(RotateClockwise)
		case device.LightTypeMultiZone:
			view.grids[view.activeGrid].Reverse()
		}
	})

	imageOpenBtn := widget.NewButtonWithIcon("", widget.NewIcon(theme.FolderOpenIcon()).Resource, func() {
		parseImage(view)
	})

	return newButtonGrid(copyBtn, confirmSelectedBtn, clearSelectedBtn, undoBtn, clearGridBtn, rotateGridBtn, imageOpenBtn)
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

func withTopBottomMargin(content fyne.CanvasObject, px float32) fyne.CanvasObject {
	pad := canvas.NewRectangle(color.Transparent)
	pad.SetMinSize(fyne.NewSize(1, px))
	return container.NewBorder(pad, pad, nil, nil, content)
}
