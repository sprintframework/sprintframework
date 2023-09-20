/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"github.com/codeallergy/uuid"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const oneMb = 1024 * 1024

type implNodeService struct {
	Application         sprint.Application         `inject`
	ApplicationFlags    sprint.ApplicationFlags    `inject`
	Properties       glue.Properties          `inject`
	ConfigRepository sprint.ConfigRepository  `inject`
	Log              *zap.Logger              `inject`

	initOnce sync.Once

	nodeIdHex string
	nodeId    uint64
	nodeName  string

	lastTimestamp atomic.Int64
	clock         atomic.Int32
}

func NodeService() sprint.NodeService {
	return &implNodeService{}
}

func (t *implNodeService) BeanName() string {
	return "node_service"
}

func (t *implNodeService) GetStats(cb func(name, value string) bool) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cb("id", t.nodeIdHex)
	cb("name", t.nodeName)
	cb("numGoroutine", strconv.Itoa(runtime.NumGoroutine()))
	cb("numCPU", strconv.Itoa(runtime.NumCPU()))
	cb("numCgoCall", strconv.FormatInt(runtime.NumCgoCall(), 10))
	cb("goVersion", runtime.Version())
	cb("memAlloc", fmt.Sprintf("%dmb", m.Alloc / oneMb))
	cb("memTotalAlloc", fmt.Sprintf("%dmb", m.TotalAlloc / oneMb))
	cb("memSys", fmt.Sprintf("%dmb", m.Sys / oneMb))
	cb("memNumGC", strconv.Itoa(int(m.NumGC)))

	return nil
}

func (t *implNodeService) PostConstruct() (err error) {

	defer util.PanicToError(&err)

	seq := t.ApplicationFlags.Node()
	if seq == 0 {
		t.nodeName = t.Application.Name()
	} else {
		t.nodeName = fmt.Sprintf("%s-%d", t.Application.Name(), t.ApplicationFlags.Node())
	}

	t.nodeIdHex = t.Properties.GetString("node.id", "")
	if t.nodeIdHex == "" {
		t.nodeIdHex, err = util.GenerateNodeId()
		if err != nil {
			return errors.Errorf("generate node id, %v", err)
		}
		err = t.ConfigRepository.Set("node.id", t.nodeIdHex)
		if err != nil {
			return errors.Errorf("set property 'node.id' with value '%s', %v", t.nodeIdHex, err)
		}
	}
	t.nodeId, err = util.ParseNodeId(t.nodeIdHex)
	return err
}

func (t *implNodeService) NodeId() uint64 {
	return t.nodeId
}

func (t *implNodeService) NodeIdHex() string {
	return t.nodeIdHex
}

func (t *implNodeService) NodeName() string {
	return t.nodeName
}

func (t *implNodeService) Issue() uuid.UUID {

	id := uuid.New(uuid.TimebasedVer1)
	id.SetTime(time.Now())
	id.SetNode(int64(t.nodeId))

	for {

		curr := id.UnixTime100Nanos()
		old := t.lastTimestamp.Load()
		if old == curr {
			id.SetClockSequence(int(t.clock.Inc()))
			break
		}

		if t.lastTimestamp.CAS(old, curr) {
			t.clock.Store(0)
			break
		}

		old = t.lastTimestamp.Load()
		if old > curr {
			id.SetTime(time.Now())
		}

	}

	return id

}

func (t *implNodeService) Parse(id uuid.UUID) (timestampMillis int64, nodeId int64, clock int) {
	return id.UnixTimeMillis(), id.Node(), id.ClockSequence()
}

