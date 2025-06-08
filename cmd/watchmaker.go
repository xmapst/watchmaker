//go:build linux && (amd64 || arm64)

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/busybox-org/watchmaker"
)

var (
	pid           uint64
	fakeTime      string
	clockIdsSlice string
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)
}

func main() {
	var clockIdsSliceDefault string
	if runtime.GOARCH == "arm64" {
		// on modern arm64 there is no __NR_time syscall;
		// glibc is using clock_gettime() wrapper [1] with CLOCK_REALTIME_COARSE clockid [2]
		//
		// [1] https://sourceware.org/git/?p=glibc.git;a=blob;f=time/time.c;h=d5dcb2e7ed83bc491ed026caf914caf4f1ae9202;hb=c804cd1c00adde061ca51711f63068c103e94eef
		// [2] https://sourceware.org/git/?p=glibc.git;a=blob;f=sysdeps/unix/sysv/linux/time-clockid.h;h=91543b69e47ce2828316ff0b3361ec435159690e;hb=c804cd1c00adde061ca51711f63068c103e94eef
		clockIdsSliceDefault = "CLOCK_REALTIME,CLOCK_REALTIME_COARSE"
	} else {
		clockIdsSliceDefault = "CLOCK_REALTIME"
	}

	flag.Uint64Var(&pid, "pid", 0, "pid of target program")
	flag.StringVar(&fakeTime, "faketime", "", "fake time (incremental/absolute value)")
	flag.StringVar(&clockIdsSlice, "clockids", "", "clockids to modify, default is "+clockIdsSliceDefault)
	flag.Parse()

	if pid <= 0 {
		log.Fatalln("pid can't is zero")
	}
	if fakeTime == "" {
		log.Fatalln("faketime can't is empty")
	}
	if clockIdsSlice == "" {
		clockIdsSlice = clockIdsSliceDefault
	}
	log.Println("pid:", pid, "faketime:", fakeTime, "clockids:", clockIdsSlice)

	offsetTime, err := watchmaker.CalculateOffset(fakeTime)
	if err != nil {
		log.Fatalln(err)
	}

	clkIds, err := watchmaker.EncodeClkIds(strings.Split(clockIdsSlice, ","))
	if err != nil {
		log.Fatalln(err)
	}

	skew, err := watchmaker.GetSkew(watchmaker.NewConfig(0, offsetTime.Nanoseconds(), clkIds))
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("modifying time, pid: %v", pid)
	err = skew.Inject(pid)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("modifying time success")

	childPIDs, err := getChildProcesses(pid)
	if err != nil {
		log.Fatalln(err)
	}
	if len(childPIDs) == 0 {
		return
	}
	log.Printf("modifying child time, pids: %v", childPIDs)
	for _, _childPid := range childPIDs {
		var skewFork *watchmaker.Skew
		skewFork, err = skew.Fork()
		if err != nil {
			log.Println(err)
			continue
		}
		err = skewFork.Inject(_childPid)
		if err != nil {
			log.Println(err)
		}
	}
	log.Println("modifying child time success")
}

const DefaultProcPrefix = "/proc"

// GetChildProcesses will return all child processes's pid. Include all generations.
// only return error when /proc/pid/tasks cannot be read
func getChildProcesses(ppid uint64) ([]uint64, error) {
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
	_pid, err := strconv.ParseUint(name, 10, 64)
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
	_, err = fmt.Fscanf(reader, "%d %s %s %d", &_pid, &comm, &state, &ppid)
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	pidMap[ppid] = append(pidMap[ppid], _pid)
}
