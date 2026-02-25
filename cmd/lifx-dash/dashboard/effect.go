package dashboard

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
	"github.com/alessio-palumbo/lifxlan-go/pkg/matrix"
	"github.com/alessio-palumbo/lifxlan-go/pkg/messages"
	"github.com/alessio-palumbo/lifxlan-go/pkg/protocol"
	"github.com/alessio-palumbo/lifxprotocol-go/gen/protocol/packets"
)

const (
	// effectOffCooldownDuration allow a small buffer between sending an effect off
	// message and the next one.
	effectOffCooldownDuration = 100 * time.Millisecond
)

var defaultColor = color.RGBA{G: 128, B: 255, A: 255}

var allEffects = []EffectDescriptor{
	EffectMatrixConcentric,
	EffectMatrixWaterfall,
	EffectMatrixRockets,
	EffectMatrixWorm,
	EffectMatrixSnake,

	EffectBufferedAnimation,

	FWEffectMatrixFlame,
	FWEffectMatrixMorph,
	FWEffectMatrixClouds,
	FWEffectMatrixSunrise,
	FWEffectMatrixSunset,

	FWEffectMultizoneMove,
}

var effectByLabel = map[string]EffectDescriptor{
	"Concentric Frames": EffectMatrixConcentric,
	"Waterfall":         EffectMatrixWaterfall,
	"Rockets":           EffectMatrixRockets,
	"Worm":              EffectMatrixWorm,
	"Snake":             EffectMatrixSnake,
	"Animation":         EffectBufferedAnimation,
	"Flame":             FWEffectMatrixFlame,
	"Morph":             FWEffectMatrixMorph,
	"Clouds":            FWEffectMatrixClouds,
	"Sunrise":           FWEffectMatrixSunrise,
	"Sunset":            FWEffectMatrixSunset,

	"Move": FWEffectMultizoneMove,
}

const (
	chainModeSequential = "sequential"
	chainModeSynced     = "synced"
)

var chainModes = map[string]matrix.ChainMode{
	chainModeSequential: matrix.ChainModeSequential,
	chainModeSynced:     matrix.ChainModeSynced,
}

const (
	directionInwards  = "inwards"
	directionOutwards = "outwards"
	directionInOut    = "in-out"
	directionOutIn    = "out-in"
)

var animationDirections = map[string]matrix.AnimationDirection{
	directionInwards:  matrix.AnimationDirectionInwards,
	directionOutwards: matrix.AnimationDirectionOutwards,
	directionInOut:    matrix.AnimationDirectionInOut,
	directionOutIn:    matrix.AnimationDirectionOutIn,
}

const (
	moveDirectionForward  = "forward"
	moveDirectionBackward = "backward"
)

var moveDirections = map[string]bool{
	moveDirectionForward:  true,
	moveDirectionBackward: false,
}

type EffectKind int

const (
	EffectMatrix EffectKind = 1 << iota
	EffectMultizone
)

type EffectDescriptor struct {
	ID            string
	Label         string
	SupportedOn   EffectKind
	IsSupportedFW func(fwVersion string) bool

	// Build default params
	NewParams func() any

	// Apply effect
	Play func(view *deviceView, params any) (stopFunc func())

	// UI
	ParamsUI func(view *deviceView, params any) fyne.CanvasObject
}

type MatrixEffectParams struct {
	SendIntervalMs int64
	ChainMode      matrix.ChainMode
	Brightness     float64
	Colors         []color.RGBA
	Size           int
	Direction      matrix.AnimationDirection
}

var EffectMatrixConcentric = EffectDescriptor{
	ID:          "matrix_concentric",
	Label:       "Concentric Frames",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixEffectParams{
			SendIntervalMs: 200,
			Brightness:     50,
			Colors:         newColorsParam(6, defaultColor),
		}
	},

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		return startMatrixEffect(view, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.ConcentricFrames(m, wrappedSender, p.SendIntervalMs, 0, p.ChainMode, p.Direction, colors...)
		})
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
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
			addChainModeIfSupported(view, p,
				LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
				LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
				LabelledSlider("Color", modalLabelWidth, colorPalette(view.parentWin, p.Colors)),
				LabelledSlider("Direction", modalLabelWidth, container.NewStack(directionSelector)),
			)...,
		)
	},
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

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) == 0 {
			return nil
		}
		return startMatrixEffect(view, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.Waterfall(m, wrappedSender, p.SendIntervalMs, 0, p.ChainMode, colors...)
		})
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
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
			addChainModeIfSupported(view, p,
				LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
				LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
				LabelledSlider("Colors", modalLabelWidth, colorPalette(view.parentWin, p.Colors)),
			)...,
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

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) == 0 {
			return nil
		}
		return startMatrixEffect(view, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.Rockets(m, wrappedSender, p.SendIntervalMs, 0, p.ChainMode, colors...)
		})
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
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
			addChainModeIfSupported(view, p,
				LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
				LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
				LabelledSlider("Colors", modalLabelWidth, colorPalette(view.parentWin, p.Colors)),
			)...,
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

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) == 0 {
			return nil
		}
		return startMatrixEffect(view, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.Worm(m, wrappedSender, p.SendIntervalMs, 0, p.ChainMode, p.Size, colors[0])
		})
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
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
			addChainModeIfSupported(view, p,
				LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
				LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
				LabelledSlider("Color", modalLabelWidth, colorPalette(view.parentWin, p.Colors)),
				LabelledSlider("Size", modalLabelWidth, sizeSlider),
			)...,
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

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) == 0 {
			return nil
		}
		return startMatrixEffect(view, func(m *matrix.Matrix, wrappedSender SendFunc) {
			matrix.Snake(m, wrappedSender, p.SendIntervalMs, 0, p.ChainMode, p.Size, colors[0])
		})
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
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
			addChainModeIfSupported(view, p,
				LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
				LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
				LabelledSlider("Color", modalLabelWidth, colorPalette(view.parentWin, p.Colors)),
				LabelledSlider("Size", modalLabelWidth, sizeSlider),
			)...,
		)
	},
}

