package watchmaker

import "golang.org/x/sys/unix"

const __WALL = 0x40000000

// waitpid waits for the process with the specified pid to exit
// and returns the pid of the exited process; returns -1 if an error occurs.
func waitpid(pid int) int {
	var status unix.WaitStatus
	wpid, err := unix.Wait4(pid, &status, __WALL, nil)
	if err != nil {
		return -1
	}
	return wpid
}
