/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintserver

import (
	"context"
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/pkg/errors"
	"github.com/sprintframework/cert"
	"github.com/sprintframework/nat"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/app"
	"github.com/sprintframework/sprintframework/sprintutils"
	"github.com/sprintframework/sprintpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/**
	ControlServer Impl
*/

var (
	ShutdownDelay = time.Second

	ErrAuthWrongRole     = errors.New("wrong role")
	ErrAuthAdminRequired = errors.New("admin role required")
	ErrAuthUserNotFound  = errors.New("user not found")
)

var (
	ErrInterrupted = errors.New("interrupted")
	ErrTimeout     = errors.New("timeout")
)

type implGrpcControlServer struct {
	sprintpb.UnimplementedControlServiceServer

	GrpcServer     *grpc.Server `inject:"bean=control-grpc-server"`
	GatewayServer  *http.Server `inject:"bean=control-gateway-server,optional"`

	Components  []sprint.Component   `inject:"optional,level=-1"`

	Application         sprint.Application      `inject`
	Properties          glue.Properties         `inject`

	AuthorizationMiddleware    sprint.AuthorizationMiddleware `inject`

	Log                   *zap.Logger                  `inject`
	NodeService           sprint.NodeService           `inject`
	JobService            sprint.JobService            `inject`
	StorageService        sprint.StorageService        `inject`
	ConfigRepository      sprint.ConfigRepository      `inject`
	CertificateService    cert.CertificateService    `inject:"optional"`
	CertificateManager    cert.CertificateManager    `inject:"optional"`

	NatService    nat.NatService  `inject:"optional"`

	startTime   time.Time
}

func ControlServer() sprint.Component {
	srv := &implGrpcControlServer{
		startTime:      time.Now(),
	}
	return srv
}

func (t *implGrpcControlServer) PostConstruct() (err error) {

	defer sprintutils.PanicToError(&err)

	sprintpb.RegisterControlServiceServer(t.GrpcServer, t)
	reflection.Register(t.GrpcServer)

	if t.GatewayServer != nil {
		api, err := sprintutils.FindGatewayHandler(t.GatewayServer, "/api/")
		if err != nil {
			return err
		}
		sprintpb.RegisterControlServiceHandlerServer(context.Background(), api, t)
	}

	return nil
}

func (t *implGrpcControlServer) BeanName() string {
	return "control_server"
}

func (t *implGrpcControlServer) GetStats(cb func(name, value string) bool) error {
	cb("start", t.startTime.String())
	if t.NatService != nil {
		serviceName := t.NatService.ServiceName()
		cb("nat", serviceName)
		if serviceName != "no_nat" {
			extIP, err := t.NatService.ExternalIP()
			if err != nil {
				cb("nat.err", err.Error())
			} else {
				cb("nat.extip", extIP.String())
			}
		}
	}
	return nil
}

func (t *implGrpcControlServer) Status(ctx context.Context, request *sprintpb.StatusRequest) (resp *sprintpb.StatusResponse, err error) {

	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = errors.Errorf("%v", v)
			}
		}
	}()

	resp = &sprintpb.StatusResponse{Stats: make(map[string]string)}

	for _, component := range t.Components {
		component.GetStats(func(name, value string) bool {
			key := fmt.Sprintf("%s.%s", component.BeanName(), name)
			resp.Stats[key] = value
			return true
		})
	}

	return resp, err
}

func (t *implGrpcControlServer) Node(ctx context.Context, req *sprintpb.Command) (resp *sprintpb.CommandResult, err error) {

	defer func() {
		if r := recover(); r != nil {
			switch v := r.(type) {
			case error:
				err = v
			case string:
				err = errors.New(v)
			default:
				err = errors.Errorf("%v", v)
			}
		}
	}()

	user, ok := t.AuthorizationMiddleware.GetUser(ctx)
	if !ok {
		return nil, ErrAuthUserNotFound
	}

	if user.Roles == nil || !user.Roles["ADMIN"] {
		return nil, ErrAuthWrongRole
	}

	username := user.Username


	restart := false
	switch req.Command {
	case "restart":
		restart = true
	case "shutdown":
	default:
		return nil, errors.Errorf("unknown command '%s'", req.Command)
	}

	t.Log.Info("ShutdownSignal", zap.Bool("restart", restart), zap.String("username", username))

	time.AfterFunc(ShutdownDelay, func() {
		t.Log.Info("ApplicationShutdown", zap.Bool("restart", restart))
		t.Application.Shutdown(restart)
	})
	return &sprintpb.CommandResult{
		Content: "OK",
	}, nil
}

func (t *implGrpcControlServer) Config(ctx context.Context, req *sprintpb.Command) (resp *sprintpb.CommandResult, err error) {

	defer sprintutils.PanicToError(&err)

	user, ok := t.AuthorizationMiddleware.GetUser(ctx)
	if !ok {
		return nil, ErrAuthUserNotFound
	}

	if user.Roles == nil || !user.Roles["ADMIN"] {
		return nil, ErrAuthWrongRole
	}
	username := user.Username


	switch req.Command {
	case "get":
		return t.configGet(req.Args)
	case "set":
		return t.configSet(req.Args, username)
	case "dump":
		return t.configDump(req.Args)
	case "list":
		return t.configList(req.Args)
	default:
		return nil, errors.Errorf("unknown command '%s'", req.Command)
	}

}

