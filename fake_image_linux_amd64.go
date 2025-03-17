package watchmaker

import (
	"fmt"
)

const timeSkewFakeImage = "fake_time_amd64.o"

// clockGettimeSkewFakeImage is the filename of fake image after compiling
const clockGettimeSkewFakeImage = "fake_clock_gettime_amd64.o"

// timeofdaySkewFakeImage is the filename of fake image after compiling
const timeOfDaySkewFakeImage = "fake_gettimeofday_amd64.o"

const varLength = 8

func (it *FakeImage) SetVarUint64(program *TracedProgram, entry *Entry, symbol string, value uint64) error {
	if offset, ok := it.offset[symbol]; ok {
		err := program.WriteUint64ToAddr(entry.StartAddress+uint64(offset), value)
		return err
	}

	return fmt.Errorf("symbol not found")
}
