package dashboard

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
)

const minContrastBrightness = 40

type ZoneCell struct {
	widget.BaseWidget

	Index         int
	Selected      bool
	SelectedColor *device.Color
	OnTapped      func() device.Color

	rect   *canvas.Rectangle
	border *canvas.Rectangle
}

func NewZoneCell(onTap func() device.Color) *ZoneCell {
	z := &ZoneCell{SelectedColor: &device.Color{}, OnTapped: onTap}
	z.ExtendBaseWidget(z)
	return z
}

func (z *ZoneCell) Tapped(_ *fyne.PointEvent) {
	z.Selected = !z.Selected
	if z.OnTapped != nil {
		*z.SelectedColor = z.OnTapped()
	}
	z.Refresh()
}

func (z *ZoneCell) CreateRenderer() fyne.WidgetRenderer {
	z.rect = &canvas.Rectangle{FillColor: color.RGBA{A: 255}}
	z.border = &canvas.Rectangle{StrokeWidth: 0.5, StrokeColor: color.White}

	z.border.StrokeColor = color.Opaque

	objects := []fyne.CanvasObject{z.rect, z.border}
	return &zoneCellRenderer{cell: z, objects: objects}
}

type zoneCellRenderer struct {
	cell    *ZoneCell
	objects []fyne.CanvasObject
}

func (r *zoneCellRenderer) Layout(size fyne.Size) {
	r.cell.rect.Resize(size)
	r.cell.border.Resize(size)
}

func (r *zoneCellRenderer) MinSize() fyne.Size {
	return fyne.NewSize(24, 24) // nice small square
}

func (r *zoneCellRenderer) Refresh() {
	if r.cell.Selected {
		r.cell.rect.FillColor = colorToRGBA(colorWithAdjustedContrast(r.cell.SelectedColor))
	} else {
		r.cell.rect.FillColor = color.RGBA{A: 255}
	}
	r.cell.rect.Refresh()
}

func (r *zoneCellRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *zoneCellRenderer) Destroy()                     {}

// colorWithAdjustedContrast sets the minimum brightness of the displayed color
// to an acceptable contrast level.
func colorWithAdjustedContrast(c *device.Color) device.Color {
	color := *c
	color.Brightness = max(color.Brightness, minContrastBrightness)
	return color
}
