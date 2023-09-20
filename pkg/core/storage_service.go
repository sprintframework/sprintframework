/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprintpb"
	"github.com/sprintframework/sprintframework/pkg/server"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"github.com/keyvalstore/store"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type implStorageService struct {
	Application sprint.Application `inject`
	Properties  glue.Properties `inject`

	StorageMap    map[string]store.ManagedDataStore `inject`
	Log           *zap.Logger                           `inject`

	availableStorages []string

	BackupFilePerm   os.FileMode   `value:"application.perm.backup.file,default=-rw-rw-r--"`

}

func StorageService() sprint.StorageService {
	return &implStorageService{}
}

func (t *implStorageService) PostConstruct() error {
	var names []string
	for k, _ := range t.StorageMap {
		names = append(names, k)
	}
	sort.Strings(names)
	t.availableStorages = names
	return nil
}

func (t *implStorageService) ExecuteQuery(name, query string, cb func(string) bool) (err error) {

	defer util.PanicToError(&err)

	s, ok := t.StorageMap[name]
	if !ok {
		return errors.Errorf("storage '%s' is not found", name)
	}

	query = strings.Trim(query, "")
	if strings.HasSuffix(query, ";") {
		query = query[:len(query)-1]
	}

	cmdEnd := strings.IndexByte(query, ' ')
	if cmdEnd == -1 {
		cmdEnd = len(query)
	}

	cmd := query[:cmdEnd]
	args := strings.TrimSpace(query[cmdEnd:])

	switch cmd {
	case "help":
		if !cb("available commands: help, list, get, set, rm, dump, scan, search") {
			return server.ErrInterrupted
		}
	case "list":
		if !cb(strings.Join(t.availableStorages, ", ")) {
			return server.ErrInterrupted
		}
	case "get":
		key := []byte(args)
		if value, err := s.Get(context.Background()).ByRawKey(key).ToBinary(); err != nil {
			return err
		} else if !cb(base64.StdEncoding.EncodeToString(value)) {
			return server.ErrInterrupted
		}
	case "set":
		keyEnd := strings.IndexByte(args, ' ')
		if keyEnd == -1 {
			return errors.New("not enough args, usage: set key value")
		}
		key := strings.TrimSpace(args[:keyEnd])
		value := strings.TrimSpace(args[keyEnd:])
		valueBase64, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return err
		}
		if err := s.Set(context.Background()).ByRawKey([]byte(key)).Binary(valueBase64); err != nil {
			return err
		} else if !cb("OK") {
			return server.ErrInterrupted
		}
	case "rm":
		key := []byte(args)
		if err := s.Remove(context.Background()).ByRawKey(key).Do(); err != nil {
			return err
		} else if !cb("OK") {
			return server.ErrInterrupted
		}
	case "dump":
		prefix := []byte(args)
		return s.Enumerate(context.Background()).ByRawPrefix(prefix).Do(func(entry *store.RawEntry) bool {
			return cb(fmt.Sprintf("%s, %s", string(entry.Key), base64.StdEncoding.EncodeToString(entry.Value)))
		})
	case "scan":
		prefix := []byte(args)
		return s.Enumerate(context.Background()).ByRawPrefix(prefix).OnlyKeys().Do(func(entry *store.RawEntry) bool {
			return cb(string(entry.Key))
		})
	case "search":
		pattern := args
		return s.Enumerate(context.Background()).OnlyKeys().Do(func(entry *store.RawEntry) bool {
			if ok, _ := filepath.Match(pattern, string(entry.Key)); ok {
				return cb(fmt.Sprintf("%s", string(entry.Key)))
			}
			return true
		})
	default:
		return errors.Errorf("unknown command cmd=%s, args=%s", cmd, args)
	}

	return nil
}

func (t *implStorageService) Console(stream sprint.StorageConsoleStream) error {

	defaultStorage := "config-storage"

	for {
		request, err := stream.Recv()
		if err != nil {
			break
		}

		if strings.HasPrefix(request.Query, "use ") {
			newStorage := strings.TrimSpace(request.Query[4:])
			if _, ok := t.StorageMap[newStorage]; !ok {
				rec := &sprintpb.StorageConsoleResponse{
					Status:  200,
					Content: fmt.Sprintf("error: storage '%s' not found, available storages '%+v", newStorage, t.availableStorages),
				}
				err = stream.Send(rec)
			} else {
				defaultStorage = newStorage
				rec := &sprintpb.StorageConsoleResponse{
					Status:  200,
					Content: fmt.Sprintf("selected storage '%s'", defaultStorage),
				}

				err = stream.Send(rec)
			}

		} else {

			err = t.ExecuteQuery(defaultStorage, request.Query, func(content string) bool {

				rec := &sprintpb.StorageConsoleResponse{
					Status:  200,
					Content: content,
				}

				return stream.Send(rec) == nil

			})

		}

		if err != nil {
			err = stream.Send(&sprintpb.StorageConsoleResponse{
				Status:  501,
				Content: fmt.Sprintf("internal error, %v", err),
			})
		}

		err = stream.Send(&sprintpb.StorageConsoleResponse{
			Status: 100,
		})

		if err != nil {
			return err
		}

	}

	return nil
}

