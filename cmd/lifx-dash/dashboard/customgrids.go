package dashboard

import "github.com/alessio-palumbo/lifxlan-go/pkg/device"

var candleProducts = map[int]struct{}{
	57:  {},
	68:  {},
	137: {},
	138: {},
	185: {},
	186: {},
	215: {},
	216: {},
	217: {},
	218: {},
}

var ceilingProducts = map[int]struct{}{
	176: {},
	177: {},
	197: {},
	198: {},
	199: {},
	200: {},
}

var ceiling13Products = map[int]struct{}{
	201: {},
	202: {},
}

var lunaProducts = map[int]struct{}{
	219: {},
	220: {},
}

type GridRules struct {
	HiddenIndexes map[int]bool
}

func CustomGridRules(d *device.Device) *GridRules {
	if _, ok := candleProducts[int(d.ProductID)]; ok {
		return &GridRules{
			HiddenIndexes: map[int]bool{
				2: true, 3: true, 4: true,
			},
		}
	}
	if _, ok := ceilingProducts[int(d.ProductID)]; ok {
		return &GridRules{
			HiddenIndexes: map[int]bool{
				0: true, 1: true, 6: true, 7: true,
				56: true, 57: true, 62: true, 63: true,
			},
		}
	}
	if _, ok := ceiling13Products[int(d.ProductID)]; ok {
		return &GridRules{
			HiddenIndexes: map[int]bool{
				0: true, 1: true, 14: true, 15: true,
				112: true, 113: true, 126: true, 127: true,
			},
		}
	}
	if _, ok := lunaProducts[int(d.ProductID)]; ok {
		return &GridRules{
			HiddenIndexes: map[int]bool{
				0: true, 6: true,
				28: true, 34: true,
			},
		}
	}

	return nil
}
