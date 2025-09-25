package main

import (
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/alessio-palumbo/lifx-dash/cmd/lifx-dash/dashboard"
	"github.com/alessio-palumbo/lifx-dash/internal/version"
	"github.com/alessio-palumbo/lifxlan-go/pkg/controller"
)

func main() {
	a := app.New()
	a.Settings()
	w := a.NewWindow("LIFX Dash")
	w.SetTitle(fmt.Sprintf("LIFX Dash v%s (%s)", version.Version, version.Commit))
	w.Resize(fyne.NewSize(800, 600))

	ctrl, err := controller.New(controller.WithHFStateRefreshPeriod(2 * time.Second))
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
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			fyne.Do(func() {
				latest := ctrl.GetDevices()
				for _, d := range latest {
					if view, ok := deviceWidgets[d.Serial]; ok {
						// Only update device if the device has changed.
						if !d.LastSeenAt.Equal(view.LastSeenAt()) {
							view.Update(d)
						}
					}
				}
			})
		}
	}()

	w.ShowAndRun()
}
