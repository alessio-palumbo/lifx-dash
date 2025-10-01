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
	w.SetTitle(fmt.Sprintf("LIFX Dash %s (%s)", version.Version, version.Commit))
	w.Resize(fyne.NewSize(800, 600))

	ctrl, err := controller.New(controller.WithHFStateRefreshPeriod(2 * time.Second))
	if err != nil {
		log.Fatal(err)
	}
	defer ctrl.Close()

	dash := dashboard.NewDashboard(w, ctrl)
	dash.Run()

	w.ShowAndRun()
}
