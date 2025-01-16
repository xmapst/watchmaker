package watchmaker

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Entry is one line in /proc/pid/maps
type Entry struct {
	StartAddress uint64
	EndAddress   uint64
	Privilege    string
	PaddingSize  uint64
	Path         string
}

// ReadMaps parse /proc/[pid]/maps and return a list of entry
// The format of /proc/[pid]/maps can be found in `man proc`.
func ReadMaps(pid int) ([]Entry, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	var entries []Entry
	for _, line := range lines {
		sections := strings.Split(line, " ")
		if len(sections) < 3 {
			continue
		}

		var path string

		if len(sections) > 5 {
			path = sections[len(sections)-1]
		}

		addresses := strings.Split(sections[0], "-")
		startAddress, err := strconv.ParseUint(addresses[0], 16, 64)
		if err != nil {
			return nil, err
		}
		endAddresses, err := strconv.ParseUint(addresses[1], 16, 64)
		if err != nil {
			return nil, err
		}

		privilege := sections[1]

		paddingSize, err := strconv.ParseUint(sections[2], 16, 64)
		if err != nil {
			return nil, err
		}

		entries = append(entries, Entry{
			startAddress,
			endAddresses,
			privilege,
			paddingSize,
			path,
		})
	}

	return entries, nil
}
