/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"context"
	"fmt"
	"github.com/keyvalstore/store"
	"github.com/sprintframework/sprint"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"strings"
	"sync"
)

var (
	ConfigBucket    = "config"
	ConfigBucketLen = len(ConfigBucket)
)

/**
	The purpose of this repository is to provide mutable property resolver that keeps state in the linked 'config-store' store from 'core' context.

	Alex Shvid
 */

type implConfigRepository struct {
	sync.Mutex
	Store   store.DataStore `inject:"bean=config-store"`

	priority int

	Log          *zap.Logger           `inject`

	watchNum  atomic.Int64
	watchMap  sync.Map       // watchNum, configWatchContext

	shuttingDown  atomic.Bool
}

type configEntryChange struct {
	key     string
	value   string   // deleted if value is empty string
}

type configWatchContext struct {
	ctx       context.Context
	cancelFn  context.CancelFunc
	prefix    string
	ch        chan<- configEntryChange
}

func ConfigRepository(priority int) sprint.ConfigRepository {
	t := &implConfigRepository{priority: priority}
	t.shuttingDown.Store(false)
	return t
}

func (t *implConfigRepository) String() string {
	return fmt.Sprintf("ConfigRepository{%d}", t.priority)
}

func (t *implConfigRepository) Priority() int {
	return t.priority
}

func (t *implConfigRepository) GetProperty(key string) (value string, ok bool) {
	if t.Backend() == nil || t.shuttingDown.Load() {
		return value, false
	}
	value, err := t.Get(key)
	if err != nil || value == "" {
		return "", false
	}
	return value, true
}

func (t *implConfigRepository) Destroy() error {
	t.shuttingDown.Store(true)
	if t.watchNum.Load() > 0 {
		t.watchMap.Range(func(key, value interface{}) bool {
			if wc, ok := value.(*configWatchContext); ok {
				wc.cancelFn()
			}
			return true
		})
	}
	return nil
}

func (t *implConfigRepository) Get(key string) (string, error) {
	value, err := t.Backend().Get(context.Background()).ByKey("%s:%s", ConfigBucket, key).ToString()
	if err != nil {
		return "", err
	}
	return value, nil
}

func (t *implConfigRepository) EnumerateAll(prefix string, cb func(key, value string) bool) error {
	return t.Backend().
		Enumerate(context.Background()).
		ByPrefix("%s:%s", ConfigBucket, prefix).
		WithBatchSize(256).
		Do(func(entry *store.RawEntry) bool {
			configKey := entry.Key[ConfigBucketLen+1:]
			return cb(string(configKey), string(entry.Value))
		})
}

func (t *implConfigRepository) Set(key, value string) error {
	err := t.doSet(key, value)
	if err != nil {
		return err
	}
	if t.watchNum.Load() != 0 {
		t.notifyAll(configEntryChange{key, value})
	}
	return nil
}

func (t *implConfigRepository) doSet(key, value string) error {
	if value == "" {
		return t.Backend().Remove(context.Background()).ByKey("%s:%s", ConfigBucket, key).Do()
	} else {
		return t.Backend().Set(context.Background()).ByKey("%s:%s", ConfigBucket, key).String(value)
	}
}

func (t *implConfigRepository) notifyAll(e configEntryChange) {
	t.watchMap.Range(func(key, value interface{}) bool {
		if wc, ok := value.(*configWatchContext); ok {
			if strings.HasPrefix(e.key, wc.prefix) {
				wc.ch <- e
			}
		}
		return true
	})
}

func (t *implConfigRepository) registerWatch(wc *configWatchContext) int64 {
	handle := t.watchNum.Inc()
	t.watchMap.Store(handle, wc)
	return handle
}

func (t *implConfigRepository) unregisterWatch(handle int64) {
	t.watchMap.Delete(handle)
}

// use Application as ctx
func (t *implConfigRepository) Watch(ctx context.Context, prefix string, cb func(key, value string) bool) (cancel context.CancelFunc, err error) {

	ctx, cancel = context.WithCancel(ctx)
	ch := make(chan configEntryChange)

	wc := &configWatchContext{
		ctx: ctx,
		cancelFn: cancel,
		prefix: prefix,
		ch: ch,
	}

	handle := t.registerWatch(wc)

	go func() {

		defer func() {
			if r := recover(); r != nil {
				switch v := r.(type) {
				case error:
					t.Log.Error("RecoverConfigWatcher", zap.Error(v))
				case string:
					t.Log.Error("RecoverConfigWatcher", zap.String("err", v))
				default:
					t.Log.Error("RecoverConfigWatcher", zap.String("err", fmt.Sprintf("%v", v)))
				}
			}
		}()

		defer func() {
			t.unregisterWatch(handle)
			close(ch)
		}()

		for {
			select {

			case <- ctx.Done():
				return

			case e := <- ch:
				if !cb(e.key, e.value) {
					return
				}

			}
		}

	}()

	return
}

func (t *implConfigRepository) Backend() store.DataStore {
	t.Lock()
	defer t.Unlock()
	return t.Store
}

func (t *implConfigRepository) SetBackend(store store.DataStore) {
	t.Lock()
	defer t.Unlock()
	t.Store = store
}
