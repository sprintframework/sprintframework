/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package cmd

var DefaultCommands = []interface{}{
	VersionCommand(),
	SetupCommand(),
	HelpCommand(),
	LicensesCommand(),
	OpenAPICommand(),
	ConfigCommand(),
	CertCommand(),
	StopCommand(),
	StatusCommand(),
	RestartCommand(),
	StorageCommand(),
	JobCommand(),
	TokenCommand(),
	RunCommand(),
	StartCommand(),
}
