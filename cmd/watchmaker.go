//go:build linux && (amd64 || arm64)

package main

import (
	"flag"
	"os"
	"strings"

	"github.com/mitchellh/go-ps"
	"github.com/xmapst/logx"

	"github.com/xmapst/watchmaker/internal/tasks"
	"github.com/xmapst/watchmaker/internal/time"
	"github.com/xmapst/watchmaker/internal/time/utils"
	"github.com/xmapst/watchmaker/internal/version"
)

var (
	pid           int
	fakeTime      string
	printVersion  bool
	clockIdsSlice string
)

func init() {
	flag.IntVar(&pid, "pid", 0, "pid of target program")
	flag.StringVar(&fakeTime, "faketime", "", "fake time (incremental/absolute value)")
	flag.StringVar(&clockIdsSlice, "clk_ids", "CLOCK_REALTIME", "all affected clock ids split with \",\"")
	flag.BoolVar(&printVersion, "version", false, "print version information and exit")
}

func main() {
	flag.Parse()

	version.PrintVersionInfo("Watchmaker")
	if printVersion {
		os.Exit(0)
	}

	if pid == 0 {
		logx.Fatalln("pid can't is zero")
	}
	offsetTime, err := time.CalculateOffset(fakeTime)
	if err != nil {
		logx.Fatalln(err)
	}
	clkIds := strings.Split(clockIdsSlice, ",")
	mask, err := utils.EncodeClkIds(clkIds)
	if err != nil {
		logx.Fatalln(err)
	}
	logx.Infoln("get clock ids mask", mask)

	s, err := time.GetSkew(time.NewConfig(0, offsetTime.Nanoseconds(), mask))
	if err != nil {
		logx.Fatalln(err)
	}

	logx.Infoln("modifying time, pid", pid, "fake time", offsetTime.String(), "mask", mask)
	err = s.Inject(tasks.SysPID(pid))
	if err != nil {
		logx.Fatalln(err)
	}

	for _, _childPid := range getChildPid(pid) {
		sf, err := s.Fork()
		if err != nil {
			logx.Errorln(err)
			continue
		}
		logx.Infoln("modifying time, ppid", pid, "pid", _childPid, "fake time", offsetTime.String(), "mask", mask)
		err = sf.Inject(tasks.SysPID(_childPid))
		if err != nil {
			logx.Errorln(err)
		}
	}
}

func getChildPid(pid int) []int {
	procs, err := ps.Processes()
	if err != nil {
		logx.Errorln(err)
		return nil
	}
	return childPid(pid, procs)
}

func childPid(pid int, procs []ps.Process) (res []int) {
	for _, p := range procs {
		if p.PPid() == pid {
			res = append(res, p.Pid())
			res = append(res, childPid(p.Pid(), procs)...)
		}
	}
	return
}
