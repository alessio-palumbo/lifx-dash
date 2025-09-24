package main

import (
	"lifx-dash/dashboard"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/alessio-palumbo/lifxlan-go/pkg/controller"
)

func main() {
	a := app.New()
	a.Settings()
	w := a.NewWindow("LIFX Dash")
	w.Resize(fyne.NewSize(400, 600))

	ctrl, err := controller.New(controller.WithHFStateRefreshPeriod(1000 * time.Second))
	if err != nil {
		log.Fatal(err)
	}
	defer ctrl.Close()

	// Perform discovery
	time.Sleep(2 * time.Second)
	devices := ctrl.GetDevices()

	list, deviceWidgets := dashboard.BuildDashboard(a, w, ctrl, devices)
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
