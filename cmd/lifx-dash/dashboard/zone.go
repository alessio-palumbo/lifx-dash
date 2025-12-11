package dashboard

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
)

type ZoneCell struct {
	widget.BaseWidget

	Index         int
	Selected      bool
	Color         *device.Color
	SelectedColor *device.Color
	OnTapped      func() device.Color

	rect   *canvas.Rectangle
	border *canvas.Rectangle

	infoWin           *widget.PopUp
	updateWidgetColor func(*device.Color)
}

func NewZoneCell(parentWin fyne.Window, color *device.Color, onTap func() device.Color) *ZoneCell {
	infoWidget := widget.NewLabel(colorToLabel(color))
	z := &ZoneCell{
		Color:         color,
		SelectedColor: &device.Color{},
		OnTapped:      onTap,
		infoWin:       widget.NewPopUp(infoWidget, parentWin.Canvas()),
		updateWidgetColor: func(c *device.Color) {
			infoWidget.SetText(colorToLabel(c))
		},
	}
	z.ExtendBaseWidget(z)
	return z
}

func (z *ZoneCell) Tapped(_ *fyne.PointEvent) {
	z.Selected = !z.Selected
	if z.Selected {
		*z.SelectedColor = z.OnTapped()
	}
	z.Refresh()
}

func (z *ZoneCell) TappedSecondary(*fyne.PointEvent) {
	if z.Visible() {
		z.infoWin.ShowAtRelativePosition(fyne.NewPos(0, z.Size().Height+5), z)
	} else {
		z.infoWin.Hide()
	}
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
	return fyne.NewSize(24, 24)
}

func (r *zoneCellRenderer) Refresh() {
	if r.cell.Selected {
		r.cell.rect.FillColor = colorToRGBA(*r.cell.SelectedColor)
		r.cell.border.StrokeWidth = 3
		r.cell.updateWidgetColor(r.cell.SelectedColor)
	} else {
		r.cell.rect.FillColor = colorToRGBA(*r.cell.Color)
		r.cell.border.StrokeWidth = 0.5
		r.cell.updateWidgetColor(r.cell.Color)
	}
	r.cell.rect.Refresh()
}

func (r *zoneCellRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *zoneCellRenderer) Destroy()                     {}

func colorToLabel(c *device.Color) string {
	return fmt.Sprintf("H:%.0f,S:%.0f,B:%.0f,K:%d", c.Hue, c.Saturation, c.Brightness, c.Kelvin)
}
