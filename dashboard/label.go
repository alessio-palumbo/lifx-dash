package dashboard

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
)

// StatusLabel is a custom widget that combines a label with a colored status circle
type StatusLabel struct {
	widget.BaseWidget
	label      *widget.Label
	circle     *canvas.Circle
	circleSize float32
	container  *fyne.Container
}

// NewStatusLabel creates a new label with a status circle
func NewStatusLabel(d device.Device, circleColor color.Color) *StatusLabel {
	label := widget.NewLabelWithStyle(deviceLabel(d), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	s := &StatusLabel{
		label:      label,
		circleSize: 14,
	}

	// Create a circle with specified color
	s.circle = canvas.NewCircle(deviceColorToRGBA(d))
	s.circle.Resize(fyne.NewSize(s.circleSize, s.circleSize))

	// Create a container for the circle to ensure it gets proper space
	circleContainer := container.NewWithoutLayout(s.circle)
	circleContainer.Resize(fyne.NewSize(s.circleSize, s.circleSize))

	// Create container with horizontal layout and padding
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.Resize(fyne.NewSize(4, 1))

	s.container = container.NewHBox(
		circleContainer,
		spacer,
		s.label,
	)

	s.ExtendBaseWidget(s)
	return s
}

// CreateRenderer implements fyne.Widget interface
func (s *StatusLabel) CreateRenderer() fyne.WidgetRenderer {
	return &statusLabelRenderer{
		statusLabel: s,
		container:   s.container,
	}
}

// MinSize returns the minimum size for this widget
func (s *StatusLabel) MinSize() fyne.Size {
	labelSize := s.label.MinSize()

	// Calculate minimum size: circle width + separator + label width
	// Height is the maximum of circle height and label height
	separatorWidth := float32(4) // Approximate separator width
	minWidth := s.circleSize + separatorWidth + labelSize.Width
	minHeight := labelSize.Height
	if s.circleSize > minHeight {
		minHeight = s.circleSize
	}

	return fyne.NewSize(minWidth, minHeight)
}

// UpdateStatus changes the circle color
func (s *StatusLabel) UpdateStatus(circleColor color.Color) {
	s.circle.FillColor = circleColor
	s.circle.Refresh()
}

// SetText updates the label text
func (s *StatusLabel) SetText(text string) {
	s.label.SetText(text)
	s.Refresh()
}

// GetText returns the current label text
func (s *StatusLabel) GetText() string {
	return s.label.Text
}

// SetCircleSize allows changing the circle size and updates layout
func (s *StatusLabel) SetCircleSize(size float32) {
	s.circleSize = size
	s.circle.Resize(fyne.NewSize(size, size))
	s.Refresh()
}

// statusLabelRenderer implements fyne.WidgetRenderer
type statusLabelRenderer struct {
	statusLabel *StatusLabel
	container   *fyne.Container
}

func (r *statusLabelRenderer) Layout(size fyne.Size) {
	r.container.Resize(size)
	// Ensure the circle maintains its size
	if len(r.container.Objects) > 0 {
		if circleContainer, ok := r.container.Objects[0].(*fyne.Container); ok {
			circleContainer.Resize(fyne.NewSize(r.statusLabel.circleSize, r.statusLabel.circleSize))
			if len(circleContainer.Objects) > 0 {
				r.statusLabel.circle.Resize(fyne.NewSize(r.statusLabel.circleSize, r.statusLabel.circleSize))
				r.statusLabel.circle.Move(fyne.NewPos(0, (size.Height-r.statusLabel.circleSize)/2))
			}
		}
	}
}

func (r *statusLabelRenderer) MinSize() fyne.Size {
	return r.statusLabel.MinSize()
}

func (r *statusLabelRenderer) Refresh() {
	r.container.Refresh()
}

func (r *statusLabelRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.container}
}

func (r *statusLabelRenderer) Destroy() {}
