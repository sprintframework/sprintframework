/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

var DefaultCommands = []interface{}{
	VersionCommand(),
	SetupCommand(),
	HelpCommand(),
	ResourcesCommand(),
	ConfigCommand(),
	CertsCommand(),
	StopCommand(),
	StatusCommand(),
	RestartCommand(),
	StorageCommand(),
	JobsCommand(),
	KeygenCommand(),
	RunCommand(),
	StartCommand(),
}
