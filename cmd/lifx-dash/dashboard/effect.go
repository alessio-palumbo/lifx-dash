package dashboard

import (
	"image/color"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
	"github.com/alessio-palumbo/lifxlan-go/pkg/matrix"
	"github.com/alessio-palumbo/lifxprotocol-go/gen/protocol/packets"
)

var allEffects = []EffectDescriptor{
	EffectMatrixWaterfall,
}

var effectByLabel = map[string]EffectDescriptor{
	"Waterfall": EffectMatrixWaterfall,
}

type SendFunc = matrix.SendFunc

type EffectKind int

const (
	EffectMatrix EffectKind = 1 << iota
	EffectStrip
)

type EffectDescriptor struct {
	ID          string
	Label       string
	SupportedOn EffectKind

	// Build default params
	NewParams func() any

	// Apply effect
	Play func(sender SendFunc, d *device.Device, params any) *atomic.Bool
	Stop func(stop *atomic.Bool)

	// UI
	ParamsUI func(parentWindow fyne.Window, params any) fyne.CanvasObject
}

type MatrixEffectParams struct {
	TransitionMs int64
	Colors       []color.RGBA
}

var EffectMatrixWaterfall = EffectDescriptor{
	ID:          "matrix_waterfall",
	Label:       "Waterfall",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		colors := make([]color.RGBA, 6)
		colors[0] = color.RGBA{R: 255, A: 255}
		return &MatrixEffectParams{
			TransitionMs: 100,
			Colors:       colors,
		}
	},

	Play: func(sender SendFunc, d *device.Device, params any) *atomic.Bool {
		p := params.(*MatrixEffectParams)
		colors := make([]packets.LightHsbk, 0, len(p.Colors))
		for _, c := range p.Colors {
			if c.A == 0 {
				continue
			}
			colors = append(colors, rgbToColor(c.R, c.G, c.B).ToDeviceColor())
		}
		if len(colors) == 0 {
			return nil
		}
		return StartMatrixWaterfallEffect(d.MatrixProperties, sender, p.TransitionMs, colors...)
	},

	Stop: func(stop *atomic.Bool) {
		if stop != nil {
			stop.Store(true)
		}
	},

	ParamsUI: func(parentWindow fyne.Window, params any) fyne.CanvasObject {
		p := params.(*MatrixEffectParams)

		transitionSlider := NewSliderWithEntry("%.0f", 50, 500, 50, float64(p.TransitionMs), func(v float64) error {
			p.TransitionMs = int64(v)
			return nil
		})

		colorCircles := container.NewGridWrap(fyne.NewSize(25, 25))
		for i := range p.Colors {
			circle := canvas.NewCircle(p.Colors[i])
			circle.StrokeColor = color.White
			circle.StrokeWidth = 2
			colorCircles.Add(NewClickableCircle(circle, func() {
				picker := dialog.NewColorPicker("", "",
					func(c color.Color) {
						circle.FillColor = c
						r, g, b, a := c.RGBA()
						p.Colors[i] = color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
					},
					parentWindow,
				)
				picker.Advanced = true
				picker.Show()
			}))
		}

		return container.NewVBox(
			LabelledSlider("Speed Ms", modalLabelWidth, transitionSlider),
			LabelledSlider("Colors", modalLabelWidth, container.NewPadded(colorCircles)),
		)
	},
}

func StartMatrixWaterfallEffect(mProps device.MatrixProperties, send SendFunc, transitionMs int64, colors ...packets.LightHsbk) *atomic.Bool {
	m := matrix.New(int(mProps.Width), int(mProps.Height), int(mProps.ChainLength))
	sender, stopped := matrix.SendWithStop(send)
	go func() {
		matrix.Waterfall(m, sender, transitionMs, 0, 0, colors...)
		stopped.Store(true)
	}()
	return stopped
}

func availableEffectsForDevice(d *device.Device) []EffectDescriptor {
	var kind EffectKind
	if d.LightType == device.LightTypeMatrix {
		kind |= EffectMatrix
	}
	if d.LightType == device.LightTypeMultiZone {
		kind |= EffectStrip
	}

	var out []EffectDescriptor
	for _, e := range allEffects {
		if e.SupportedOn&kind != 0 {
			out = append(out, e)
		}
	}
	return out
}

func effectLabels(effects []EffectDescriptor) []string {
	labels := make([]string, len(effects))
	for i, e := range effects {
		labels[i] = e.Label
	}
	return labels
}
