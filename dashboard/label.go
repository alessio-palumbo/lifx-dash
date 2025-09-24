package dashboard

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
)

// StatusLabel combines a label with a colored status circle and tooltip
type StatusLabel struct {
	widget.BaseWidget
	text        string
	circleColor color.Color
	circleSize  float32

	// UI components
	label   *widget.Label
	circle  *canvas.Circle
	content *fyne.Container

	// Info window
	parentWin  fyne.Window
	tooltipWin *widget.PopUp
	showInfo   bool

	// Clipboard
	serial    string
	clipboard fyne.Clipboard
	copiedWin *widget.PopUp
}

// NewStatusLabel creates a new status label widget
func NewStatusLabel(d device.Device, parentWin fyne.Window) *StatusLabel {
	s := &StatusLabel{
		text:        d.Label,
		circleColor: deviceColorToRGBA(&d),
		circleSize:  14,
		parentWin:   parentWin,
		serial:      d.Serial.String(),
	}

	s.buildUI(&d)
	s.ExtendBaseWidget(s)
	return s
}

// buildUI constructs the internal UI components
func (s *StatusLabel) buildUI(d *device.Device) {
	s.label = widget.NewLabelWithStyle(s.text, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	s.circle = canvas.NewCircle(s.circleColor)
	s.circle.Resize(fyne.NewSize(s.circleSize, s.circleSize))

	// Create a container for the circle to ensure it gets proper space
	circleContainer := container.NewWithoutLayout(s.circle)
	circleContainer.Resize(s.circle.Size())

	// Create container with horizontal layout and padding
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.Resize(fyne.NewSize(4, 1))

	// Simple horizontal layout with automatic spacing
	s.content = container.NewHBox(circleContainer, spacer, s.label)

	tooltipLabel := widget.NewLabel(deviceInfo(d))
	s.tooltipWin = widget.NewPopUp(tooltipLabel, s.parentWin.Canvas())
	copiedLabel := widget.NewLabel("Serial copied!")
	s.copiedWin = widget.NewPopUp(copiedLabel, s.parentWin.Canvas())
}

func (s *StatusLabel) UpdateStatus(color color.Color) {
	s.circleColor = color
	s.circle.FillColor = color
	s.circle.Refresh()
}

func (s *StatusLabel) SetText(text string) {
	s.text = text
	s.label.SetText(text)
}

func (s *StatusLabel) GetText() string {
	return s.text
}

func (s *StatusLabel) SetClipboard(clipboard fyne.Clipboard) {
	s.clipboard = clipboard
}

func (s *StatusLabel) Tapped(*fyne.PointEvent) {
	if s.showInfo {
		s.tooltipHide()
		return
	}

	// Position below the widget
	s.tooltipWin.ShowAtRelativePosition(fyne.NewPos(0, s.Size().Height+5), s)
	s.showInfo = true
}

func (s *StatusLabel) tooltipHide() {
	s.tooltipWin.Hide()
	s.showInfo = false
	s.Refresh()
}

func (s *StatusLabel) TappedSecondary(*fyne.PointEvent) {
	s.clipboard.SetContent(s.serial)
	// Position on top of the label on a slight left offset
	s.copiedWin.ShowAtRelativePosition(fyne.NewPos(5, -5), s)
	fyne.Do(func() {
		time.Sleep(400 * time.Millisecond)
		s.copiedWin.Hide()
	})
}

func (s *StatusLabel) MouseIn(*desktop.MouseEvent)    {}
func (s *StatusLabel) MouseOut()                      {}
func (s *StatusLabel) MouseMoved(*desktop.MouseEvent) {}

// CreateRenderer implements fyne.Widget with custom positioning
// This makes sure the circle is positioned correctly in the label.
func (s *StatusLabel) CreateRenderer() fyne.WidgetRenderer {
	return &statusLabelRenderer{
		statusLabel: s,
		content:     s.content,
	}
}

// statusLabelRenderer handles custom layout with proper circle positioning
type statusLabelRenderer struct {
	statusLabel *StatusLabel
	content     *fyne.Container
}

func (r *statusLabelRenderer) Layout(size fyne.Size) {
	r.content.Resize(size)

	// Manually position the circle to be vertically centered
	if len(r.content.Objects) > 0 {
		if circleContainer, ok := r.content.Objects[0].(*fyne.Container); ok {
			circleContainer.Resize(fyne.NewSize(r.statusLabel.circleSize, r.statusLabel.circleSize))
			if len(circleContainer.Objects) > 0 {
				r.statusLabel.circle.Resize(fyne.NewSize(r.statusLabel.circleSize, r.statusLabel.circleSize))
				// Center the circle vertically within the widget's height
				r.statusLabel.circle.Move(fyne.NewPos(0, (size.Height-r.statusLabel.circleSize)/2))
			}
		}
	}
}

func (r *statusLabelRenderer) MinSize() fyne.Size {
	labelSize := r.statusLabel.label.MinSize()
	// Calculate minimum size: circle width + separator + label width
	// Height is the maximum of circle height and label height
	separatorWidth := float32(4)
	minWidth := r.statusLabel.circleSize + separatorWidth + labelSize.Width
	minHeight := max(r.statusLabel.circleSize, labelSize.Height)
	return fyne.NewSize(minWidth, minHeight)
}

func (r *statusLabelRenderer) Refresh() {
	r.content.Refresh()
}

func (r *statusLabelRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.content}
}

func (r *statusLabelRenderer) Destroy() {}
