/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/codeallergy/glue"
)

type coreScanner struct {
	scan []interface{}
}

func CoreScanner(scan... interface{}) glue.Scanner {
	return &coreScanner {
		scan: scan,
	}
}

func (t *coreScanner) Beans() []interface{} {

	beans := []interface{}{
		ZapLogFactory(),
		HCLogFactory(),
		NodeService(),
		ConfigRepository(10000),
		JobService(),
		StorageService(),
		MailService(),
		&struct {
			ChildContexts []glue.ChildContext `inject`
		}{},
	}

	return append(beans, t.scan...)
}

