/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
)

type implStatusNode struct {
	Application sprint.Application `inject`
	Context glue.Context `inject`
}

func StatusNode() *implStatusNode {
	return &implStatusNode{}
}

func (t *implStatusNode) Run(args []string) error {

	return sprint.DoWithControlClient(t.Context, func(client sprint.ControlClient) error {
		status, err := client.Status()
		if err == nil {
			println(status)
		}
		return err
	})

}
