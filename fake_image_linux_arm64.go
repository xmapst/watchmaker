package watchmaker

import (
	"fmt"
)

// clockGettimeSkewFakeImage is the filename of fake image after compiling
const clockGettimeSkewFakeImage = "fake_clock_gettime_arm64.o"

// timeofdaySkewFakeImage is the filename of fake image after compiling
const timeOfDaySkewFakeImage = "fake_gettimeofday_arm64.o"

// one variable will use two pointers place
const varLength = 16

func (it *FakeImage) SetVarUint64(program *TracedProgram, entry *Entry, symbol string, value uint64) error {
	if offset, ok := it.offset[symbol]; ok {
		variableOffset := entry.StartAddress + uint64(offset) + 8

		err := program.WriteUint64ToAddr(entry.StartAddress+uint64(offset), variableOffset)
		if err != nil {
			return err
		}

		err = program.WriteUint64ToAddr(variableOffset, value)
		return err
	}

	return fmt.Errorf("symbol not found")
}
