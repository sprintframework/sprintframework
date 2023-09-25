/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintcmd

var DefaultCommands = []interface{}{
	VersionCommand(),
	SetupCommand(),
	HelpCommand(),
	ResourcesCommand(),
	ConfigCommand(),
	CertsCommand(),
	StorageCommand(),
	JobsCommand(),
	KeygenCommand(),
	NodeCommand(),
	RunNode(),
	StartNode(),
	StopNode(),
	RestartNode(),
	StatusNode(),
}
