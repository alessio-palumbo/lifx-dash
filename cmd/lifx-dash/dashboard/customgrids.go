package dashboard

import "github.com/alessio-palumbo/lifxlan-go/pkg/device"

var candleProducts = map[int]struct{}{
	57:  {},
	67:  {},
	68:  {},
	81:  {},
	96:  {},
	137: {},
	138: {},
	185: {},
	186: {},
	187: {},
	188: {},
	215: {},
	216: {},
	217: {},
	218: {},
}

var lunaProducts = map[int]struct{}{
	219: {},
	220: {},
}

var ceilingProducts = map[int]struct{}{
	// Ceiling 15"
	145: {},
	146: {},
	176: {},
	177: {},

	// Ceiling 13"
	265: {},
	266: {},
}

var ceilingCapsuleProducts = map[int]struct{}{
	201: {},
	202: {},
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
	if _, ok := lunaProducts[int(d.ProductID)]; ok {
		return &GridRules{
			HiddenIndexes: map[int]bool{
				0: true, 6: true,
				28: true, 34: true,
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
	if _, ok := ceilingCapsuleProducts[int(d.ProductID)]; ok {
		return &GridRules{
			HiddenIndexes: map[int]bool{
				0: true, 1: true, 14: true, 15: true,
				112: true, 113: true, 126: true, 127: true,
			},
		}
	}

	return nil
}
