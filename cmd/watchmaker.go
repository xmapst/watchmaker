//go:build linux && (amd64 || arm64)

package main

import (
	"flag"
	"log"

	"github.com/xmapst/watchmaker"
)

var (
	pid      uint64
	fakeTime string
)

func init() {
	flag.Uint64Var(&pid, "pid", 0, "pid of target program")
	flag.StringVar(&fakeTime, "faketime", "", "fake time (incremental/absolute value)")
}

func main() {
	flag.Parse()

	if pid <= 0 {
		log.Fatalln("pid can't is zero")
	}
	offsetTime, err := watchmaker.CalculateOffset(fakeTime)
	if err != nil {
		log.Fatalln(err)
	}

	skew, err := watchmaker.GetSkew(watchmaker.NewConfig(0, offsetTime.Nanoseconds(), 1))
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
