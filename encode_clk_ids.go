package watchmaker

import (
	"fmt"
)

// EncodeClkIds will convert array of clk ids into a mask
func EncodeClkIds(clkIds []string) (uint64, error) {
	mask := uint64(0)

	for _, id := range clkIds {
		// refer to `uapi/linux/time.h`
		switch id {
		case "CLOCK_REALTIME":
			mask |= 1 << 0
		case "CLOCK_MONOTONIC":
			mask |= 1 << 1
		case "CLOCK_PROCESS_CPUTIME_ID":
			mask |= 1 << 2
		case "CLOCK_THREAD_CPUTIME_ID":
			mask |= 1 << 3
		case "CLOCK_MONOTONIC_RAW":
			mask |= 1 << 4
		case "CLOCK_REALTIME_COARSE":
			mask |= 1 << 5
		case "CLOCK_MONOTONIC_COARSE":
			mask |= 1 << 6
		case "CLOCK_BOOTTIME":
			mask |= 1 << 7
		case "CLOCK_REALTIME_ALARM":
			mask |= 1 << 8
		case "CLOCK_BOOTTIME_ALARM":
			mask |= 1 << 9
		default:
			return 0, fmt.Errorf("unknown clock id %s", id)
		}
	}

	return mask, nil
}
