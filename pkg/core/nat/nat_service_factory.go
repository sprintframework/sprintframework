/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package nat

import (
	"github.com/codeallergy/glue"
	"github.com/sprintframework/sprint"
	"github.com/pkg/errors"
	"strings"
	"reflect"
)

type implNatServiceFactory struct {
	Properties   glue.Properties  `inject`
}

func NatServiceFactory() glue.FactoryBean {
	return &implNatServiceFactory{}
}

// Parses a NAT interface description stored in application.nat property.
// The following formats are currently accepted.
// Note that mechanism names are not case-sensitive.
//
//     "" or "none"         return empty NAT
//     "extip:77.12.33.4"   will assume the local machine is reachable on the given IP
//     "any"                uses the first auto-detected mechanism
//     "upnp"               uses the Universal Plug and Play protocol
//     "pmp"                uses NAT-PMP with an auto-detected gateway address
//     "pmp:192.168.0.1"    uses NAT-PMP with the given gateway address
func (t *implNatServiceFactory) Object() (object interface{}, err error) {

	expr := t.Properties.GetString("application.nat", "")

	parts := strings.SplitN(expr, ":", 2)
	switch strings.ToLower(parts[0]) {
	case "", "none", "off", "no":
		return NoNatService(), nil
	case "any", "auto", "on", "yes":
		return autoDiscovery(), nil
	case "extip", "ip", "ext":
		if len(parts) > 1 {
			return ExternalIPService(parts[1])
		} else {
			return nil, errors.Errorf("missing IP address in property application.nat='%s'", expr)
		}
	case "upnp":
		return upnpDiscovery(), nil
	case "pmp", "natpmp", "nat-pmp":
		if len(parts) > 1 {
			return PMPService(parts[1])
		}
		return pmpDiscovery(), nil
	default:
		return nil, errors.Errorf("unknown mechanism %q in property application.nat='%s'", parts[0], expr)
	}
}

func (t *implNatServiceFactory) ObjectType() reflect.Type {
	return sprint.NatServiceClass
}

func (t *implNatServiceFactory) ObjectName() string {
	return "nat_service_factory"
}

func (t *implNatServiceFactory) Singleton() bool {
	return true
}

func autoDiscovery() sprint.NatService {
	return startAutoDiscovery("UPnP or NAT-PMP", func() sprint.NatService {
		found := make(chan sprint.NatService, 2)
		go func() { found <- discoverUPnP() }()
		go func() { found <- discoverPMP() }()
		for i := 0; i < cap(found); i++ {
			if c := <-found; c != nil {
				return c
			}
		}
		return NoNatService()
	})
}

func upnpDiscovery() sprint.NatService {
	return startAutoDiscovery("UPnP", discoverUPnP)
}

func pmpDiscovery() sprint.NatService {
	return startAutoDiscovery("NAT-PMP", discoverPMP)
}

