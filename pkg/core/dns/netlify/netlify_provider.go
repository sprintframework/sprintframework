/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package netlify

import (
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns/netlify"
	"github.com/pkg/errors"
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"os"
	"strings"
)

type implNetlifyProvider struct {
	Properties   glue.Properties  `inject`
}

func NetlifyProvider() sprint.DNSProvider {
	return &implNetlifyProvider{}
}

func (t *implNetlifyProvider) BeanName() string {
	return "netlify_provider"
}

func (t *implNetlifyProvider) Detect(whois *sprint.Whois) bool {
	for _, ns := range whois.NServer {
		if strings.HasSuffix(strings.ToLower(ns), ".nsone.net") {
			return true
		}
	}
	return false
}

func (t *implNetlifyProvider) RegisterChallenge(legoClient interface{}, token string) error {

	client, ok := legoClient.(*lego.Client)
	if !ok {
		return errors.Errorf("expected *lego.Client instance")
	}

	if token == "" {
		token = t.Properties.GetString("netlify.token", "")
	}

	if token == "" {
		token = os.Getenv("NETLIFY_TOKEN")
	}

	if token == "" {
		return errors.New("netlify token not found")
	}

	conf := netlify.NewDefaultConfig()
	conf.Token = token

	prov, err := netlify.NewDNSProviderConfig(conf)
	if err != nil {
		return err
	}

	return client.Challenge.SetDNS01Provider(prov)
}


func (t *implNetlifyProvider) NewClient() (sprint.DNSProviderClient, error) {

	token := t.Properties.GetString("netlify.token", "")

	if token == "" {
		token = os.Getenv("NETLIFY_TOKEN")
	}

	if token == "" {
		return nil, errors.New("netlify.token is empty in config and empty system env NETLIFY_TOKEN")
	}

	return NewClient(token), nil
}