// Firmware Effects

type MatrixFWEffectParams struct {
	SpeedM     int64
	SpeedS     int64
	SpeedMs    int64
	Brightness float64
	Colors     []color.RGBA
	Frames     [][]packets.LightHsbk

	// FW Params
	MinSaturation uint32
	SoftOff       bool
	MoveDirection bool
}

var FWEffectMatrixFlame = EffectDescriptor{
	ID:          "fw_matrix_flame",
	Label:       "Flame",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixFWEffectParams{
			SpeedS: 3,
		}
	},

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixFWEffectParams)
		return startMatrixFWEffect(view.sendMsg, messages.SetMatrixFlameEffect(time.Duration(p.SpeedS)*time.Second))
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
		p := params.(*MatrixFWEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 1, 25, 1, float64(p.SpeedS), func(v float64) error {
			p.SpeedS = int64(v)
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed S", modalLabelWidth, intervalSlider),
		)
	},
}

var FWEffectMatrixMorph = EffectDescriptor{
	ID:          "fw_matrix_morph",
	Label:       "Morph",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixFWEffectParams{
			SpeedS:     3,
			Colors:     newColorsParam(6, defaultColor),
			Brightness: 100,
		}
	},

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixFWEffectParams)
		colors := selectedColorsToLightHSBK(p.Colors, &p.Brightness)
		if len(colors) > 16 {
			colors = colors[:16]
		}
		return startMatrixFWEffect(view.sendMsg, messages.SetMatrixMorphEffect(time.Duration(p.SpeedS)*time.Second, colors...))
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
		p := params.(*MatrixFWEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 1, 25, 1, float64(p.SpeedS), func(v float64) error {
			p.SpeedS = int64(v)
			return nil
		})
		brightnessSlider := NewSliderWithEntry("%.0f", 1, 100, 1, p.Brightness, func(v float64) error {
			p.Brightness = v
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed S", modalLabelWidth, intervalSlider),
			LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
			LabelledSlider("Colors", modalLabelWidth, colorPalette(view.parentWin, p.Colors)),
		)
	},
}

var FWEffectMatrixClouds = EffectDescriptor{
	ID:            "fw_matrix_clouds",
	Label:         "Clouds",
	SupportedOn:   EffectMatrix,
	IsSupportedFW: func(fwVersion string) bool { return requiresMinVersion(fwVersion, 4, 8) },

	NewParams: func() any {
		return &MatrixFWEffectParams{
			SpeedS:        100,
			MinSaturation: 50,
		}
	},

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixFWEffectParams)
		return startMatrixFWEffect(view.sendMsg, messages.SetMatrixCloudsEffect(time.Duration(p.SpeedS)*time.Second, &p.MinSaturation))
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
		p := params.(*MatrixFWEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 1, 100, 1, float64(p.SpeedS), func(v float64) error {
			p.SpeedS = int64(v)
			return nil
		})
		minSaturationSlider := NewSliderWithEntry("%.0f", 1, 255, 1, float64(p.MinSaturation), func(v float64) error {
			p.MinSaturation = uint32(v)
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed S", modalLabelWidth, intervalSlider),
			LabelledSlider("Min Saturation", modalLabelWidth, minSaturationSlider),
		)
	},
}

