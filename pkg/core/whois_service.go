/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/likexian/whois"
	"github.com/sprintframework/sprint"
	"go.uber.org/zap"
	"strings"
)

type implWhoisService struct {
	Log          *zap.Logger           `inject`
}

func WhoisService() sprint.WhoisService {
	return &implWhoisService{}
}

func (t *implWhoisService) Whois(domain string) (string, error) {
	return whois.Whois(domain)
}

func (t *implWhoisService) Parse(result string) *sprint.Whois {

	resp := new(sprint.Whois)

	lines := strings.Split(result, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) <= 1 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		switch key {
		case "domain", "domain name":
			resp.Domain = value
		case "nserver", "name server":
			if strings.HasSuffix(value, ".") {
				value = value[:len(value)-1]
			}
			resp.NServer = append(resp.NServer, value)
		case "state", "registrant country":
			resp.State = value
		case "person", "registrant name":
			resp.Person = value
		case "e-mail", "registrant email":
			resp.Email = value
		case "registrar":
			resp.Registrar = value
		case "created", "creation date":
			resp.Created = value
		case "paid-till", "registrar registration expiration date", "registry expiry date":
			resp.PaidTill = value
		default:
			if strings.HasSuffix(key, "expiration date") || strings.HasSuffix(key, "expiry date") {
				resp.PaidTill = value
			}
		}

	}

	return resp
}

