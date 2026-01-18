package dashboard

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ZoneGrid struct {
	widget.BaseWidget

	Cells []*ZoneCell
	Rows  int
	Cols  int

	grid    *fyne.Container
	overlay *canvas.Rectangle

	dragStart *fyne.Position
	dragEnd   *fyne.Position
}

func NewZoneGrid(rows int, cols int, cells []*ZoneCell, hiddenIndexes map[int]bool) *ZoneGrid {
	z := &ZoneGrid{
		Rows:  rows,
		Cols:  cols,
		Cells: cells,
		overlay: &canvas.Rectangle{
			StrokeColor: color.RGBA{255, 255, 255, 255},
			StrokeWidth: 2,
		},
	}

	// Build UI cells preserving exact positions while ignoring hidden cells.
	uiCells := make([]fyne.CanvasObject, 0, len(cells))
	var realIndex int
	for range rows {
		rowCells := make([]fyne.CanvasObject, 0, cols)

		for col := 0; col < cols && realIndex < len(cells); col++ {
			if hiddenIndexes[realIndex] {
				cells[realIndex].SetInactive()
			}
			rowCells = append(rowCells, cells[realIndex])
			realIndex++
		}

		rowGrid := container.NewGridWithColumns(cols, rowCells...)
		uiCells = append(uiCells, rowGrid)
	}

	z.grid = container.NewVBox(uiCells...)
	z.ExtendBaseWidget(z)
	return z
}

func (z *ZoneGrid) CreateRenderer() fyne.WidgetRenderer {
	objects := []fyne.CanvasObject{z.grid, z.overlay}
	return &zoneGridRenderer{
		z:    z,
		objs: objects,
	}
}

func (z *ZoneGrid) Tapped(_ *fyne.PointEvent) {
	// No action here – individual ZoneCell handles taps.
}

func (z *ZoneGrid) Dragged(ev *fyne.DragEvent) {
	if z.dragStart == nil {
		p := ev.Position
		z.dragStart = &p
	}
	p := ev.Position
	z.dragEnd = &p
	z.Refresh()
}

func (z *ZoneGrid) DragEnd() {
	if z.dragStart != nil && z.dragEnd != nil {
		z.applyDrag()
	}
	z.dragStart = nil
	z.dragEnd = nil
	z.Refresh()
}

func (z *ZoneGrid) applyDrag() {
	start := *z.dragStart
	end := *z.dragEnd

	minX := min(start.X, end.X)
	maxX := max(start.X, end.X)
	minY := min(start.Y, end.Y)
	maxY := max(start.Y, end.Y)

	cellW := z.Size().Width / float32(z.Cols)
	cellH := z.Size().Height / float32(z.Rows)

	for r := range z.Rows {
		for c := range z.Cols {
			x1 := float32(c) * cellW
			y1 := float32(r) * cellH
			x2 := x1 + cellW
			y2 := y1 + cellH

			if intersects(minX, minY, maxX, maxY, x1, y1, x2, y2) {
				idx := r*z.Cols + c
				cell := z.Cells[idx]
				cell.SetSelected(true)
			}
		}
	}
}

type zoneGridRenderer struct {
	z    *ZoneGrid
	objs []fyne.CanvasObject
}

func (r *zoneGridRenderer) Layout(size fyne.Size) {
	r.z.grid.Resize(size)
	r.z.grid.Move(fyne.NewPos(0, 0))

	if r.z.dragStart != nil && r.z.dragEnd != nil {
		x1 := min(r.z.dragStart.X, r.z.dragEnd.X)
		y1 := min(r.z.dragStart.Y, r.z.dragEnd.Y)
		x2 := max(r.z.dragStart.X, r.z.dragEnd.X)
		y2 := max(r.z.dragStart.Y, r.z.dragEnd.Y)
		r.z.overlay.Move(fyne.NewPos(x1, y1))
		r.z.overlay.Resize(fyne.NewSize(x2-x1, y2-y1))
		r.z.overlay.Show()
	} else {
		r.z.overlay.Hide()
	}
	r.z.overlay.Refresh()
}

func (r *zoneGridRenderer) MinSize() fyne.Size {
	return r.z.grid.MinSize()
}
func (r *zoneGridRenderer) Refresh() {
	r.Layout(r.z.Size())
	// Refresh overlay and children
	for _, o := range r.objs {
		o.Refresh()
	}
	canvas.Refresh(r.z)
}
func (r *zoneGridRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *zoneGridRenderer) Destroy()                     {}

func intersects(ax1, ay1, ax2, ay2, bx1, by1, bx2, by2 float32) bool {
	return !(bx2 < ax1 || bx1 > ax2 || by2 < ay1 || by1 > ay2)
}
