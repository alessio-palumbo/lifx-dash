package dashboard

import (
	"image"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
	"github.com/disintegration/imaging"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

func parseImage(view *deviceView) {
	d := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, view.parentWin)
				return
			}
			if reader == nil {
				// User cancelled
				return
			}
			defer reader.Close()

			img, _, err := image.Decode(reader)
			if err != nil {
				dialog.ShowError(err, view.parentWin)
				return
			}

			var grid []device.Color
			switch view.device.LightType {
			case device.LightTypeMatrix:
				grid = fitToGridPixels(img, view.device.MatrixProperties.Width, view.device.MatrixProperties.Height)
			case device.LightTypeMultiZone:
				grid = fillZones(img, len(view.device.MultizoneProperties.Zones))
			default:
				return
			}

			for _, g := range view.grids {
				for i, c := range g.Cells {
					c.resetColor(&grid[i])
				}
			}
		},
		view.parentWin,
	)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".jpg", ".jpeg", ".png", ".gif", ".webp"}))
	d.Show()
}

// fitToGridPixels resizes an image to fit a matrix of the given size, preserving
// proportions and placing it at the center of the matrix, returning a []device.Color.
func fitToGridPixels(img image.Image, gridW, gridH int) []device.Color {
	fitted := imaging.Fit(img, gridW, gridH, imaging.Lanczos)
	canvas := imaging.New(gridW, gridH, color.NRGBA{0, 0, 0, 255})
	out := imaging.PasteCenter(canvas, fitted)

	return zonesForImage(out, gridW, gridH)
}

// fillZones parses an image and applies an ideal grid based on nZones and the imag aspect ratio,
// preserving spatial structure and cropping or padding the image if needed. It then returns a
// flattened []device.Color with direction flipped per row.
func fillZones(img image.Image, nZones int) []device.Color {
	src := img.Bounds()
	aspect := float64(src.Dx()) / float64(src.Dy())

	gridW, gridH := gridForZones(nZones, aspect)

	filled := imaging.Fill(img, gridW, gridH, imaging.Center, imaging.Lanczos)

	grid := make([]device.Color, 0, gridW*gridH)
	for y := range gridH {
		for x := range gridW {
			grid = append(grid, nrgbaToColor(filled.NRGBAAt(x, y)))
		}
	}

	strip := snakeFlatten(grid, gridW, gridH)

	// Trim or pad to exact zone count
	if len(strip) > nZones {
		return strip[:nZones]
	}
	return strip
}

func zonesForImage(img *image.NRGBA, gridW, gridH int) []device.Color {
	grid := make([]device.Color, 0, gridW*gridH)
	for y := range gridH {
		for x := range gridW {
			grid = append(grid, nrgbaToColor(img.NRGBAAt(x, y)))
		}
	}
	return grid
}

func gridForZones(zones int, aspect float64) (w, h int) {
	h = max(1, int(math.Sqrt(float64(zones)/aspect)))
	w = max(1, zones/h)
	return
}

func snakeFlatten(grid []device.Color, w, h int) []device.Color {
	out := make([]device.Color, 0, w*h)

	for y := range h {
		if y%2 == 0 {
			for x := range w {
				out = append(out, grid[y*w+x])
			}
		} else {
			for x := w - 1; x >= 0; x-- {
				out = append(out, grid[y*w+x])
			}
		}
	}
	return out
}

func nrgbaToColor(c color.NRGBA) device.Color {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	// Brightness: 0–100
	brightness := max * 100.0
	// brightness = math.Max(1, brightness)

	// Saturation: 0–100
	var saturation float64
	if max != 0 {
		saturation = (delta / max) * 100.0
	}

	// Hue: 0–360
	var hue float64
	if delta == 0 {
		hue = 0
	} else if max == r {
		hue = math.Mod((g-b)/delta, 6)
		hue *= 60
	} else if max == g {
		hue = ((b-r)/delta + 2) * 60
	} else {
		hue = ((r-g)/delta + 4) * 60
	}

	if hue < 0 {
		hue += 360
	}

	return device.Color{
		Hue:        hue,
		Saturation: saturation,
		Brightness: brightness,
		Kelvin:     6500, // sensible default for grayscale
	}
}
