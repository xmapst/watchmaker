// Copyright 2021 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package time

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/xmapst/logx"

	"github.com/xmapst/watchmaker/internal/cerr"
	"github.com/xmapst/watchmaker/internal/tasks"
)

// clockGettimeSkewFakeImage is the filename of fake image after compiling
const clockGettimeSkewFakeImage = "fake_clock_gettime.o"

// clockGettime is the target function would be replaced
const clockGettime = "clock_gettime"

// These three consts corresponding to the three extern variables in the fake_clock_gettime.c
const (
	externVarClockIdsMask = "CLOCK_IDS_MASK"
	externVarTvSecDelta   = "TV_SEC_DELTA"
	externVarTvNsecDelta  = "TV_NSEC_DELTA"
)

// timeofdaySkewFakeImage is the filename of fake image after compiling
const timeOfDaySkewFakeImage = "fake_gettimeofday.o"

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

func NewConfig(deltaSeconds int64, deltaNanoSeconds int64, clockIDsMask uint64) Config {
	return Config{
		deltaSeconds:     deltaSeconds,
		deltaNanoSeconds: deltaNanoSeconds,
		clockIDsMask:     clockIDsMask,
	}
}

func (c *Config) DeepCopy() tasks.Object {
	return &Config{
		c.deltaSeconds,
		c.deltaNanoSeconds,
		c.clockIDsMask,
	}
}

// Merge implement how to merge time skew tasks.
func (c *Config) Merge(a tasks.Mergeable) error {
	A, OK := a.(*Config)
	if OK {
		// TODO: Add more reasonable merge method
		c.deltaSeconds += A.deltaSeconds
		c.deltaNanoSeconds += A.deltaNanoSeconds
		c.clockIDsMask |= A.clockIDsMask
		return nil
	}
	return cerr.NotType[*Config]().WrapInput(a).Err()
}

type ConfigCreatorParas struct {
	Config        Config
	PodProcessMap *tasks.PodContainerNameProcessMap
}

// New assumes we get ConfigCreatorParas from values.
// New will init a struct just like PodHandler(ProcessGroupHandler(Skew))
func (c *Config) New(values interface{}) (tasks.Injectable, error) {
	paras, ok := values.(ConfigCreatorParas)
	if !ok {
		return nil, errors.New("not ConfigCreatorParas")
	}

	skew, err := GetSkew(paras.Config)
	if err != nil {
		return nil, err
	}

	newGroupProcessHandler :=
		tasks.NewProcessGroupHandler(&skew)
	newPodHandler := tasks.NewPodHandler(paras.PodProcessMap,
		&newGroupProcessHandler)
	return &newPodHandler, nil
}

// Assign assumes the input injectable is *tasks.PodHandler.
// We also assume the SubProcess of podHandler is *tasks.ProcessGroupHandler
// and the LeaderProcess of ProcessGroupHandler is *Skew.
func (c *Config) Assign(injectable tasks.Injectable) error {
	podHandler, ok := injectable.(*tasks.PodHandler)
	if !ok {
		return errors.New(fmt.Sprintf("type %t is not *tasks.PodHandler", injectable))
	}
	groupProcessHandler, ok := podHandler.SubProcess.(*tasks.ProcessGroupHandler)
	if !ok {
		return errors.New(fmt.Sprintf("type %t is not *tasks.ProcessGroupHandler", podHandler.SubProcess))
	}
	I, ok := groupProcessHandler.LeaderProcess.(*Skew)
	if !ok {
		return errors.New(fmt.Sprintf("type %t is not *Skew", groupProcessHandler.LeaderProcess))
	}

	I.SkewConfig = *c
	return nil
}

// Skew implements ProcessGroup.
// We locked Skew injecting and recovering to avoid conflict.
type Skew struct {
	SkewConfig   Config
	clockGetTime *FakeImage
	getTimeOfDay *FakeImage

	locker sync.Mutex
}

func GetSkew(c Config) (Skew, error) {
	clockGetTimeImage, err := LoadFakeImageFromEmbedFs(clockGettimeSkewFakeImage, clockGettime)
	if err != nil {
		return Skew{}, errors.Wrap(err, "load fake image")
	}

	getTimeOfDayimage, err := LoadFakeImageFromEmbedFs(timeOfDaySkewFakeImage, getTimeOfDay)
	if err != nil {
		return Skew{}, errors.Wrap(err, "load fake image")
	}

	return Skew{
		SkewConfig:   c,
		clockGetTime: clockGetTimeImage,
		getTimeOfDay: getTimeOfDayimage,
		locker:       sync.Mutex{},
	}, nil
}

func (s *Skew) Fork() (tasks.ChaosOnProcessGroup, error) {
	// TODO : to KEAO can I share FakeImage between threads?
	skew, err := GetSkew(s.SkewConfig)
	if err != nil {
		return nil, err
	}

	return &skew, nil
}

func (s *Skew) Assign(injectable tasks.Injectable) error {
	I, OK := injectable.(*Skew)
	if OK {
		I.SkewConfig = *s.SkewConfig.DeepCopy().(*Config)
		return nil
	}
	return cerr.NotType[*Skew]().WrapInput(injectable).Err()
}

func (s *Skew) Inject(pid tasks.IsID) error {
	s.locker.Lock()
	defer s.locker.Unlock()
	sysPID, ok := pid.(tasks.SysPID)
	if !ok {
		return tasks.ErrNotTypeSysID.WrapInput(pid).Err()
	}

	logx.Infoln("injecting time skew", "pid", pid)

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
func (s *Skew) Recover(pid tasks.IsID) error {
	s.locker.Lock()
	defer s.locker.Unlock()
	sysPID, ok := pid.(tasks.SysPID)
	if !ok {
		return tasks.ErrNotTypeSysID.WrapInput(pid).Err()
	}

	logx.Infoln("recovering time skew", "pid", pid)

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
			return errors.Wrapf(err1, "time skew all failed %v", err2)
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
