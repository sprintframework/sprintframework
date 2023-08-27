/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package core

import (
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/sprintframework/sprintpb"
	"github.com/sprintframework/sprint"
	"go.uber.org/zap"
	"strings"
	"github.com/pkg/errors"
	"fmt"
)

type implDynDNSService struct {

	Log           *zap.Logger              `inject`

	CertificateRepository sprint.CertificateRepository  `inject`
	DNSProviders          map[string]sprint.DNSProvider `inject`
	NatService            sprint.NatService             `inject`

	providerMap   map[string]sprint.DNSProvider // key is the provider name, not bean_name
	providerList  []string

}

func DynDNSService() sprint.DynDNSService {
	return &implDynDNSService{
		providerMap: make(map[string]sprint.DNSProvider),
	}
}

func (t *implDynDNSService) BeanName() string {
	return "dyn_dns_service"
}

func (t *implDynDNSService) PostConstruct() (err error) {

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
			t.Log.Error("PostConstruct", zap.Error(err))
		}
	}()

	for beanName, prov := range t.DNSProviders {
		name := beanName
		if strings.HasSuffix(name, "_provider") {
			name = name[:len(name) - len("_provider")]
		}
		t.providerMap[name] = prov
		t.providerList = append(t.providerList, name)
	}

	return nil
}

func (t *implDynDNSService) EnsureAllPublic(subDomains ...string) error {

	return t.EnsureCustom(func(client sprint.DNSProviderClient, zone string, externalIP string) error {
		return t.doEnsureAllPublic(client, zone, externalIP, subDomains)
	})

}

func (t *implDynDNSService) EnsureCustom(cb func(client sprint.DNSProviderClient, zone string, externalIP string) error) error {

	var list []*sprintpb.Zone

	err := t.CertificateRepository.ListZones("", func(entry *sprintpb.Zone) bool {
		if entry.DnsProvider != "" {
			list = append(list, entry)
		}
		return true
	})
	if err != nil {
		return err
	}

	serviceName := t.NatService.ServiceName()
	t.Log.Info("DynDNS", zap.String("nat", serviceName))
	
	var externalIP string
	if t.NatService.AllowMapping() {
		extIP, err := t.NatService.ExternalIP()
		if err != nil {
			t.Log.Error("NatExternalIP", zap.Error(err))
		} else {
			externalIP = extIP.String()
		}
	}

	var listErr []error

	for _, entry := range list {
		prov, ok := t.providerMap[entry.DnsProvider]
		if !ok {
			listErr = append(listErr, errors.Errorf( "dns provider '%s' not found for zone '%s'", entry.DnsProvider, entry.Zone))
			continue
		}

		client, err := prov.NewClient()
		if err != nil {
			listErr = append(listErr, errors.Wrapf(err, "init dns provider '%s';", entry.DnsProvider))
			continue
		}

		if externalIP == "" {
			externalIP, err = client.GetPublicIP()
			if err != nil {
				listErr = append(listErr, errors.Wrapf(err, "get public ip form dns provider '%s';", entry.DnsProvider))
				continue
			}
		}

		err = cb(client, entry.Zone, externalIP)
		if err != nil {
			listErr = append(listErr, errors.Wrapf(err, "ensure provider '%s';", entry.DnsProvider))
		}

	}

	if len(listErr) > 0 {
		return errors.Errorf("errors %+v", listErr)
	}

	return nil
}

func (t *implDynDNSService) doEnsureAllPublic(client sprint.DNSProviderClient, zone string, externalIP string, subDomains []string) error {

	zone = dns01.UnFqdn(zone)

	list, err := client.GetRecords(zone)
	if err != nil {
		return err
	}

	cache := make(map[string]bool)
	for _, subDomain := range subDomains {
		if subDomain == "" {
			cache[zone] = true
		} else {
			cache[fmt.Sprintf("%s.%s", subDomain, zone)] = true
		}
	}

	var listErr []error

	for _, record := range list {
		if record.Type == "A" {
			host := strings.ToLower(record.Hostname)
			if cache[host] && record.Value != externalIP {

				if err = client.RemoveRecord(zone, record.ID); err == nil {
					record.Value = externalIP
					record, err = client.CreateRecord(zone, record)
				}

				if err != nil {
					listErr = append(listErr, errors.Wrapf(err, "recreate record for zone '%s' with id '%s' name '%s' and type '%s' value '%s';", zone, record.ID, record.Hostname, record.Type, externalIP))
				}

			}
		}
	}

	if len(listErr) > 0 {
		return errors.Errorf("errors %+v", listErr)
	}

	return nil
}