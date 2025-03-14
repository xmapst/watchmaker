//go:build linux && (amd64 || arm64)

package main

import (
	"flag"
	"log"
	"strings"

	"github.com/xmapst/watchmaker"
)

var (
	pid           uint64
	fakeTime      string
	clockIdsSlice string
)

func main() {
	flag.Uint64Var(&pid, "pid", 0, "pid of target program")
	flag.StringVar(&fakeTime, "faketime", "", "fake time (incremental/absolute value)")
	flag.StringVar(&clockIdsSlice, "clockids", "", "clockids to modify, default is CLOCK_REALTIME")
	flag.Parse()

	if pid <= 0 {
		log.Fatalln("pid can't is zero")
	}
	if fakeTime == "" {
		log.Fatalln("faketime can't is empty")
	}
	if clockIdsSlice == "" {
		clockIdsSlice = "CLOCK_REALTIME"
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

	childPIDs, err := watchmaker.GetChildProcesses(pid)
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
