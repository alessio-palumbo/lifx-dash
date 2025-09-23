package main

import (
	"lifx-dash/dashboard"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/alessio-palumbo/lifxlan-go/pkg/controller"
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
)

func main() {
	a := app.New()
	a.Settings()
	w := a.NewWindow("LIFX Dash")
	w.Resize(fyne.NewSize(400, 600))

	// ctrl, err := controller.New(controller.WithHFStateRefreshPeriod(1000 * time.Second))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer ctrl.Close()
	ctrl := &controller.Controller{}

	d0, _ := device.SerialFromHex("0xd073d5000000")
	d1, _ := device.SerialFromHex("0xd073d5000001")
	// Perform discovery
	// time.Sleep(2 * time.Second)
	// devices := ctrl.GetDevices()
	devices := []device.Device{
		{Label: "Device0", Serial: d0},
		{Label: "Device1", Serial: d1},
	}

	list, deviceWidgets := dashboard.BuildDashboard(w, ctrl, devices)
	w.SetContent(list)

	// Background refresh loop
	go func() {
		// The ticker should not trigger too fast to avoid updating stale
		// state when best-effort toggle state is applied.
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fyne.Do(func() {
				latest := ctrl.GetDevices()
				for _, d := range latest {
					if view, ok := deviceWidgets[d.Serial]; ok {
						view.Update(d)
					}
				}
			})
		}
	}()

	w.ShowAndRun()
}
