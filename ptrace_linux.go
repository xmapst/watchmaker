package watchmaker

import (
	"bytes"
	"debug/elf"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const waitPidErrorMessage = "waitpid ret value: %d"

// If it's on 64-bit platform, `^uintptr(0)` will get a 64-bit number full of one.
// After shifting right for 63-bit, only 1 will be left. Than we got 8 here.
// If it's on 32-bit platform, After shifting nothing will be left. Than we got 4 here.
const ptrSize = 4 << (^uintptr(0) >> 63)

var threadRetryLimit = 10

// TracedProgram is a program traced by ptrace
type TracedProgram struct {
	pid     int
	tids    []int
	Entries []Entry

	backupRegs *unix.PtraceRegs
	backupCode []byte
}

// Pid return the pid of traced program
func (p *TracedProgram) Pid() int {
	return p.pid
}

func waitPid(pid int) error {
	status := unix.WaitStatus(0)
	wpid, err := unix.Wait4(pid, &status, 0, nil)
	if err != nil {
		return err
	}

	if wpid == pid {
		return nil
	}

	return fmt.Errorf(waitPidErrorMessage, wpid)
}

// Trace ptrace all threads of a process
func Trace(pid int) (*TracedProgram, error) {
	traceSuccess := false

	tidMap := make(map[int]bool)
	var attachedTids []int

	// 定义统一清理逻辑：如果 traceSuccess 未被置为 true，则对所有已 attach 的线程进行 detach
	defer func() {
		if !traceSuccess {
			for _, tid := range attachedTids {
				if err := unix.PtraceDetach(tid); err != nil && !strings.Contains(err.Error(), "no such process") {
					log.Println("detach failed", "tid", tid, "error", err)
				}
			}
		}
	}()

	// 循环遍历线程组，直到线程数稳定不再增加
	for {
		threads, err := os.ReadDir(fmt.Sprintf("/proc/%d/task", pid))
		if err != nil {
			return nil, err
		}

		subset := true
		tids := make(map[int]bool)
		for _, thread := range threads {
			var tid64 int64
			tid64, err = strconv.ParseInt(thread.Name(), 10, 32)
			if err != nil {
				return nil, err
			}
			tid := int(tid64)

			if tidMap[tid] {
				tids[tid] = true
				continue
			}
			subset = false

			err = unix.PtraceSeize(tid)
			if err != nil {
				return nil, err
			}

			err = unix.PtraceInterrupt(tid)
			if err != nil {
				return nil, err
			}
			// 成功 attach 后，记录 tid 用于后续统一 detach
			attachedTids = append(attachedTids, tid)

			if err = waitPid(tid); err != nil {
				return nil, err
			}

			//log.Println("attach successfully, process task id", tid)
			tids[tid] = true
			tidMap[tid] = true
		}

		if subset {
			tidMap = tids
			break
		}
	}

	// 将 tidMap 中的 key 转换为 slice
	var tidsList []int
	for tid := range tidMap {
		tidsList = append(tidsList, tid)
	}

	slices.Sort(tidsList)

	entries, err := ReadMaps(pid)
	if err != nil {
		return nil, err
	}

	program := &TracedProgram{
		pid:        pid,
		tids:       tidsList,
		Entries:    entries,
		backupRegs: &unix.PtraceRegs{},
		backupCode: make([]byte, unixInstrSize),
	}

	traceSuccess = true

	return program, nil
}

// Detach detaches from all threads of the processes
func (p *TracedProgram) Detach() error {
	for _, tid := range p.tids {
		//log.Println("detaching, process task id", tid)
		err := unix.PtraceDetach(tid)

		if err != nil {
			if !strings.Contains(err.Error(), "no such process") {
				return err
			}
		}
	}
	//log.Println("Successfully detach and rerun process, pid", p.pid)
	return nil
}

// Protect will backup regs and rip into fields
func (p *TracedProgram) Protect() error {
	err := getRegs(p.pid, p.backupRegs)
	if err != nil {
		return err
	}

	_, err = unix.PtracePeekData(p.pid, getIp(p.backupRegs), p.backupCode)
	if err != nil {
		return err
	}

	return nil
}

// Restore will restore regs and rip from fields
func (p *TracedProgram) Restore() error {
	err := setRegs(p.pid, p.backupRegs)
	if err != nil {
		return err
	}

	_, err = unix.PtracePokeData(p.pid, getIp(p.backupRegs), p.backupCode)
	if err != nil {
		return err
	}

	return nil
}

// Wait waits until the process stops
func (p *TracedProgram) Wait() error {
	return waitPid(p.pid)
}

// Step moves one step forward
func (p *TracedProgram) Step() error {
	err := unix.PtraceSingleStep(p.pid)
	if err != nil {
		return err
	}

	return p.Wait()
}

// Mmap runs mmap syscall
func (p *TracedProgram) Mmap(length uint64, fd uint64) (uint64, error) {
	return p.Syscall(unix.SYS_MMAP, 0, length, unix.PROT_READ|unix.PROT_WRITE|unix.PROT_EXEC, unix.MAP_ANON|unix.MAP_PRIVATE, fd, 0)
}

// ReadSlice reads from addr and return a slice
func (p *TracedProgram) ReadSlice(addr uint64, size uint64) (*[]byte, error) {
	buffer := make([]byte, size)

	// constructs a local iovec (a description of a memory buffer)
	localIov := []unix.Iovec{
		{
			Base: &buffer[0],
			Len:  size,
		},
	}

	// construct remote iovec (target process memory address and length)
	remoteIov := []unix.RemoteIovec{
		{
			Base: uintptr(addr),
			Len:  int(size),
		},
	}

	// calling the ProcessVMReadv system call
	_, err := unix.ProcessVMReadv(p.pid, localIov, remoteIov, 0)
	if err != nil {
		return nil, err
	}

	return &buffer, nil
}

// WriteSlice writes a buffer into addr
func (p *TracedProgram) WriteSlice(addr uint64, buffer []byte) error {
	size := len(buffer)

	// construct a local iovec (pointing to the data we want to write)
	localIov := []unix.Iovec{
		{
			Base: &buffer[0],
			Len:  uint64(size),
		},
	}

	// construct a remote iovec (the memory area to be written in the target process)
	remoteIov := []unix.RemoteIovec{
		{
			Base: uintptr(addr),
			Len:  size,
		},
	}

	// call ProcessVMWritev to write data to the target process memory
	_, err := unix.ProcessVMWritev(p.pid, localIov, remoteIov, 0)
	if err != nil {
		return err
	}

	return nil
}

func alignBuffer(buffer []byte) []byte {
	if buffer == nil {
		return nil
	}

	alignedSize := (len(buffer) / ptrSize) * ptrSize
	if alignedSize < len(buffer) {
		alignedSize += ptrSize
	}
	clonedBuffer := make([]byte, alignedSize)
	copy(clonedBuffer, buffer)

	return clonedBuffer
}

// PtraceWriteSlice uses ptrace rather than process_vm_write to write a buffer into addr
func (p *TracedProgram) PtraceWriteSlice(addr uint64, buffer []byte) error {
	wroteSize := 0

	buffer = alignBuffer(buffer)

	for wroteSize+ptrSize <= len(buffer) {
		_addr := uintptr(addr + uint64(wroteSize))
		data := buffer[wroteSize : wroteSize+ptrSize]

		_, err := unix.PtracePokeData(p.pid, _addr, data)
		if err != nil {
			return fmt.Errorf("%T write to addr %x with %+v failed", err, addr, data)
		}

		wroteSize += ptrSize
	}

	return nil
}

// GetLibBuffer reads an entry
func (p *TracedProgram) GetLibBuffer(entry *Entry) (*[]byte, error) {
	if entry.PaddingSize > 0 {
		return nil, fmt.Errorf("entry with padding size is not supported")
	}

	size := entry.EndAddress - entry.StartAddress

	return p.ReadSlice(entry.StartAddress, size)
}

// MmapSlice mmaps a slice and return it's addr
func (p *TracedProgram) MmapSlice(slice []byte) (*Entry, error) {
	size := uint64(len(slice))

	addr, err := p.Mmap(size, 0)
	if err != nil {
		return nil, err
	}

	err = p.WriteSlice(addr, slice)
	if err != nil {
		return nil, err
	}

	return &Entry{
		StartAddress: addr,
		EndAddress:   addr + size,
		Privilege:    "rwxp",
		PaddingSize:  0,
		Path:         "",
	}, nil
}

// FindSymbolInEntry finds symbol in entry through parsing elf
func (p *TracedProgram) FindSymbolInEntry(symbolName string, entry *Entry) (uint64, uint64, error) {
	libBuffer, err := p.GetLibBuffer(entry)
	if err != nil {
		return 0, 0, err
	}

	reader := bytes.NewReader(*libBuffer)
	vdsoElf, err := elf.NewFile(reader)
	if err != nil {
		return 0, 0, err
	}

	loadOffset := uint64(0)

	for _, prog := range vdsoElf.Progs {
		if prog.Type == elf.PT_LOAD {
			loadOffset = prog.Vaddr - prog.Off

			// break here is enough for vdso
			break
		}
	}

	symbols, err := vdsoElf.DynamicSymbols()
	if err != nil {
		return 0, 0, err
	}
	for _, symbol := range symbols {
		if symbol.Name == symbolName {
			offset := symbol.Value

			return entry.StartAddress + (offset - loadOffset), symbol.Size, nil
		}
	}
	return 0, 0, fmt.Errorf("cannot find symbol")
}

// WriteUint64ToAddr writes uint64 to addr
func (p *TracedProgram) WriteUint64ToAddr(addr uint64, value uint64) error {
	valueSlice := make([]byte, 8)
	endian.PutUint64(valueSlice, value)
	err := p.WriteSlice(addr, valueSlice)
	return err
}
