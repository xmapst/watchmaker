package watchmaker

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func CalculateOffset(offsetStr string) (time.Duration, error) {
	if offsetStr == "" || offsetStr == "0" || offsetStr == "null" {
		return 0, nil
	}
	// try parsing into a time string
	if t, err := ParseDateAny(offsetStr); err == nil {
		if t.UTC().Nanosecond() > time.Now().UTC().Nanosecond() {
			return time.Now().Sub(t), nil
		}
		return t.Sub(time.Now()), nil
	}

	// try parsing into seconds, minutes, hours, days, years
	offsetStr = strings.ToLower(offsetStr)
	unit := offsetStr[len(offsetStr)-1]
	value, err := strconv.Atoi(offsetStr)
	if err != nil {
		value, err = strconv.Atoi(offsetStr[:len(offsetStr)-1])
		if err != nil {
			return 0, fmt.Errorf("unable to parse offset：%v", err)
		}
	}
	var res time.Duration
	switch unit {
	case 's':
		res = time.Duration(value) * time.Second
	case 'm':
		res = time.Duration(value) * time.Minute
	case 'h':
		res = time.Duration(value) * time.Hour
	case 'd':
		res = time.Duration(value) * 24 * time.Hour
	case 'y':
		res = time.Duration(value) * 365 * 24 * time.Hour
	default:
		// processed by seconds by default
		res = time.Duration(value) * time.Second
	}
	return res, nil
}
