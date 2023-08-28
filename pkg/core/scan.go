/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sealmod"
)

type coreScanner struct {
	Scan     []interface{}
}

func CoreScanner(scan... interface{}) sprint.CoreScanner {
	return &coreScanner {
		Scan: scan,
	}
}

func (t *coreScanner) CoreBeans() []interface{} {

	beans := []interface{}{
		LogFactory(),
		NodeService(),
		ConfigRepository(10000),
		JobService(),
		StorageService(),
		WhoisService(),
		sealmod.Scanner(),
		MailService(),
		&struct {
			ClientScanners []sprint.ClientScanner `inject`
			ServerScanners []sprint.ServerScanner `inject`
		}{},
	}

	return append(beans, t.Scan...)
}

