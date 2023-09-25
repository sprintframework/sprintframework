/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcore

import (
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/codeallergy/uuid"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"go.uber.org/atomic"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const oneMb = 1024 * 1024

type implNodeService struct {
	Application         sprint.Application         `inject`
	ApplicationFlags    sprint.ApplicationFlags    `inject`
	Properties          glue.Properties            `inject`
	ConfigRepository    sprint.ConfigRepository    `inject`

	initOnce sync.Once

	nodeIdHex string
	nodeId    uint64
	nodeSeq   int

	LocalNodeName       string         `value:"node.local,default="`
	LANNodeName         string         `value:"node.lan,default="`
	WANNodeName         string         `value:"node.wan,default="`
	DataCenterName      string         `value:"node.dc,default=default"`

	/**
	Default: Host Name + Node Sequence Number
	*/

	advertiseName  string

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
	cb("local", t.LocalNodeName)
	cb("lan", t.LANNodeName)
	cb("wan", t.WANNodeName)
	cb("dc", t.DataCenterName)
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

	defer sprintutils.PanicToError(&err)

	t.nodeIdHex = t.Properties.GetString("node.id", "")
	if t.nodeIdHex == "" {
		t.nodeIdHex, err = sprintutils.GenerateNodeId()
		if err != nil {
			return errors.Errorf("generate node id, %v", err)
		}
		err = t.ConfigRepository.Set("node.id", t.nodeIdHex)
		if err != nil {
			return errors.Errorf("set property 'node.id' with value '%s', %v", t.nodeIdHex, err)
		}
	}

	t.nodeId, err = sprintutils.ParseNodeId(t.nodeIdHex)
	if err != nil {
		return err
	}

	t.nodeSeq = t.ApplicationFlags.Node()

	if t.LocalNodeName == "" {
		t.LocalNodeName = sprintutils.AppendNodeSequence(t.Application.Name(), t.nodeSeq)
	}

	if t.LANNodeName == "" {

		hostname, err := os.Hostname()
		if err != nil {
			return err
		}

		t.LANNodeName = sprintutils.AppendNodeName(t.LocalNodeName, hostname)
	}

	if t.WANNodeName == "" {
		t.WANNodeName = sprintutils.AppendNodeName(t.LANNodeName, t.DataCenterName)
	}

	return nil
}

func (t *implNodeService) NodeId() uint64 {
	return t.nodeId
}

func (t *implNodeService) NodeIdHex() string {
	return t.nodeIdHex
}

func (t *implNodeService) LocalName() string {
	return t.LocalNodeName
}

func (t *implNodeService) LANName() string {
	return t.LANNodeName
}

func (t *implNodeService) WANName() string {
	return t.WANNodeName
}

func (t *implNodeService) DCName() string {
	return t.DataCenterName
}

func (t *implNodeService) NodeSeq() int {
	return t.nodeSeq
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

