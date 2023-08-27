/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package dns

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprintframework/pkg/core/dns/netlify"
	"github.com/sprintframework/sprint"
)

type dnsProviderScanner struct {
	Scan     []interface{}
}

func DNSProviderScanner(scan... interface{}) glue.Scanner {
	return &dnsProviderScanner{
		Scan: scan,
	}
}

func (t *dnsProviderScanner) Beans() []interface{} {

	beans := []interface{}{
		netlify.NetlifyProvider(),
		&struct {
			DNSProviders []sprint.DNSProvider `inject`
		}{},
	}

	return append(beans, t.Scan...)
}

