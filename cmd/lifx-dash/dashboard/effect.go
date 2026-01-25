package dashboard

import (
	"image/color"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
	"github.com/alessio-palumbo/lifxlan-go/pkg/matrix"
	"github.com/alessio-palumbo/lifxlan-go/pkg/protocol"
	"github.com/alessio-palumbo/lifxprotocol-go/gen/protocol/packets"
)

var defaultColor = color.RGBA{G: 128, B: 255, A: 255}

var allEffects = []EffectDescriptor{
	EffectMatrixWaterfall,
	EffectMatrixRockets,
	EffectMatrixWorm,
	EffectMatrixSnake,
	EffectMatrixConcentric,
}

var effectByLabel = map[string]EffectDescriptor{
	"Waterfall":         EffectMatrixWaterfall,
	"Rockets":           EffectMatrixRockets,
	"Worm":              EffectMatrixWorm,
	"Snake":             EffectMatrixSnake,
	"Concentric Frames": EffectMatrixConcentric,
}

const (
	directionInwards  = "inwards"
	directionOutwards = "outwards"
	directionInOut    = "in-out"
	directionOutIn    = "out-in"
)

var animationDirections = map[string]int{
	directionInwards:  int(matrix.AnimationDirectionInwards),
	directionOutwards: int(matrix.AnimationDirectionOutwards),
	directionInOut:    int(matrix.AnimationDirectionInOut),
	directionOutIn:    int(matrix.AnimationDirectionOutIn),
}

type SendFunc = func(msg *protocol.Message) error

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
	SendIntervalMs int64
	Brightness     float64
	Colors         []color.RGBA
	Size           int
	Direction      int
}

var EffectMatrixWaterfall = EffectDescriptor{
	ID:          "matrix_waterfall",
	Label:       "Waterfall",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixEffectParams{
			SendIntervalMs: 100,
			Brightness:     50,
			Colors:         newColorsParam(6, defaultColor),
		}
	},

	Play: func(sender SendFunc, d *device.Device, params any) *atomic.Bool {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) == 0 {
			return nil
		}
		return startMatrixEffect(sender, d, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.Waterfall(m, wrappedSender, p.SendIntervalMs, 0, 0, colors...)
		})
	},

	Stop: func(stop *atomic.Bool) {
		if stop != nil {
			stop.Store(true)
		}
	},

	ParamsUI: func(parentWindow fyne.Window, params any) fyne.CanvasObject {
		p := params.(*MatrixEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 50, 500, 50, float64(p.SendIntervalMs), func(v float64) error {
			p.SendIntervalMs = int64(v)
			return nil
		})
		brightnessSlider := NewSliderWithEntry("%.0f", 1, 100, 1, p.Brightness, func(v float64) error {
			p.Brightness = v
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
			LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
			LabelledSlider("Colors", modalLabelWidth, container.NewPadded(colorPalette(parentWindow, p.Colors))),
		)
	},
}

var EffectMatrixRockets = EffectDescriptor{
	ID:          "matrix_rocket",
	Label:       "Rockets",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixEffectParams{
			SendIntervalMs: 100,
			Brightness:     50,
			Colors:         newColorsParam(6, defaultColor),
		}
	},

	Play: func(sender SendFunc, d *device.Device, params any) *atomic.Bool {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) == 0 {
			return nil
		}
		return startMatrixEffect(sender, d, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.Rockets(m, wrappedSender, p.SendIntervalMs, 0, 0, colors...)
		})
	},

	Stop: func(stop *atomic.Bool) {
		if stop != nil {
			stop.Store(true)
		}
	},

	ParamsUI: func(parentWindow fyne.Window, params any) fyne.CanvasObject {
		p := params.(*MatrixEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 50, 500, 50, float64(p.SendIntervalMs), func(v float64) error {
			p.SendIntervalMs = int64(v)
			return nil
		})
		brightnessSlider := NewSliderWithEntry("%.0f", 1, 100, 1, p.Brightness, func(v float64) error {
			p.Brightness = v
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
			LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
			LabelledSlider("Colors", modalLabelWidth, container.NewPadded(colorPalette(parentWindow, p.Colors))),
		)
	},
}

var EffectMatrixWorm = EffectDescriptor{
	ID:          "matrix_worm",
	Label:       "Worm",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixEffectParams{
			SendIntervalMs: 100,
			Brightness:     50,
			Colors:         newColorsParam(1, defaultColor),
			Size:           4,
		}
	},

	Play: func(sender SendFunc, d *device.Device, params any) *atomic.Bool {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) == 0 {
			return nil
		}
		return startMatrixEffect(sender, d, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.Worm(m, wrappedSender, p.SendIntervalMs, 0, 0, p.Size, colors[0])
		})
	},

	Stop: func(stop *atomic.Bool) {
		if stop != nil {
			stop.Store(true)
		}
	},

	ParamsUI: func(parentWindow fyne.Window, params any) fyne.CanvasObject {
		p := params.(*MatrixEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 50, 500, 50, float64(p.SendIntervalMs), func(v float64) error {
			p.SendIntervalMs = int64(v)
			return nil
		})
		sizeSlider := NewSliderWithEntry("%.0f", 1, 10, 1, float64(p.Size), func(v float64) error {
			p.Size = int(v)
			return nil
		})
		brightnessSlider := NewSliderWithEntry("%.0f", 1, 100, 1, p.Brightness, func(v float64) error {
			p.Brightness = v
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
			LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
			LabelledSlider("Color", modalLabelWidth, container.NewPadded(colorPalette(parentWindow, p.Colors))),
			LabelledSlider("Size", modalLabelWidth, sizeSlider),
		)
	},
}

