package watchmaker

import (
	"encoding/binary"
	"fmt"
	"syscall"
)

var endian = binary.LittleEndian

const syscallInstrSize = 2

const nrProcessVMReadv = 310
const nrProcessVMWritev = 311

func getIp(regs *syscall.PtraceRegs) uintptr {
	return uintptr(regs.Rip)
}

func getRegs(pid int, regsout *syscall.PtraceRegs) error {
	err := syscall.PtraceGetRegs(pid, regsout)
	if err != nil {
		return fmt.Errorf("%T get registers of process %d", err, pid)
	}

	return nil
}

func setRegs(pid int, regs *syscall.PtraceRegs) error {
	err := syscall.PtraceSetRegs(pid, regs)
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

	var regs syscall.PtraceRegs

	err = getRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}
	// set the registers according to the syscall convention. Learn more about
	// it in `man 2 syscall`. In x86_64 the syscall nr is stored in rax
	// register, and the arguments are stored in rdi, rsi, rdx, r10, r8, r9 in
	// order
	regs.Rax = number
	for index, arg := range args {
		// All these registers are hard coded for x86 platform
		if index == 0 {
			regs.Rdi = arg
		} else if index == 1 {
			regs.Rsi = arg
		} else if index == 2 {
			regs.Rdx = arg
		} else if index == 3 {
			regs.R10 = arg
		} else if index == 4 {
			regs.R8 = arg
		} else if index == 5 {
			regs.R9 = arg
		} else {
			return 0, fmt.Errorf("too many arguments for a syscall")
		}
	}
	err = setRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}

	instruction := make([]byte, syscallInstrSize)
	ip := getIp(p.backupRegs)

	// set the current instruction (the ip register points to) to the `syscall`
	// instruction. In x86_64, the `syscall` instruction is 0x050f.
	binary.LittleEndian.PutUint16(instruction, 0x050f)
	_, err = syscall.PtracePokeData(p.pid, ip, instruction)
	if err != nil {
		return 0, fmt.Errorf("%T writing data %v to %x", err, instruction, ip)
	}

	// run one instruction, and stop
	err = p.Step()
	if err != nil {
		return 0, err
	}

	// read registers, the return value of syscall is stored inside rax register
	err = getRegs(p.pid, &regs)
	if err != nil {
		return 0, err
	}

	// restore the state saved at beginning.
	return regs.Rax, p.Restore()
}

// JumpToFakeFunc writes jmp instruction to jump to fake function
func (p *TracedProgram) JumpToFakeFunc(originAddr uint64, targetAddr uint64) error {
	instructions := make([]byte, 16)

	// mov rax, targetAddr;
	// jmp rax ;
	instructions[0] = 0x48
	instructions[1] = 0xb8
	endian.PutUint64(instructions[2:10], targetAddr)
	instructions[10] = 0xff
	instructions[11] = 0xe0

	return p.PtraceWriteSlice(originAddr, instructions)
}
