package dashboard

import (
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/controller"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
)

func BuildDashboard(a fyne.App, w fyne.Window, ctrl *controller.Controller, devices []device.Device) (fyne.CanvasObject, map[device.Serial]*deviceView) {
	groups, sortedGroups := groupDevices(devices)
	deviceWidgets := make(map[device.Serial]*deviceView)
	var sections []fyne.CanvasObject

	for _, groupName := range sortedGroups {
		var cards []fyne.CanvasObject
		for _, d := range groups[groupName] {
			view := newDeviceView(w, ctrl, &d)
			view.label.SetClipboard(a.Clipboard())

			deviceWidgets[d.Serial] = view
			cards = append(cards, view.content)
		}

		grid := container.NewGridWrap(fyne.NewSize(200, 150), cards...)
		header := widget.NewButton(groupName, func() {
			if grid.Visible() {
				grid.Hide()
			} else {
				grid.Show()
			}
			grid.Refresh()
		})

		sep := canvas.NewRectangle(color.RGBA{R: 100, G: 100, B: 100, A: 255})
		sep.SetMinSize(fyne.NewSize(0, 5))
		sections = append(sections, container.NewVBox(header, grid, sep))
	}

	return container.NewVScroll(container.NewVBox(sections...)), deviceWidgets
}

func groupDevices(devices []device.Device) (map[string][]device.Device, []string) {
	groups := make(map[string][]device.Device)
	var sortedGroups []string

	for _, d := range devices {
		group := d.Group
		if group == "" {
			group = "Ungrouped"
		}
		if _, ok := groups[group]; !ok {
			sortedGroups = append(sortedGroups, group)
		}
		groups[group] = append(groups[group], d)
	}

	slices.Sort(sortedGroups)
	return groups, sortedGroups
}
