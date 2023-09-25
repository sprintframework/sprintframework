/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcmd

import (
	"encoding/base64"
	"fmt"
	"github.com/codeallergy/glue"
	"github.com/pkg/errors"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/sprintutils"
	"strconv"
	"strings"
	"time"
)

type implKeygenCommand struct {
	Application      sprint.Application      `inject`
	ApplicationFlags sprint.ApplicationFlags `inject`
	Properties       glue.Properties      `inject`
}

func KeygenCommand() sprint.Command {
	return &implKeygenCommand{}
}

func (t *implKeygenCommand) BeanName() string {
	return "keygen"
}

func (t *implKeygenCommand) Help() string {
	helpText := `
Usage: ./%s keygen [command]

	Issues token and keys using on startup or in rpc calls.

Commands:

  boot                      Generates the bootstrap token using for decryption of configs and databases.

  auth                      Generates JWT authorization token using in gRPC and HTTPS.

  verify                    Verify the JWT token and decodes arguments.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, t.Application.Executable()))
}

func (t *implKeygenCommand) Synopsis() string {
	return "keygen commands [boot, auth, verify]"
}

func (t *implKeygenCommand) Run(args []string) (err error) {

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

	if len(args) == 0 {
		return errors.Errorf("keygen command needs argument: %s", t.Synopsis())
	}
	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "boot":
		return t.generateBootstrapToken(args)
	case "auth":
		return t.generateAuthToken(args)
	case "verify":
		return t.verifyAuthToken(args)
	default:
		return errors.Errorf("unknown sub-command '%s' for token command", cmd)
	}
}

func (t *implKeygenCommand) generateBootstrapToken(args []string) error {
	if token, err := sprintutils.GenerateToken(); err == nil {
		println(token)
		return nil
	} else {
		return err
	}
}

func (t *implKeygenCommand) generateAuthToken(args []string) error {

	if len(args) < 4 {
		return errors.Errorf("Usage: ./%s keygen auth username roles context ttl-in-days", t.Application.Executable())
	}

	username := args[0]
	roles := args[1]
	context := args[2]
	ttlDaysStr := args[3]

	contextMap := make(map[string]string)
	pairs := strings.Split(context, ",")
	for _, pair := range pairs {
		i := strings.IndexByte(pair, '=')
		if i == -1 {
			contextMap[pair] = ""
		} else {
			contextMap[pair[0:i]] = pair[i+1:]
		}
	}

	secret := sprintutils.PromptPassword("Enter JWT secret key: ")
	secretKey, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		return err
	}

	ttlDays, err := strconv.ParseInt(ttlDaysStr, 10, 64)
	if err != nil {
		return errors.Errorf("error on parsing days '%s', %v", ttlDaysStr, err)
	}

	indexedRoles := make(map[string]bool)
	for _, role := range strings.Split(roles, ",") {
		indexedRoles[strings.TrimSpace(role)] = true
	}

	user := &sprint.AuthorizedUser{
		Username:  username,
		Roles:     indexedRoles,
		Context:   contextMap,
		ExpiresAt: time.Now().Unix() + ttlDays*24*3600,
	}

	token, err := sprintutils.GenerateAuthToken(secretKey, user)
	if err != nil {
		return err
	}

	println(token)
	return nil
}

func (t *implKeygenCommand) verifyAuthToken(args []string) error {

	var authToken string
	if len(args) > 0 {
		authToken = args[0]
	} else {
		tokenKey := strings.ToUpper(fmt.Sprintf("%s_auth", t.Application.Name()))
		authToken = t.Properties.GetString(tokenKey, "")
	}

	if authToken == "" {
		return errors.New("auth token not found")
	}

	secret := sprintutils.PromptPassword("Enter JWT secret key: ")
	secretKey, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		return err
	}

	user, err := sprintutils.VerifyAuthToken(secretKey, authToken)
	if err != nil {
		errors.Errorf("verify error, %v", err)
	}

	fmt.Printf("%s, %+v, %s, expires at %s\n", user.Username, user.Roles, user.Context, time.Unix(user.ExpiresAt, 0).String())
	return nil
}
