/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"context"
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/server"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"gopkg.in/natefinch/lumberjack.v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func doWithServers(core glue.Context, cb func([]sprint.Server) error) (err error) {

	var contextList []glue.Context

	defer func() {

		var listErr []error
		if r := recover(); r != nil {
			listErr = append(listErr, errors.Errorf("recovered on error: %v", r))
		}

		for _, ctx := range contextList {
			if e := ctx.Close(); e != nil {
				listErr = append(listErr, e)
			}
		}

		if len(listErr) > 0 {
			err = errors.Errorf("%v", listErr)
		}

	}()

	list := core.Bean(sprint.ServerScannerClass, glue.DefaultLevel)
	if len(list) == 0 {
		return errors.New("no one sprint.ServerScanner found in core context")
	}

	for i, s := range list {
		scanner, ok := s.Object().(sprint.ServerScanner)
		if !ok {
			return errors.Errorf("invalid object found for sprint.ServerScanner on position %d in core context", i)
		}
		ctx, err := core.Extend(scanner.ServerBeans()...)
		if err != nil {
			return errors.Errorf("server creation context %v failed by %v", s, err)
		}
		contextList = append(contextList, ctx)
	}

	var serverList []sprint.Server
	for _, ctx := range contextList {

		for i, bean := range ctx.Bean(sprint.ServerClass, glue.DefaultLevel) {
			if srv, ok := bean.Object().(sprint.Server); ok {
				serverList = append(serverList, srv)
			} else {
				return errors.Errorf("invalid object found for sprint.Server on position %d in server context", i)
			}
		}

		for i, bean := range ctx.Bean(sprint.GrpcServerClass, glue.DefaultLevel) {
			if srv, ok := bean.Object().(*grpc.Server); ok {
				s := server.NewGrpcServer(bean.Name(), srv)
				if err := ctx.Inject(s); err != nil {
					return errors.Errorf("injection error for server '%s' of *grpc.Server on position %d in server context, %v", bean.Name(), i, err)
				}
				serverList = append(serverList, s)
			} else {
				return errors.Errorf("invalid object found for *grpc.Server on position %d in server context", i)
			}
		}

		for i, bean := range ctx.Bean(sprint.HttpServerClass, glue.DefaultLevel) {
			if srv, ok := bean.Object().(*http.Server); ok {
				s := server.NewHttpServer(srv)
				if err := ctx.Inject(s); err != nil {
					return errors.Errorf("injection error for server '%s' of *http.Server on position %d in server context, %v", srv.Addr, i, err)
				}
				serverList = append(serverList, s)
			} else {
				return errors.Errorf("invalid object found for *http.Server on position %d in server context", i)
			}
		}

	}

	return cb(serverList)
}

func runServers(application sprint.Application, flags sprint.ApplicationFlags, core glue.Context, log *zap.Logger) error {

	return doWithServers(core, func(servers []sprint.Server) error {

		defer func() {
			if r := recover(); r != nil {
				switch v := r.(type) {
				case error:
					log.Error("Recover", zap.Error(v))
				case string:
					log.Error("Recover", zap.String("error", v))
				default:
					log.Error("Recover", zap.String("error", fmt.Sprintf("%v", v)))
				}
			}
		}()

		if len(servers) == 0 {
			return errors.New("sprint.Server instances are not found in server context")
		}

		c, cancel := context.WithCancel(context.Background())
		defer cancel()

		var boundServers []sprint.Server
		for _, server := range servers {
			if err := server.Bind(); err != nil {
				log.Error("Bind", zap.Error(err))
			} else {
				boundServers = append(boundServers, server)
			}
		}

		cnt := 0
		g, _ := errgroup.WithContext(c)

		for _, server := range boundServers {
			g.Go(server.Serve)
			cnt++
		}
		log.Info("NodeStarted", zap.Int("Servers", cnt), zap.Int("Node", flags.Node()))

		go func() {

			signalCh := make(chan os.Signal, 10)
			signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

			var signal os.Signal

			waitAgain:
			select {
			case signal = <- signalCh:
			case <- application.Done():
				signal = syscall.SIGABRT
			}

			if signal == syscall.SIGHUP {
				list := core.Bean(sprint.LumberjackClass, 1)
				if len(list) > 0 {
					for _, bean := range list {
						if logger, ok := bean.Object().(*lumberjack.Logger); ok {
							logger.Rotate()
						}
					}
					goto waitAgain
				}
				// no lumberjack found, restart application
				application.Shutdown(true)
			}

			log.Info("StopNodeSignal", zap.String("signal", signal.String()))
			total := 0
			for _, server := range boundServers {
				server.Stop()
				total++
			}
			log.Info("NodeStopped", zap.Int("cnt", total), zap.Int("node", flags.Node()))
			log.Sync()
			cancel()

		}()

		return g.Wait()
	})

}

func doInCore(parent glue.Context, withBean interface{}, cb func(core glue.Context) error) error {

	list := parent.Bean(sprint.CoreScannerClass, glue.DefaultLevel)
	if len(list) != 1 {
		return errors.Errorf("expected one core scanner in context, but found %d", len(list))
	}

	core, err := parent.Extend(list[0].Object().(sprint.CoreScanner).CoreBeans()...)
	if err != nil {
		return errors.Errorf("failed to create core context, %v", err)
	}
	defer core.Close()

	err = core.Inject(withBean)
	if err != nil {
		return err
	}

	return cb(core)
}