func (t *implGrpcControlServer) configGet(args []string) (resp *sprintpb.CommandResult, err error) {

	if len(args) < 1 {
		return nil, errors.New("config get command needs key argument")
	}

	key := args[0]

	value, err := t.ConfigRepository.Get(key)
	if err != nil {
		return nil, errors.Errorf("get config entry by key '%s', %v", key, err)
	}

	if app.IsHiddenProperty(key) {
		value = "******"
	}

	return &sprintpb.CommandResult{Content: value}, nil
}

func (t *implGrpcControlServer) configSet(args []string, username string) (resp *sprintpb.CommandResult, err error) {

	if len(args) < 2 {
		return nil, errors.New("config set command needs key and value arguments")
	}

	key := args[0]
	value := args[1]

	if err := t.ConfigRepository.Set(key, value); err != nil {
		return nil, errors.Errorf("set config entry by key '%s', %v", key, err)
	}

	t.Log.Info("ConfigSet", zap.String("key", key), zap.String("user", username), zap.Bool("emptyValue", value == ""))

	return &sprintpb.CommandResult{Content: "OK"}, nil
}

func (t *implGrpcControlServer) configDump(args []string) (resp *sprintpb.CommandResult, err error) {

	var prefix string
	if len(args) > 0 {
		prefix = args[0]
	}

	var out strings.Builder

	err = t.ConfigRepository.EnumerateAll(prefix, func(key, value string) bool {
		if app.IsHiddenProperty(key) {
			value = "******"
		}
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(value)
		out.WriteByte('\n')
		return true
	})

	return &sprintpb.CommandResult{Content: out.String()}, err

}

func (t *implGrpcControlServer) configList(args []string) (resp *sprintpb.CommandResult, err error) {

	var prefix string
	if len(args) > 0 {
		prefix = args[0]
		args = args[1:]
	}

	limit := 80
	if len(args) > 0 {
		limit, err = strconv.Atoi(args[0])
		if err != nil {
			return nil, errors.Errorf("parsing limit '%s', %v", args[0], err)
		}
	}

	var out strings.Builder

	err = t.ConfigRepository.EnumerateAll(prefix, func(key, value string) bool {
		if app.IsHiddenProperty(key) {
			value = "******"
		}
		if len(value) > limit {
			value = value[:limit] + "..."
			value = strings.ReplaceAll(value, "\n", " ")
		}
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(value)
		out.WriteByte('\n')
		return true
	})

	return &sprintpb.CommandResult{Content: out.String()}, err

}

func (t *implGrpcControlServer) Certificate(ctx context.Context, req *sprintpb.Command) (resp *sprintpb.CommandResult, err error) {

	defer sprintutils.PanicToError(&err)

	user, ok := t.AuthorizationMiddleware.GetUser(ctx)
	if !ok {
		return nil, ErrAuthUserNotFound
	}

	if user.Roles == nil || !user.Roles["ADMIN"] {
		return nil, ErrAuthWrongRole
	}

	if req.Command == "manager" {
		if t.CertificateManager != nil {
			content, err := t.CertificateManager.ExecuteCommand(req.Command, req.Args)
			if err != nil {
				return nil, err
			}
			return &sprintpb.CommandResult{Content: content}, nil
		} else {
			return &sprintpb.CommandResult{Content: "Error: certificate manager not found in context"}, nil
		}

	} else {
		if t.CertificateService != nil {
			content, err := t.CertificateService.ExecuteCommand(req.Command, req.Args)
			if err != nil {
				return nil, err
			}
			return &sprintpb.CommandResult{Content: content}, nil
		} else {
			return &sprintpb.CommandResult{Content: "Error: certificate service not found in context"}, nil
		}

	}

}

func (t *implGrpcControlServer) Job(ctx context.Context, req *sprintpb.Command) (resp *sprintpb.CommandResult, err error) {

	defer sprintutils.PanicToError(&err)

	if !t.AuthorizationMiddleware.HasUserRole(ctx, "ADMIN") {
			return nil, ErrAuthAdminRequired
		}

	content, err := t.JobService.ExecuteCommand(req.Command, req.Args)
	if err != nil {
		return nil, err
	}

	return &sprintpb.CommandResult{Content: content}, nil

}

func (t *implGrpcControlServer) Storage(ctx context.Context, req *sprintpb.Command) (resp *sprintpb.CommandResult, err error) {

	defer sprintutils.PanicToError(&err)

	if !t.AuthorizationMiddleware.HasUserRole(ctx, "ADMIN") {
		return nil, ErrAuthAdminRequired
	}

	content, err := t.StorageService.ExecuteCommand(req.Command, req.Args)
	if err != nil {
		return nil, err
	}
	
	return &sprintpb.CommandResult{
		Content: content,
	}, nil

}

func (t *implGrpcControlServer) StorageConsole(stream sprintpb.ControlService_StorageConsoleServer) (err error) {

	defer sprintutils.PanicToError(&err)

	if !t.AuthorizationMiddleware.HasUserRole(stream.Context(), "ADMIN") {
		return ErrAuthAdminRequired
	}

	err = t.StorageService.Console(stream)
	if err != nil {
		t.Log.Error("StorageConsole",
			zap.Error(err))
	}
	return err
}
