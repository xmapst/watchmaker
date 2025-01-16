package watchmaker

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
)

// vdsoEntryName is the name of the vDSO entry
const vdsoEntryName = "[vdso]"

// FakeImage introduce the replacement of VDSO ELF entry and customizable variables.
// FakeImage could be constructed by LoadFakeImageFromEmbedFs(), and then used by FakeClockInjector.
type FakeImage struct {
	// symbolName is the name of the symbol to be replaced.
	symbolName string
	// content presents .text section which has been "manually relocation", the address of extern variables have been calculated manually
	content []byte
	// offset stores the table with variable name, and it's address in content.
	// the key presents extern variable name, ths value is the address/offset within the content.
	offset map[string]int
	// OriginFuncCode stores the raw func code like getTimeOfDay & ClockGetTime.
	OriginFuncCode []byte
	// OriginAddress stores the origin address of OriginFuncCode.
	OriginAddress uint64
	// fakeEntry stores the fake entry
	fakeEntry *Entry
}

func NewFakeImage(symbolName string, content []byte, offset map[string]int) *FakeImage {
	return &FakeImage{symbolName: symbolName, content: content, offset: offset}
}

// AttachToProcess would use ptrace to replace the VDSO ELF entry with FakeImage.
// Each item in parameter "variables" needs a corresponding entry in FakeImage.offset.
func (it *FakeImage) AttachToProcess(pid int, variables map[string]uint64) (err error) {
	if len(variables) != len(it.offset) {
		return fmt.Errorf("fake image: extern variable number not match")
	}

	runtime.LockOSThread()
	defer func() {
		runtime.UnlockOSThread()
	}()

	program, err := Trace(pid)
	if err != nil {
		return fmt.Errorf("%v ptrace on target process, pid: %d", err, pid)
	}
	defer func() {
		err = program.Detach()
		if err != nil {
			log.Println(err, "fail to detach program", "pid", program.Pid())
		}
	}()

	vdsoEntry, err := FindVDSOEntry(program)
	if err != nil {
		return fmt.Errorf("%v PID : %d", err, pid)
	}

	fakeEntry, err := it.FindInjectedImage(program, len(variables))
	if err != nil {
		return fmt.Errorf("%v PID : %d", err, pid)
	}
	// target process has not been injected yet
	if fakeEntry == nil {
		fakeEntry, err = it.InjectFakeImage(program, vdsoEntry)
		if err != nil {
			return fmt.Errorf("%v injecting fake image , PID : %d", err, pid)
		}
		defer func() {
			if err != nil {
				errIn := it.TryReWriteFakeImage(program)
				if errIn != nil {
					log.Println(errIn, "rewrite fail, recover fail")
				}
				it.OriginFuncCode = nil
				it.OriginAddress = 0
			}
		}()
	}

	for k, v := range variables {
		err = it.SetVarUint64(program, fakeEntry, k, v)

		if err != nil {
			return fmt.Errorf("%v set %s for time skew, pid: %d", err, k, pid)
		}
	}

	return
}

func FindVDSOEntry(program *TracedProgram) (*Entry, error) {
	var vdsoEntry *Entry
	for index := range program.Entries {
		// reverse loop is faster
		e := program.Entries[len(program.Entries)-index-1]
		if e.Path == vdsoEntryName {
			vdsoEntry = &e
			break
		}
	}
	if vdsoEntry == nil {
		return nil, fmt.Errorf("VDSOEntry is not found")
	}
	return vdsoEntry, nil
}

// FindInjectedImage find injected image to avoid redundant inject.
func (it *FakeImage) FindInjectedImage(program *TracedProgram, varNum int) (*Entry, error) {
	// minus tailing variable part
	// every variable has 8 bytes
	if it.fakeEntry != nil {
		content, err := program.ReadSlice(it.fakeEntry.StartAddress, it.fakeEntry.EndAddress-it.fakeEntry.StartAddress)
		if err != nil {
			log.Println("ReadSlice fail")
			return nil, nil
		}
		if varNum*8 > len(it.content) {
			return nil, fmt.Errorf("variable num bigger than content num")
		}
		contentWithoutVariable := (*content)[:len(it.content)-varNum*varLength]
		expectedContentWithoutVariable := it.content[:len(it.content)-varNum*varLength]
		log.Println("successfully read slice", "content", contentWithoutVariable, "expected content", expectedContentWithoutVariable)

		if bytes.Equal(contentWithoutVariable, expectedContentWithoutVariable) {
			log.Println("slice found")
			return it.fakeEntry, nil
		}
		log.Println("slice not found")
	}
	return nil, nil
}

// InjectFakeImage Usage CheckList:
// When error : TryReWriteFakeImage after InjectFakeImage.
func (it *FakeImage) InjectFakeImage(program *TracedProgram,
	vdsoEntry *Entry) (*Entry, error) {
	fakeEntry, err := program.MmapSlice(it.content)
	if err != nil {
		return nil, fmt.Errorf("%T mmap fake image", err)
	}
	it.fakeEntry = fakeEntry
	originAddr, size, err := program.FindSymbolInEntry(it.symbolName, vdsoEntry)
	if err != nil {
		return nil, fmt.Errorf("%T find origin %s in vdso", err, it.symbolName)
	}
	funcBytes, err := program.ReadSlice(originAddr, size)
	if err != nil {
		return nil, fmt.Errorf("%T ReadSlice failed", err)
	}
	err = program.JumpToFakeFunc(originAddr, fakeEntry.StartAddress)
	if err != nil {
		errIn := it.TryReWriteFakeImage(program)
		if errIn != nil {
			log.Println(errIn, "rewrite fail, recover fail")
		}
		return nil, fmt.Errorf("%T override origin %s", err, it.symbolName)
	}

	it.OriginFuncCode = *funcBytes
	it.OriginAddress = originAddr
	return fakeEntry, nil
}

func (it *FakeImage) TryReWriteFakeImage(program *TracedProgram) error {
	if it.OriginFuncCode != nil {
		err := program.PtraceWriteSlice(it.OriginAddress, it.OriginFuncCode)
		if err != nil {
			return err
		}
		it.OriginFuncCode = nil
		it.OriginAddress = 0
	}
	return nil
}

// Recover the injected image. If injected image not found ,
// Recover will not return error.
func (it *FakeImage) Recover(pid int, vars map[string]uint64) error {
	runtime.LockOSThread()
	defer func() {
		runtime.UnlockOSThread()
	}()
	if it.OriginFuncCode == nil {
		return nil
	}
	program, err := Trace(pid)
	if err != nil {
		return fmt.Errorf("%T ptrace on target process, pid: %d", err, pid)
	}
	defer func() {
		err = program.Detach()
		if err != nil {
			log.Println(err, "fail to detach program", "pid", program.Pid())
		}
	}()

	fakeEntry, err := it.FindInjectedImage(program, len(vars))
	if err != nil {
		return fmt.Errorf("%T FindInjectedImage , pid: %d", err, pid)
	}
	if fakeEntry == nil {
		return nil
	}

	err = it.TryReWriteFakeImage(program)
	return err
}
