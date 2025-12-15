package dashboard

import (
	"github.com/alessio-palumbo/lifxlan-go/pkg/device"
)

// externalColor is an alias of device.Color but with json tags
// Brightness and Saturation values are represented as float between 0 and 1.
type externalColor struct {
	Hue        float64 `json:"hue"`
	Saturation float64 `json:"saturation"`
	Brightness float64 `json:"brightness"`
	Kelvin     uint16  `json:"kelvin"`
}

func colorToExternal(c *device.Color) externalColor {
	e := externalColor(*c)
	e.Saturation = e.Saturation / 100
	e.Brightness = e.Brightness / 100
	return e
}
