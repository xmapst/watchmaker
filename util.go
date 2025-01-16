package watchmaker

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

const DefaultProcPrefix = "/proc"

// GetChildProcesses will return all child processes's pid. Include all generations.
// only return error when /proc/pid/tasks cannot be read
func GetChildProcesses(ppid uint64) ([]uint64, error) {
	procs, err := os.ReadDir(DefaultProcPrefix)
	if err != nil {
		return nil, fmt.Errorf("%T read /proc/pid/tasks , ppid : %d", err, ppid)
	}

	pidMap := make(map[uint64][]uint64) // Map of parent PID to child PIDs
	var mu sync.Mutex                   // Mutex for synchronizing map writes
	var wg sync.WaitGroup               // WaitGroup to manage goroutines

	for _, proc := range procs {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			processStat(&mu, name, pidMap)
		}(proc.Name())
	}

	wg.Wait()

	// Collect all child PIDs recursively starting from the given ppid.
	result := collectAllChildren(ppid, pidMap)
	return result, nil
}

func collectAllChildren(ppid uint64, pidMap map[uint64][]uint64) []uint64 {
	var result []uint64
	for _, child := range pidMap[ppid] {
		result = append(result, child)
		result = append(result, collectAllChildren(child, pidMap)...)
	}
	return result
}

// processStat parses a process's stat file and updates the pidMap with parent-child relationships.
func processStat(mu *sync.Mutex, name string, pidMap map[uint64][]uint64) {
	pid, err := strconv.ParseUint(name, 10, 64)
	if err != nil {
		return
	}

	statusPath := filepath.Join(DefaultProcPrefix, name, "stat")
	reader, err := os.Open(statusPath)
	if err != nil {
		return
	}
	defer reader.Close()

	var (
		ppid  uint64
		comm  string
		state string
	)
	// according to procfs's man page
	_, err = fmt.Fscanf(reader, "%d %s %s %d", &pid, &comm, &state, &ppid)
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	pidMap[ppid] = append(pidMap[ppid], pid)
}

// EncodeClkIds will convert array of clk ids into a mask
//func EncodeClkIds(clkIds []string) (uint64, error) {
//	mask := uint64(0)
//
//	for _, id := range clkIds {
//		// refer to `uapi/linux/time.h`
//		switch id {
//		case "CLOCK_REALTIME":
//			mask |= 1 << 0
//		case "CLOCK_MONOTONIC":
//			mask |= 1 << 1
//		case "CLOCK_PROCESS_CPUTIME_ID":
//			mask |= 1 << 2
//		case "CLOCK_THREAD_CPUTIME_ID":
//			mask |= 1 << 3
//		case "CLOCK_MONOTONIC_RAW":
//			mask |= 1 << 4
//		case "CLOCK_REALTIME_COARSE":
//			mask |= 1 << 5
//		case "CLOCK_MONOTONIC_COARSE":
//			mask |= 1 << 6
//		case "CLOCK_BOOTTIME":
//			mask |= 1 << 7
//		case "CLOCK_REALTIME_ALARM":
//			mask |= 1 << 8
//		case "CLOCK_BOOTTIME_ALARM":
//			mask |= 1 << 9
//		default:
//			return 0, fmt.Errorf("unknown clock id %s", id)
//		}
//	}
//
//	return mask, nil
//}