func (t *implStorageService) LocalConsole(writer io.StringWriter, errWriter io.StringWriter) error {

	defaultStorage := "config-storage"

	for {
		query := util.Prompt("Enter query [exit]: ")
		if query == "" {
			continue
		}
		if query == "exit" {
			break
		}
		if query == "list" {
			writer.WriteString(fmt.Sprintf("%+v\n", t.availableStorages))
			continue
		}

		if strings.HasPrefix(query, "use ") {
			newStorage := strings.TrimSpace(query[4:])
			if _, ok := t.StorageMap[newStorage]; !ok {
				errWriter.WriteString(fmt.Sprintf("error: storage '%s' not found, available storages '%+v\n", newStorage, t.availableStorages))
			} else {
				defaultStorage = newStorage
				writer.WriteString(fmt.Sprintf("selected storage '%s'\n", defaultStorage))
			}
			continue
		}

		err := t.ExecuteQuery(defaultStorage, query, func(s string) bool {
			writer.WriteString(fmt.Sprintf("%s\n", s))
			return true
		})

		if err != nil {
			errWriter.WriteString(fmt.Sprintf("error: %v\n", err))
		}
	}

	return nil

}

func (t *implStorageService) ExecuteCommand(cmd string, args []string) (answer string, err error) {

	defer util.PanicToError(&err)

	start := time.Now()

	if cmd == "list" {
		return strings.Join(t.availableStorages, ", "), nil
	}

	if len(args) < 1 {
		return "", errors.New("expected storage name as argument of the command")
	}

	name := args[0]
	args = args[1:]

	s, ok := t.StorageMap[name]
	if !ok {
		return "", errors.Errorf("storage '%s' is not found", name)
	}

	switch strings.ToLower(cmd) {

	case "compact":
		if len(args) < 1 {
			return "", errors.New("compact command needs discardRatio argument")
		}
		discardRatio, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return "", errors.New("second argument 'discardRatio' must be double")
		}
		if err := s.Compact(discardRatio); err != nil {
			t.Log.Error("Compact", zap.Error(err))
			return "", err
		} else {
			t.Log.Info("Compact", zap.Float64("elapsed", time.Since(start).Seconds()))
		}

	case "drop":
		if len(args) < 1 {
			return "", errors.New("drop command needs prefix argument")
		}
		prefix := args[0]
		if !strings.HasSuffix(prefix, ":") {
			return "", errors.New("invalid prefix, must end with ':'")
		}
		if err := s.DropWithPrefix([]byte(prefix)); err != nil {
			t.Log.Error("DropWithPrefix", zap.String("prefix", prefix), zap.Error(err))
			return "", err
		} else {
			t.Log.Info("DropWithPrefix", zap.String("prefix", prefix), zap.Float64("elapsed", time.Since(start).Seconds()))
		}

	case "clean":
		if err := s.DropAll(); err != nil {
			t.Log.Error("DropAll", zap.Error(err))
			return "", err
		} else {
			t.Log.Info("DropAll", zap.Float64("elapsed", time.Since(start).Seconds()))
		}

	case "dump":
		if len(args) < 2 {
			return "", errors.New("dump command needs path argument and timestamp")
		}
		localFilePath := args[0]
		since, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			return "", errors.New("second argument 'since' must be integer")
		}
		dstFile, err := os.OpenFile(localFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, t.BackupFilePerm)
		if err != nil {
			t.Log.Error("BackupCreateFile", zap.String("localFilePath", localFilePath), zap.Error(err))
			return "", err
		}
		if newSince, err := s.Backup(dstFile, since); err != nil {
			t.Log.Error("Backup", zap.String("localFilePath", localFilePath), zap.Uint64("since", since), zap.Error(err))
			return "", err
		} else {
			t.Log.Info("Backup", zap.Float64("elapsed", time.Since(start).Seconds()))
			return fmt.Sprintf("Last: %d", newSince), nil
		}

	case "restore":
		if len(args) < 1 {
			return "", errors.New("restore command needs path argument")
		}
		localFilePath := args[0]
		srcFile, err := os.OpenFile(localFilePath, os.O_RDONLY, t.BackupFilePerm)
		if err != nil {
			t.Log.Error("RestoreOpenFile", zap.String("localFilePath", localFilePath), zap.Error(err))
			return "", err
		}
		if err := s.Restore(srcFile); err != nil {
			t.Log.Error("Restore", zap.String("localFilePath", localFilePath), zap.Error(err))
			return "", err
		} else {
			t.Log.Info("Restore",  zap.String("localFilePath", localFilePath), zap.Float64("elapsed", time.Since(start).Seconds()))
		}

	default:
		return "", errors.Errorf("unknown command '%s'", cmd)
	}

	return "OK", nil

}