var EffectMatrixSnake = EffectDescriptor{
	ID:          "matrix_snake",
	Label:       "Snake",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixEffectParams{
			SendIntervalMs: 100,
			Brightness:     50,
			Colors:         newColorsParam(1, defaultColor),
			Size:           4,
		}
	},

	Play: func(sender SendFunc, d *device.Device, params any) *atomic.Bool {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) == 0 {
			return nil
		}
		return startMatrixEffect(sender, d, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.Snake(m, wrappedSender, p.SendIntervalMs, 0, 0, p.Size, colors[0])
		})
	},

	Stop: func(stop *atomic.Bool) {
		if stop != nil {
			stop.Store(true)
		}
	},

	ParamsUI: func(parentWindow fyne.Window, params any) fyne.CanvasObject {
		p := params.(*MatrixEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 50, 500, 50, float64(p.SendIntervalMs), func(v float64) error {
			p.SendIntervalMs = int64(v)
			return nil
		})
		sizeSlider := NewSliderWithEntry("%.0f", 1, 10, 1, float64(p.Size), func(v float64) error {
			p.Size = int(v)
			return nil
		})
		brightnessSlider := NewSliderWithEntry("%.0f", 1, 100, 1, p.Brightness, func(v float64) error {
			p.Brightness = v
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
			LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
			LabelledSlider("Color", modalLabelWidth, container.NewPadded(colorPalette(parentWindow, p.Colors))),
			LabelledSlider("Size", modalLabelWidth, sizeSlider),
		)
	},
}

var EffectMatrixConcentric = EffectDescriptor{
	ID:          "matrix_concentric",
	Label:       "Concentric Frames",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixEffectParams{
			SendIntervalMs: 200,
			Brightness:     50,
			Colors:         newColorsParam(1, defaultColor),
		}
	},

	Play: func(sender SendFunc, d *device.Device, params any) *atomic.Bool {
		p := params.(*MatrixEffectParams)
		var color *packets.LightHsbk
		if colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness); len(colors) > 0 {
			color = &colors[0]
		}
		return startMatrixEffect(sender, d, func(m *matrix.Matrix, wrappedSender SendFunc) {
			direction := matrix.ParseAnimationDirection(p.Direction)
			matrix.ConcentricFrames(m, wrappedSender, p.SendIntervalMs, 0, 0, direction, color)
		})
	},

	Stop: func(stop *atomic.Bool) {
		if stop != nil {
			stop.Store(true)
		}
	},

	ParamsUI: func(parentWindow fyne.Window, params any) fyne.CanvasObject {
		p := params.(*MatrixEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 50, 500, 50, float64(p.SendIntervalMs), func(v float64) error {
			p.SendIntervalMs = int64(v)
			return nil
		})
		brightnessSlider := NewSliderWithEntry("%.0f", 1, 100, 1, p.Brightness, func(v float64) error {
			p.Brightness = v
			return nil
		})
		directionSelector := selectFromLabels([]string{directionInwards, directionOutwards, directionInOut, directionOutIn}, directionInwards, func(selected string) {
			if v, ok := animationDirections[selected]; ok {
				p.Direction = v
			}
		})
		return container.NewVBox(
			LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
			LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
			LabelledSlider("Color", modalLabelWidth, container.NewPadded(colorPalette(parentWindow, p.Colors))),
			LabelledSlider("Direction", modalLabelWidth, container.NewPadded(directionSelector)),
		)
	},
}

func startMatrixEffect(send SendFunc, d *device.Device, f func(m *matrix.Matrix, wrappedSender SendFunc)) *atomic.Bool {
	m := matrix.New(int(d.MatrixProperties.Width), int(d.MatrixProperties.Height), int(d.MatrixProperties.ChainLength))
	sender, stopped := matrix.SendWithStop(send)
	go func() {
		f(m, sender)
		stopped.Store(true)
	}()
	return stopped
}

func selectedColorsToLightHSBK(cc []color.RGBA, brightnessOverride *float64) []packets.LightHsbk {
	colors := make([]packets.LightHsbk, 0, len(cc))
	for _, c := range cc {
		if c.A == 0 || (c.R+c.G+c.B == 0) {
			continue
		}
		rgbColor := rgbToColor(c.R, c.G, c.B)
		if brightnessOverride != nil {
			rgbColor.Brightness = *brightnessOverride
		}
		colors = append(colors, rgbColor.ToDeviceColor())
	}
	return colors
}

func selectFromLabels(labels []string, defaultLabel string, f func(label string)) *widget.Select {
	w := widget.NewSelect(labels, func(label string) {
		f(label)
	})
	if defaultLabel != "" {
		w.SetSelected(defaultLabel)
	}
	return w
}

func colorPalette(parentWindow fyne.Window, cc []color.RGBA) *fyne.Container {
	colorCircles := container.NewGridWrap(fyne.NewSize(25, 25))
	for i := range cc {
		circle := canvas.NewCircle(cc[i])
		circle.StrokeColor = color.White
		circle.StrokeWidth = 2
		colorCircles.Add(NewClickableCircle(circle, func() {
			picker := dialog.NewColorPicker("", "",
				func(c color.Color) {
					circle.FillColor = c
					r, g, b, a := c.RGBA()
					cc[i] = color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
				},
				parentWindow,
			)
			picker.Advanced = true
			picker.Show()
		}))
	}
	return colorCircles
}

func newColorsParam(n int, initial ...color.RGBA) []color.RGBA {
	colors := make([]color.RGBA, n)
	if len(initial) > 0 {
		for i := 0; i < min(n, len(initial)); i++ {
			colors[i] = initial[i]
		}
	}
	return colors
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
