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

	// UI components
	label      *widget.Label
	circle     *canvas.Circle
	circleSize float32
	content    *fyne.Container

	// Info window
	parentWin fyne.Window
	infoWin   *widget.PopUp

	// Clipboard
	clipboard fyne.Clipboard
	copiedWin *widget.PopUp
	copyText  string
}

// NewStatusLabel creates a new status label widget
func NewStatusLabel(parentWin fyne.Window, d *device.Device) *StatusLabel {
	s := &StatusLabel{
		parentWin:  parentWin,
		circleSize: 14,
		copyText:   d.Serial.String(),
	}

	s.buildUI(d)
	s.ExtendBaseWidget(s)
	return s
}

// buildUI constructs the internal UI components
func (s *StatusLabel) buildUI(d *device.Device) {
	s.label = widget.NewLabelWithStyle(d.Label, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	s.circle = canvas.NewCircle(deviceColorToRGBA(d))
	s.circle.Resize(fyne.NewSize(s.circleSize, s.circleSize))

	// Create a container for the circle to ensure it gets proper space
	circleContainer := container.NewWithoutLayout(s.circle)
	circleContainer.Resize(s.circle.Size())

	// Create container with horizontal layout and padding
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.Resize(fyne.NewSize(4, 1))

	// Simple horizontal layout with automatic spacing
	s.content = container.NewHBox(circleContainer, spacer, s.label)

	infoWidget := widget.NewLabel(deviceInfo(d))
	s.infoWin = widget.NewPopUp(infoWidget, s.parentWin.Canvas())

	copyWidget := widget.NewLabel("Serial copied!")
	s.copiedWin = widget.NewPopUp(copyWidget, s.parentWin.Canvas())
}

func (s *StatusLabel) UpdateStatus(text string, color color.Color) {
	s.label.SetText(text)
	s.circle.FillColor = color
	s.circle.Refresh()
}

func (s *StatusLabel) SetClipboard(clipboard fyne.Clipboard) {
	s.clipboard = clipboard
}

// Tapped shows the info window.
// Note: If the window is already showing a tap will hide it behind the current
// window without the need to track its current state.
func (s *StatusLabel) Tapped(*fyne.PointEvent) {
	s.infoWin.ShowAtRelativePosition(fyne.NewPos(0, s.Size().Height+5), s)
}

// TappedSecondary copies the set text to the system clipboard
// and briefly shows a popup for user feedback.
func (s *StatusLabel) TappedSecondary(*fyne.PointEvent) {
	s.clipboard.SetContent(s.copyText)

	// Position on top of the label on a slight left offset
	s.copiedWin.ShowAtRelativePosition(fyne.NewPos(5, -5), s)
	fyne.Do(func() {
		time.Sleep(400 * time.Millisecond)
		s.copiedWin.Hide()
	})
}

func (s *StatusLabel) MouseIn(*desktop.MouseEvent) {}

// MoustOut hides the info windows with a slight delay.
func (s *StatusLabel) MouseOut() {
	fyne.Do(func() {
		time.Sleep(200 * time.Millisecond)
		s.infoWin.Hide()
	})
}

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
