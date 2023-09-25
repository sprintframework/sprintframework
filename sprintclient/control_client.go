/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"github.com/sprintframework/sprintpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"sort"
	"strings"
	"sync"
)

type implControlClient struct {
	GrpcConn   *grpc.ClientConn         `inject`
	TlsConfig   *tls.Config             `inject:"optional"`

	client     sprintpb.ControlServiceClient
	closeOnce  sync.Once
}

func ControlClient() sprint.ControlClient {
	return &implControlClient{}
}

func (t *implControlClient) PostConstruct() error {
	t.client = sprintpb.NewControlServiceClient(t.GrpcConn)
	return nil
}

/**
	This control service implControlClient is always exist, therefore it would be an owner of grpcConn object in context
 */
func (t *implControlClient) Destroy() (err error) {
	t.closeOnce.Do(func() {
		if t.GrpcConn != nil {
			err = t.GrpcConn.Close()
		}
	})
	return
}

func (t *implControlClient) wrapError(err error) error {
	var protocol string
	if t.TlsConfig != nil {
		if t.TlsConfig.InsecureSkipVerify {
			protocol = "tls insecure"
		} else {
			protocol = "tls"
		}
	} else {
		protocol = "plain"
	}
	return status.Errorf(codes.Unavailable, "grpc %s invocation '%s', %v", protocol, t.GrpcConn.Target(), err)
}

func (t *implControlClient) Status() (string, error) {

	if resp, err := t.client.Status(context.Background(), new(sprintpb.StatusRequest)); err != nil {
		return "", t.wrapError(err)
	} else {

		var out strings.Builder

		var keys []string
		for k, _ := range resp.Stats {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			out.WriteString(fmt.Sprintf("%s: %s\n", k, resp.Stats[k]))
		}

		return out.String(), nil
	}

}

func (t *implControlClient) Shutdown(restart bool) (string, error) {

	req := new(sprintpb.Command)

	if restart {
		req.Command = "restart"
	} else {
		req.Command = "shutdown"
	}

	if resp, err := t.client.Node(context.Background(), req); err != nil {
		return "", t.wrapError(err)
	} else {
		return resp.Content, nil
	}
}

func (t *implControlClient) ConfigCommand(command string, args []string) (string, error) {

	req := &sprintpb.Command {
		Command: command,
		Args: args,
	}

	if resp, err := t.client.Config(context.Background(), req); err != nil {
		return "", t.wrapError(err)
	} else {
		return resp.Content, nil
	}
}

func (t *implControlClient) CertificateCommand(command string, args []string) (string, error) {

	req := &sprintpb.Command {
		Command: command,
		Args: args,
	}

	if resp, err := t.client.Certificate(context.Background(), req); err != nil {
		return "", t.wrapError(err)
	} else {
		return resp.Content, nil
	}
}

func (t *implControlClient) JobCommand(command string, args []string) (string, error) {

	req := &sprintpb.Command {
		Command: command,
		Args: args,
	}

	if resp, err := t.client.Job(context.Background(), req); err != nil {
		return "", t.wrapError(err)
	} else {
		return resp.Content, nil
	}
}


func (t *implControlClient) StorageCommand(command string, args []string) (string, error) {

	req := &sprintpb.Command {
		Command: command,
		Args: args,
	}

	if resp, err := t.client.Storage(context.Background(), req); err != nil {
		return "", t.wrapError(err)
	} else {
		return resp.Content, nil
	}
}


func (t *implControlClient) StorageConsole(writer io.StringWriter, errWriter io.StringWriter) error {

	stream, err := t.client.StorageConsole(context.Background())
	if err != nil {
		return t.wrapError(err)
	}

	barrier := make(chan int, 1)

	go func() {
		defer func() {
			barrier <- -1
		}()
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				errWriter.WriteString(fmt.Sprintf("error: recv i/o %v\n", err))
				break
			}
			switch resp.Status {
			case 100:
				barrier <- 1
			case 200:
				writer.WriteString(fmt.Sprintf("%s\n", resp.Content))
			default:
				errWriter.WriteString(fmt.Sprintf("error: code %d, %s\n", resp.Status, resp.Content))
			}
		}
	}()

	for {
		query := sprintutils.Prompt("Enter query [exit]: ")
		if query == "" {
			continue
		}
		if query == "exit" {
			break
		}
		request := &sprintpb.StorageConsoleRequest{
			Query: query,
		}
		err = stream.Send(request)
		if err != nil {
			errWriter.WriteString(fmt.Sprintf("error: send i/o %v\n", err))
			break
		}
		if <-barrier == -1 {
			break
		}
	}

	stream.CloseSend()
	return nil
}


