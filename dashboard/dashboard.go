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

func BuildDashboard(ctrl *controller.Controller, devices []device.Device) (fyne.CanvasObject, map[device.Serial]*deviceView) {
	groups := groupDevices(devices)

	deviceWidgets := make(map[device.Serial]*deviceView) // map by device ID
	var sections []fyne.CanvasObject

	var sortedGroups []string
	for k := range groups {
		sortedGroups = append(sortedGroups, k)
	}
	slices.Sort(sortedGroups)

	for _, groupName := range sortedGroups {
		devs := groups[groupName]
		// section title
		// title := widget.NewLabelWithStyle(groupName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		// title.SizeName = theme.SizeNameSubHeadingText
		title := canvas.NewText(groupName, color.Color(color.RGBA{255, 0, 0, 255}))
		title.Alignment = fyne.TextAlignLeading
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.TextSize = 16

		// grid of device widgets
		var cards []fyne.CanvasObject
		for _, d := range devs {
			view := newDeviceView(ctrl, d)
			deviceWidgets[d.Serial] = view
			cards = append(cards, view.content) //container.NewStack(view.Card))
		}
		grid := container.NewGridWithColumns(6, cards...) // 2 columns

		// group VBox
		sep := canvas.NewRectangle(color.RGBA{R: 100, G: 100, B: 100, A: 255})
		sep.SetMinSize(fyne.NewSize(0, 10)) // height of 2 pixels, width expands in container
		header := widget.NewButton(groupName, func() {
			if grid.Visible() {
				grid.Hide()
			} else {
				grid.Show()
			}
			grid.Refresh()
		})
		section := container.NewVBox(header, grid, sep)
		sections = append(sections, section)
	}

	return container.NewVScroll(container.NewVBox(sections...)), deviceWidgets
}

func groupDevices(devices []device.Device) map[string][]device.Device {
	groups := make(map[string][]device.Device)
	for _, d := range devices {
		group := d.Group
		if group == "" {
			group = "Ungrouped"
		}
		groups[group] = append(groups[group], d)
	}
	return groups
}