var FWEffectMatrixSunrise = EffectDescriptor{
	ID:            "fw_matrix_sunrise",
	Label:         "Sunrise",
	SupportedOn:   EffectMatrix,
	IsSupportedFW: func(fwVersion string) bool { return requiresMinVersion(fwVersion, 4, 8) },

	NewParams: func() any {
		return &MatrixFWEffectParams{
			SpeedM: 1,
		}
	},

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixFWEffectParams)
		speed := time.Duration(p.SpeedM) * time.Minute
		return startMatrixFWEffect(view.sendMsg, messages.SetMatrixSunriseEffect(&speed))
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
		p := params.(*MatrixFWEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 1, 100, 1, float64(p.SpeedM), func(v float64) error {
			p.SpeedM = int64(v)
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed M", modalLabelWidth, intervalSlider),
		)
	},
}

var FWEffectMatrixSunset = EffectDescriptor{
	ID:            "fw_matrix_sunset",
	Label:         "Sunset",
	SupportedOn:   EffectMatrix,
	IsSupportedFW: func(fwVersion string) bool { return requiresMinVersion(fwVersion, 4, 8) },

	NewParams: func() any {
		return &MatrixFWEffectParams{
			SpeedM: 1,
		}
	},

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixFWEffectParams)
		speed := time.Duration(p.SpeedM) * time.Minute
		return startMatrixFWEffect(view.sendMsg, messages.SetMatrixSunsetEffect(&speed, true))
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
		p := params.(*MatrixFWEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 1, 100, 1, float64(p.SpeedM), func(v float64) error {
			p.SpeedM = int64(v)
			return nil
		})
		return container.NewVBox(
			LabelledSlider("Speed M", modalLabelWidth, intervalSlider),
		)
	},
}

var FWEffectMultizoneMove = EffectDescriptor{
	ID:          "fw_multizone_move",
	Label:       "Move",
	SupportedOn: EffectMultizone,

	NewParams: func() any {
		return &MatrixFWEffectParams{
			SpeedS: 20,
		}
	},

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixFWEffectParams)
		speed := time.Duration(p.SpeedS) * time.Second
		return startMZFWEffect(view.sendMsg, messages.SetMultizoneMoveEffect(speed, p.MoveDirection))
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
		p := params.(*MatrixFWEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 1, 60, 1, float64(p.SpeedS), func(v float64) error {
			p.SpeedS = int64(v)
			return nil
		})
		moveDirection := selectFromLabels([]string{moveDirectionForward, moveDirectionBackward}, moveDirectionForward, func(selected string) {
			if v, ok := moveDirections[selected]; ok {
				p.MoveDirection = v
			}
		})
		return container.NewVBox(
			LabelledSlider("Speed S", modalLabelWidth, intervalSlider),
			LabelledSlider("Direction", modalLabelWidth, container.NewStack(moveDirection)),
		)
	},
}

var EffectBufferedAnimation = EffectDescriptor{
	ID:          "buffered_animation",
	Label:       "Animation",
	SupportedOn: EffectMatrix,

	NewParams: func() any {
		return &MatrixFWEffectParams{
			SpeedMs:    200,
			Brightness: 50,
			// TODO make this dynamic once supported_frame_buffers field is exposed in the protocol.
			Frames: make([][]packets.LightHsbk, 5),
		}
	},

	Play: func(view *deviceView, params any) func() {
		p := params.(*MatrixFWEffectParams)
		var frames [][]packets.LightHsbk
		for _, f := range p.Frames {
			if f != nil {
				frames = append(frames, rotateMatrixForOrientation(view.activeGrid, view.device.MatrixProperties, f))
			}
		}
		msgs, nextFrame := messages.SetMatrixFrameAnimation(view.activeGrid, 1, view.device.MatrixProperties.Width, frames, p.Brightness, 0)
		for _, msg := range msgs {
			view.sendMsg(msg)
		}

		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(time.Duration(p.SpeedMs) * time.Millisecond)
			for {
				select {
				case <-ticker.C:
					view.sendMsg(nextFrame())
				case <-done:
					return
				}
			}
		}()

		return func() {
			close(done)
		}
	},

	ParamsUI: func(view *deviceView, params any) fyne.CanvasObject {
		p := params.(*MatrixFWEffectParams)
		intervalSlider := NewSliderWithEntry("%.0f", 50, 5000, 50, float64(p.SpeedMs), func(v float64) error {
			p.SpeedMs = int64(v)
			return nil
		})
		brightnessSlider := NewSliderWithEntry("%.0f", 1, 100, 1, p.Brightness, func(v float64) error {
			p.Brightness = v
			return nil
		})

		vbox := container.NewVBox(
			LabelledSlider("Speed Ms", modalLabelWidth, intervalSlider),
			LabelledSlider("Brightness", modalLabelWidth, brightnessSlider),
		)

		frameBox := container.NewVBox()
		frameList := NewDynamicList[*FrameItem](frameBox, len(p.Frames))

		addFrameBtn := widget.NewButton("Add Frame", nil)
		addFrameBtn.OnTapped = func() {
			frame := newFrame(frameList, addFrameBtn, view, p.Frames)
			if frameList.Add(frame) {
				addFrameBtn.Disable()
			}
		}
		frameList.OnChange = func(items []*FrameItem) {
			for i, f := range items {
				f.Label.SetText(fmt.Sprintf("Frame %d/%d", i+1, len(p.Frames)))
			}

			if frameList.IsFull() {
				addFrameBtn.SetText("Max frames reached")
				addFrameBtn.Disable()
				return
			}

			addFrameBtn.SetText("Add Frame")
			if hasEmptyFrame(p.Frames, len(items)) {
				addFrameBtn.Disable()
				return
			}
			addFrameBtn.Enable()
		}

		return container.NewVBox(
			vbox,
			frameBox,
			container.NewBorder(nil, nil, nil, nil, addFrameBtn),
		)
	},
}

