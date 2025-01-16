package watchmaker

import (
	"fmt"
	"sync"
)

// clockGettime is the target function would be replaced
const clockGettime = "clock_gettime"

// These three consts corresponding to the three extern variables in the fake_clock_gettime.c
const (
	externVarClockIdsMask = "CLOCK_IDS_MASK"
	externVarTvSecDelta   = "TV_SEC_DELTA"
	externVarTvNsecDelta  = "TV_NSEC_DELTA"
)

// getTimeOfDay is the target function would be replaced
const getTimeOfDay = "gettimeofday"

// Config is the summary config of get_time_of_day and clock_get_time.
// Config here is only for injector of k8s pod.
// We divide group injector on linux process , pod injector for k8s and
// the base injector , so we can simply create another config struct just
// for linux process for watchmaker.
type Config struct {
	deltaSeconds     int64
	deltaNanoSeconds int64
	clockIDsMask     uint64
}

func NewConfig(deltaSeconds int64, deltaNanoSeconds int64, clockIDsMask uint64) *Config {
	return &Config{
		deltaSeconds:     deltaSeconds,
		deltaNanoSeconds: deltaNanoSeconds,
		clockIDsMask:     clockIDsMask,
	}
}

func (c *Config) DeepCopy() *Config {
	return &Config{
		deltaSeconds:     c.deltaSeconds,
		deltaNanoSeconds: c.deltaNanoSeconds,
		clockIDsMask:     c.clockIDsMask,
	}
}

// Merge implement how to merge time skew tasks.
func (c *Config) Merge(a *Config) {
	// TODO: Add more reasonable merge method
	c.deltaSeconds += a.deltaSeconds
	c.deltaNanoSeconds += a.deltaNanoSeconds
	c.clockIDsMask |= a.clockIDsMask
	return
}

type ConfigCreatorParas struct {
	Config Config
}

// Skew implements ProcessGroup.
// We locked Skew injecting and recovering to avoid conflict.
type Skew struct {
	SkewConfig   *Config
	clockGetTime *FakeImage
	getTimeOfDay *FakeImage

	locker sync.Mutex
}

func GetSkew(c *Config) (*Skew, error) {
	clockGetTimeImage, err := LoadFakeImageFromEmbedFs(clockGettimeSkewFakeImage, clockGettime)
	if err != nil {
		return nil, fmt.Errorf("load fake image err: %v", err)
	}

	getTimeOfDayimage, err := LoadFakeImageFromEmbedFs(timeOfDaySkewFakeImage, getTimeOfDay)
	if err != nil {
		return nil, fmt.Errorf("load fake image err: %v", err)
	}

	return &Skew{
		SkewConfig:   c,
		clockGetTime: clockGetTimeImage,
		getTimeOfDay: getTimeOfDayimage,
		locker:       sync.Mutex{},
	}, nil
}

func (s *Skew) Fork() (*Skew, error) {
	// TODO : to KEAO can I share FakeImage between threads?
	skew, err := GetSkew(s.SkewConfig)
	if err != nil {
		return nil, err
	}

	return skew, nil
}

func (s *Skew) Inject(sysPID uint64) error {
	s.locker.Lock()
	defer s.locker.Unlock()

	err := s.clockGetTime.AttachToProcess(int(sysPID), map[string]uint64{
		externVarClockIdsMask: s.SkewConfig.clockIDsMask,
		externVarTvSecDelta:   uint64(s.SkewConfig.deltaSeconds),
		externVarTvNsecDelta:  uint64(s.SkewConfig.deltaNanoSeconds),
	})
	if err != nil {
		return err
	}

	err = s.getTimeOfDay.AttachToProcess(int(sysPID), map[string]uint64{
		externVarTvSecDelta:  uint64(s.SkewConfig.deltaSeconds),
		externVarTvNsecDelta: uint64(s.SkewConfig.deltaNanoSeconds),
	})
	if err != nil {
		return err
	}
	return nil
}

// Recover clock_get_time & get_time_of_day one by one ,
// if error comes from clock_get_time.Recover we will continue recover another fake image
// and merge errors.
func (s *Skew) Recover(sysPID uint64) error {
	s.locker.Lock()
	defer s.locker.Unlock()

	err1 := s.clockGetTime.Recover(int(sysPID), map[string]uint64{
		externVarClockIdsMask: s.SkewConfig.clockIDsMask,
		externVarTvSecDelta:   uint64(s.SkewConfig.deltaSeconds),
		externVarTvNsecDelta:  uint64(s.SkewConfig.deltaNanoSeconds),
	})
	if err1 != nil {
		err2 := s.getTimeOfDay.Recover(int(sysPID), map[string]uint64{
			externVarTvSecDelta:  uint64(s.SkewConfig.deltaSeconds),
			externVarTvNsecDelta: uint64(s.SkewConfig.deltaNanoSeconds),
		})
		if err2 != nil {
			return fmt.Errorf("%v time skew all failed %v", err1, err2)
		}
		return err1
	}

	err2 := s.getTimeOfDay.Recover(int(sysPID), map[string]uint64{
		externVarTvSecDelta:  uint64(s.SkewConfig.deltaSeconds),
		externVarTvNsecDelta: uint64(s.SkewConfig.deltaNanoSeconds),
	})
	if err2 != nil {
		return err2
	}

	return nil
}
