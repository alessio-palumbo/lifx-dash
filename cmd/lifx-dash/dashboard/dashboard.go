package dashboard

import (
	"image/color"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/alessio-palumbo/lifxlan-go/pkg/controller"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
)

// Dashboard manages the UI and background refresh loop.
type Dashboard struct {
	app       fyne.App
	win       fyne.Window
	clipboard fyne.Clipboard

	ctrl          *controller.Controller
	devices       []device.Device
	deviceWidgets map[device.Serial]*deviceView
}

func NewDashboard(win fyne.Window, ctrl *controller.Controller) *Dashboard {
	return &Dashboard{
		win:       win,
		clipboard: fyne.CurrentApp().Clipboard(),

		ctrl:          ctrl,
		deviceWidgets: make(map[device.Serial]*deviceView),
	}
}

// Run starts the refresh loop and initializes the UI.
func (d *Dashboard) Run() {
	// Give the controller time to discover devices
	time.Sleep(2 * time.Second)
	d.refreshDevices()

	// Start background refresh loop
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			d.refreshDevices()
		}
	}()
}

// refreshDevices fetches the latest devices and updates the dashboard if needed.
func (d *Dashboard) refreshDevices() {
	latest := d.ctrl.GetDevices()

	// If the list has changed update the dashboard
	if d.devicesChanged(latest) {
		list, views := d.build(latest)

		fyne.Do(func() {
			d.win.SetContent(list)
		})

		d.devices = latest
		d.deviceWidgets = views
		// Dashboard has been refreshed, skip widgets update
		return
	}

	fyne.Do(func() {
		for _, dev := range latest {
			if view, ok := d.deviceWidgets[dev.Serial]; ok {
				// Update widgets only if device state has been refreshed
				if !dev.LastSeenAt.Equal(view.LastSeenAt()) {
					view.Update(dev)
				}
			}
		}
	})
}

// devicesChanged compares the current device list with the latest discovery result.
// It returns true if the number of devices differs or if any device serial does not match,
// indicating that the dashboard needs to be rebuilt.
func (d *Dashboard) devicesChanged(latest []device.Device) bool {
	if len(d.devices) != len(latest) {
		return true
	}
	for i := range d.devices {
		if d.devices[i].Serial != latest[i].Serial {
			return true
		}
	}
	return false
}

func (d *Dashboard) build(devices []device.Device) (fyne.CanvasObject, map[device.Serial]*deviceView) {
	groups, sortedGroups := groupDevices(devices)
	deviceWidgets := make(map[device.Serial]*deviceView)
	var sections []fyne.CanvasObject

	for _, groupName := range sortedGroups {
		var cards []fyne.CanvasObject
		for _, device := range groups[groupName] {
			view := newDeviceView(d.win, d.ctrl, &device)
			view.label.SetClipboard(d.clipboard)

			deviceWidgets[device.Serial] = view
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