type FrameItem struct {
	Container *fyne.Container
	Label     *widget.Label
}

func (f *FrameItem) CanvasObject() fyne.CanvasObject {
	return f.Container
}

func hasEmptyFrame(frames [][]packets.LightHsbk, count int) bool {
	for i := range count {
		if frames[i] == nil {
			return true
		}
	}
	return false
}

func newFrame(list *DynamicList[*FrameItem], addFrameBtn *widget.Button, view *deviceView, frames [][]packets.LightHsbk) *FrameItem {
	thumbnail := NewThumbnail(nil, canvas.ImageFillStretch, 80, 30)
	obj := new(FrameItem)

	frameBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		index := list.IndexOf(obj)
		if index < 0 {
			return
		}

		ParseImage(view, func(grid []device.Color, img image.Image) {
			frame := make([]packets.LightHsbk, len(grid))
			for j, c := range grid {
				frame[j] = c.ToDeviceColor()
			}

			frames[index] = frame
			thumbnail.SetImage(img)

			// allow adding another frame once this one has data
			if !list.IsFull() {
				addFrameBtn.Enable()
			}
		})
	})

	clearBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		index := list.IndexOf(obj)
		if index < 0 {
			return
		}

		// shift slice data
		copy(frames[index:], frames[index+1:])
		frames[len(frames)-1] = nil
		list.Remove(obj)
	})

	frameContainer := container.NewBorder(nil, nil, container.NewStack(frameBtn, thumbnail.Image), clearBtn, nil)
	obj.Label, obj.Container = LabelledSliderWithLabel("", modalLabelWidth, frameContainer)
	return obj
}

func startMatrixEffect(view *deviceView, f func(m *matrix.Matrix, wrappedSender SendFunc)) (stopFunc func()) {
	props := view.device.MatrixProperties
	m := matrix.New(int(props.Width), int(props.Height), int(props.ChainLength))
	sender, stopped := matrix.SendWithStop(view.sendMsg)
	go func() {
		f(m, sender)
		stopped.Store(true)
	}()

	return func() {
		if stopped != nil {
			stopped.Store(true)
		}
	}
}

func startMatrixFWEffect(send SendFunc, msg *protocol.Message) (stopFunc func()) {
	send(msg)
	return func() {
		send(messages.SetMatrixEffectOff())
		time.Sleep(effectOffCooldownDuration)
	}
}

func startMZFWEffect(send SendFunc, msg *protocol.Message) (stopFunc func()) {
	send(msg)
	return func() {
		send(messages.SetMultizoneEffectOff())
		time.Sleep(effectOffCooldownDuration)
	}
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
	return container.NewPadded(colorCircles)
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
		kind |= EffectMultizone
	}

	var out []EffectDescriptor
	for _, e := range allEffects {
		if e.SupportedOn&kind != 0 {
			if e.IsSupportedFW == nil || e.IsSupportedFW(d.FirmwareVersion) {
				out = append(out, e)
			}
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

func addChainModeIfSupported(view *deviceView, p *MatrixEffectParams, objects ...fyne.CanvasObject) []fyne.CanvasObject {
	if view.device.LightType == device.LightTypeMatrix && view.device.MatrixProperties.ChainLength > 1 {
		chainModeSelector := selectFromLabels([]string{chainModeSynced, chainModeSequential}, chainModeSynced, func(selected string) {
			if v, ok := chainModes[selected]; ok {
				p.ChainMode = v
			}
		})
		objects = append(objects, LabelledSlider("ChainMode", modalLabelWidth, container.NewStack(chainModeSelector)))

	}
	return objects
}

func requiresMinVersion(fwVersion string, maj, min int) bool {
	parts := strings.Split(fwVersion, ".")
	if len(parts) != 2 {
		log.Println("Unexpected fwVersion", fwVersion)
		return false
	}

	majV, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Println("Error parsing major version:", err)
		return false
	}
	if majV < maj {
		return false
	}

	minV, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Println("Error parsing minor version:", err)
		return false
	}
	return minV >= min
}
