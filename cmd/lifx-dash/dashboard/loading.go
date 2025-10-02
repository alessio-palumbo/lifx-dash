package dashboard

import (
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	loadingDotCount     = 3
	loadingDotSize      = 10
	loadingTickInterval = 200 * time.Millisecond
)

var (
	loadingDotActiveColor   = color.RGBA{R: 140, G: 26, B: 146, A: 100}
	loadingDotInactiveColor = color.Gray{Y: 180}
)

type LoadingDots struct {
	box  *fyne.Container
	dots []*loadingDot

	mu      sync.Mutex
	stopCh  chan struct{}
	running bool
}

func NewLoadingDots() *LoadingDots {
	ld := &LoadingDots{
		box:  container.NewHBox(),
		dots: make([]*loadingDot, loadingDotCount),
	}
	for i := range loadingDotCount {
		ld.dots[i] = newLoadingDot(loadingDotSize, loadingDotActiveColor)
		ld.dots[i].Resize(fyne.NewSize(loadingDotSize, loadingDotSize))
		ld.box.Add(ld.dots[i])
	}

	return ld
}

func (ld *LoadingDots) Run() {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	if ld.running {
		return
	}
	ld.stopCh = make(chan struct{})
	ld.running = true

	go func() {
		defer func() {
			ld.mu.Lock()
			ld.running = false
			ld.mu.Unlock()
		}()

		ticker := time.NewTicker(loadingTickInterval)
		defer ticker.Stop()
		var idx int

		for {
			select {
			case <-ld.stopCh:
				return
			case <-ticker.C:
				fyne.Do(func() {
					for i, dot := range ld.dots {
						dot.SetActive(i == idx)
					}
				})
				idx = (idx + 1) % len(ld.dots)
			}
		}
	}()
}

func (ld *LoadingDots) Stop() {
	ld.mu.Lock()
	defer ld.mu.Unlock()
	if !ld.running {
		return
	}
	close(ld.stopCh)
	ld.running = false
}

func (ld *LoadingDots) Object() fyne.CanvasObject {
	return ld.box
}

// loadingDot is a tiny custom widget that renders a fixed-size filled circle.
type loadingDot struct {
	widget.BaseWidget
	circle   *canvas.Circle
	diameter float32

	// state
	active bool
}

// newLoadingDot constructs a dot widget with the given diameter and initial color.
func newLoadingDot(diameter float32, col color.Color) *loadingDot {
	d := &loadingDot{
		circle:   canvas.NewCircle(col),
		diameter: diameter,
	}
	d.ExtendBaseWidget(d)
	return d
}

// CreateRenderer implements fyne.Widget
func (d *loadingDot) CreateRenderer() fyne.WidgetRenderer {
	// renderer holds the circle (we reuse the widget.circle pointer)
	r := &dotRenderer{dot: d, circle: d.circle}
	return r
}

// SetActive toggles the dot appearance. Must be called on the main thread (see animation).
func (d *loadingDot) SetActive(active bool) {
	if d.active == active {
		return
	}
	d.active = active
	d.Refresh() // schedules a redraw
}

// Min intrinsic size for the widget
func (d *loadingDot) MinSize() fyne.Size {
	return fyne.NewSize(d.diameter, d.diameter)
}

// dotRenderer renders the single circle
type dotRenderer struct {
	dot    *loadingDot
	circle *canvas.Circle
}

func (r *dotRenderer) Layout(size fyne.Size) {
	// give the circle the full size
	r.circle.Resize(size)
}

func (r *dotRenderer) MinSize() fyne.Size {
	return r.dot.MinSize()
}

func (r *dotRenderer) Refresh() {
	// pick color based on active flag
	if r.dot.active {
		r.circle.FillColor = loadingDotActiveColor
	} else {
		r.circle.FillColor = loadingDotInactiveColor
	}
	r.circle.Refresh()
}

func (r *dotRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.circle}
}

func (r *dotRenderer) Destroy() {}
