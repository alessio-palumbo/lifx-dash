package dashboard

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
	"github.com/alessio-palumbo/lifxprotocol-go/gen/protocol/packets"
)

type Rotation int

const (
	RotateClockwise Rotation = iota
	RotateCounterClockwise
)

type ZoneGrid struct {
	widget.BaseWidget

	Rows  int
	Cols  int
	Cells []*ZoneCell
	// Store is size N*N where N = max(Rows, Cols)
	Store []*ZoneCell
	// Extent is the side length of the square rotation space.
	// It is always max(Rows, Cols) and guarantees lossless rotation
	// for irregular grids.
	Extent        int
	HiddenIndexes map[int]bool

	parentWin fyne.Window

	grid    *fyne.Container
	overlay *canvas.Rectangle

	dragStart *fyne.Position
	dragEnd   *fyne.Position
}

func NewZoneGrid(view *deviceView, zones []packets.LightHsbk, gridWidth int) *ZoneGrid {
	rows := len(zones) / gridWidth
	cols := gridWidth
	extent := max(rows, cols)

	z := &ZoneGrid{
		Rows:      rows,
		Cols:      cols,
		Store:     make([]*ZoneCell, extent*extent),
		Extent:    extent,
		parentWin: view.parentWin,
		grid:      container.NewVBox(),
		overlay: &canvas.Rectangle{
			StrokeColor: color.RGBA{255, 255, 255, 255},
			StrokeWidth: 2,
		},
	}

	if r := CustomGridRules(view.device); r != nil {
		z.HiddenIndexes = r.HiddenIndexes
	}

	rowOff, colOff := z.storeOffset()

	for i, zone := range zones {
		color := device.NewColor(zone)
		cell := NewZoneCell(view.parentWin, &color)

		r := i / cols
		c := i % cols

		dst := z.storeIndex((rowOff + r), (colOff + c))
		z.Store[dst] = cell
	}

	z.projectToGrid()
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

func (z *ZoneGrid) Rotate(dir Rotation) {
	center := float64(z.Extent-1) / 2
	next := make([]*ZoneCell, z.Extent*z.Extent)

	for r := range z.Extent {
		for c := range z.Extent {
			cell := z.Store[z.storeIndex(r, c)]
			if cell == nil {
				continue
			}

			x := float64(c) - center
			y := float64(r) - center

			var nx, ny float64
			if dir == RotateClockwise {
				nx, ny = -y, x
			} else {
				nx, ny = y, -x
			}

			nc := int(math.Round(nx + center))
			nr := int(math.Round(ny + center))

			if nr >= 0 && nr < z.Extent && nc >= 0 && nc < z.Extent {
				next[z.storeIndex(nr, nc)] = cell
			}
		}
	}

	z.Store = next
	z.projectToGrid()
}

func (z *ZoneGrid) projectToGrid() {
	rowOff, colOff := z.storeOffset()

	z.Cells = make([]*ZoneCell, z.Rows*z.Cols)

	for r := 0; r < z.Rows; r++ {
		for c := 0; c < z.Cols; c++ {
			src := z.storeIndex((rowOff + r), (colOff + c))
			dst := r*z.Cols + c

			if cell := z.Store[src]; cell != nil {
				z.Cells[dst] = cell
			} else {
				z.Cells[dst] = NewZoneCell(z.parentWin, &device.Color{})
			}
		}
	}

	z.buildGrid()
	z.Refresh()
}

// buildGrid builds UI cells preserving exact positions while ignoring hidden cells.
func (z *ZoneGrid) buildGrid() {
	z.grid.Objects = nil

	var realIndex int
	for range z.Rows {
		rowCells := make([]fyne.CanvasObject, 0, z.Cols)

		for col := 0; col < z.Cols && realIndex < len(z.Cells); col++ {
			z.Cells[realIndex].SetActive()
			if z.HiddenIndexes[realIndex] {
				z.Cells[realIndex].SetInactive()
			}
			rowCells = append(rowCells, z.Cells[realIndex])
			realIndex++
		}

		rowGrid := container.NewGridWithColumns(z.Cols, rowCells...)
		z.grid.Add(rowGrid)
	}

	z.grid.Refresh()
}

// storeOffset returns the row and column offset required to center the
// visible grid (Rows x Cols) inside the square Store matrix (Extent x Extent).
//
// This offset must be applied consistently when projecting cells
// between Store and the visible grid to preserve symmetry during rotations.
func (z *ZoneGrid) storeOffset() (rowOff, colOff int) {
	return (z.Extent - z.Rows) / 2, (z.Extent - z.Cols) / 2
}

// storeIndex converts 2D Store coordinates (row, col) into the linear index
// used by the Store slice.
//
// Store is always indexed using Extent as its stride (not Rows or Cols),
// because Store represents the canonical square matrix used for rotation.
func (z *ZoneGrid) storeIndex(rowIdx, colIdx int) int {
	return rowIdx*z.Extent + colIdx
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
