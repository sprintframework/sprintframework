/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"os"
	"sync"
	"time"
	"fmt"
)

type implAutoupdateService struct {

	Log         *zap.Logger       `inject`
	Application sprint.Application `inject`

	AutoupdateFile    string      `value:"autoupdate.file,default="`

	watcher                *fsnotify.Watcher
	cacheStat              fileInformation
	triggerUpdateTimestamp atomic.Int64

	freezeNextId           atomic.Int64
	freezeMap              sync.Map  // key is handler int64, value is the job name (string)
	triggerAfterUnfreeze   atomic.Bool

	closeOnce              sync.Once
}

type fileInformation struct {
	Name    string
	Size    int64
	ModTime time.Time
}

func AutoupdateService() sprint.AutoupdateService {
	return &implAutoupdateService{}
}

func (t *implAutoupdateService) PostConstruct() error {
	autoupdateFile := t.AutoupdateFile
	if autoupdateFile != "" {
		err := t.doStart(autoupdateFile)
		if err != nil {
			t.Log.Error("AutoupdateWatcher", zap.String("autoupdateFile", autoupdateFile), zap.Error(err))
		}
		return err
	}
	return nil
}

func (t *implAutoupdateService) doStart(autoupdateFile string) error {

	var err error
	t.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return errors.Errorf("create watcher %v", err)
	}

	stat, err := os.Stat(autoupdateFile)
	if err != nil {
		return errors.Errorf("autoupdate file '%s' not found %v", autoupdateFile, err)
	}

	t.cacheStat.Name = autoupdateFile
	t.cacheStat.Size = stat.Size()
	t.cacheStat.ModTime = stat.ModTime()

	err = t.watcher.Add(autoupdateFile)
	if err != nil {
		return errors.Errorf("listen updates on file '%s' by watcher %v", autoupdateFile, err)
	}

	t.Log.Info("AutoupdateWatch", zap.String("autoupdateFile", autoupdateFile))
	go t.foregroundLoop()

	return nil
}

func (t *implAutoupdateService) foregroundLoop() {

	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				t.Log.Error("AutoupdateWatcher", zap.Error(v))
			case string:
				t.Log.Error("AutoupdateWatcher", zap.String("err", v))
			default:
				t.Log.Error("AutoupdateWatcher", zap.String("err", fmt.Sprintf("%v", v)))
			}
		}
	}()

	t.Log.Info("AutoupdateWatcherStart")
	for {
		select {
		case event, ok := <-t.watcher.Events:
			if ok {
				t.onEvent(event)
			}
		case err, ok := <-t.watcher.Errors:
			if ok {
				t.Log.Error("AutoupdateWatcher", zap.Error(err))
			}
		case <-t.Application.Done():
			t.Log.Info("AutoupdateWatcherStop")
			return
		}
	}
}


func (t *implAutoupdateService) onEvent(event fsnotify.Event) {
	if event.Op == fsnotify.Create || event.Op == fsnotify.Write {
		if stat, err := os.Stat(event.Name); err == nil {
			if t.cacheStat.Name != event.Name {
				t.Log.Error("AutoupdateName",
					zap.String("eventName", event.Name),
					zap.String("cacheName", t.cacheStat.Name),
				)
				return
			}
			if stat.Size() != t.cacheStat.Size || stat.ModTime().After(t.cacheStat.ModTime) {
				t.triggerUpdate()
			}
		}
	}
}

func (t *implAutoupdateService) Destroy() error {
	t.closeOnce.Do(func() {
		t.watcher.Close()
	})
	return nil
}

func (t *implAutoupdateService)	Freeze(jobName string) int64 {
	id := t.freezeNextId.Inc()
	t.freezeMap.Store(id, jobName)
	return id
}

func (t *implAutoupdateService)	Unfreeze(handle int64) {
	t.freezeMap.Delete(handle)
	if !t.hasFreezeJob() && t.triggerAfterUnfreeze.Load() {
		t.triggerUpdate()
	}
}

func (t *implAutoupdateService)	FreezeJobs() map[int64]string {
	cache := make(map[int64]string)
	t.freezeMap.Range(func(key, value interface{}) bool {
		id := key.(int64)
		name := value.(string)
		cache[id] = name
		return true
	})
	return cache
}

func (t *implAutoupdateService)	hasFreezeJob() (rez bool) {
	t.freezeMap.Range(func(key, value interface{}) bool {
		rez = true
		return false
	})
	return
}

func (t *implAutoupdateService) triggerUpdate() {
	if t.hasFreezeJob() {
		// what if someone unfreezed autoupdate during this nope before we setup triggerAfterUnfreeze=true :))) call again hasFreezeJob after
		t.triggerAfterUnfreeze.Store(true)
		if t.hasFreezeJob() {
			return
		}
	}
	current := time.Now().UnixNano()
	t.triggerUpdateTimestamp.Store(current)
	time.AfterFunc(time.Second, func() {
		if t.triggerUpdateTimestamp.Load() == current {
			if !isFileLocked(t.cacheStat.Name) {
				t.Log.Info("AutoupdateTriggerRestart")
				t.Application.Shutdown(true)
			} else {
				t.Log.Error("AutoupdateFileLocked", zap.String("cacheName", t.cacheStat.Name))
				// try yo update again
				t.triggerUpdate()
			}
		}
	})
}

func isFileLocked(filePath string) bool {
	if file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_EXCL, 0); err != nil {
		return true
	} else {
		file.Close()
		return false
	}
}

