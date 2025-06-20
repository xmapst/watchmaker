package watchmaker

import (
	"encoding/binary"
	"fmt"
	"log"

	"golang.org/x/sys/unix"
)

var endian = binary.LittleEndian

const unixInstrSize = 4

// see kernel source /include/uapi/linux/elf.h
const nrPRStatus = 1

func getIp(regs *unix.PtraceRegs) uintptr {
	return uintptr(regs.Pc)
}

func getRegs(pid int, regsout *unix.PtraceRegs) error {
	err := unix.PtraceGetRegSetArm64(pid, nrPRStatus, (*unix.PtraceRegsArm64)(regsout))
	if err != nil {
		return fmt.Errorf("%T get registers of process %d", err, pid)
	}
	return nil
}

func setRegs(pid int, regs *unix.PtraceRegs) error {
	err := unix.PtraceSetRegSetArm64(pid, nrPRStatus, (*unix.PtraceRegsArm64)(regs))
	if err != nil {
		return fmt.Errorf("%T set registers of process %d", err, pid)
	}
	return nil
}

// Syscall runs a syscall at main thread of process
func (p *TracedProgram) Syscall(number uint64, args ...uint64) (uint64, error) {
	// save the original registers and the current instructions
	err := p.Protect()
	if err != nil {
		return 0, err
	}

	var regs unix.PtraceRegs

	err = getRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}
	// set the registers according to the syscall convention. Learn more about
	// it in `man 2 syscall`. In aarch64 the syscall nr is stored in w8, and the
	// arguments are stored in x0, x1, x2, x3, x4, x5 in order
	regs.Regs[8] = number
	for index, arg := range args {
		if index > 6 {
			return 0, fmt.Errorf("too many arguments for a syscall")
		} else {
			regs.Regs[index] = arg
		}
	}
	err = setRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}

	instruction := make([]byte, unixInstrSize)
	ip := getIp(p.backupRegs)

	// most aarch64 devices are little endian
	// 0xd4000001 is `svc #0` to call the system call
	endian.PutUint32(instruction, 0xd4000001)
	_, err = unix.PtracePokeData(p.pid, ip, instruction)
	if err != nil {
		return 0, fmt.Errorf("%T writing data %v to %x", err, instruction, ip)
	}

	// run one instruction, and stop
	log.Printf("[SYSCALL DEBUG] running %x from %#x", instruction, ip)
	err = p.Step()
	if err != nil {
		return 0, err
	}

	// read registers, the return value of syscall is stored inside x0 register
	err = getRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}
	log.Printf("[SYSCALL DEBUG] regs: %v", regs)

	// TODO: why "strategy 1" mmap() is failing on arm64 with 0 returned from this proc?
	// https://stackoverflow.com/questions/37167141/linux-syscalls-and-errno

	return regs.Regs[0], p.Restore()
}

// JumpToFakeFunc writes jmp instruction to jump to fake function
func (p *TracedProgram) JumpToFakeFunc(originAddr uint64, targetAddr uint64) error {
	instructions := make([]byte, 16)

	// LDR x9, #8
	// BR x9
	// targetAddr
	endian.PutUint32(instructions[0:], 0x58000049)
	endian.PutUint32(instructions[4:], 0xD61F0120)

	endian.PutUint64(instructions[8:], targetAddr)

	return p.PtraceWriteSlice(originAddr, instructions)
}
