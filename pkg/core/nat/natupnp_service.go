/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package nat

import (
	"fmt"
	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"github.com/sprintframework/sprint"
	"github.com/sprintframework/sprintframework/pkg/util"
	"net"
	"strings"
	"time"
	"github.com/pkg/errors"
)

const (
	soapRequestTimeout = 3 * time.Second
)

type implUpnpService struct {
	dev         *goupnp.RootDevice
	service     string
	client      upnpClient
	rateLimiter util.RateLimiter
}

func (t *implUpnpService) ServiceName() string {
	return "UPNP " + t.service
}

func (t *implUpnpService) natEnabled() bool {
	var ok bool
	var err error
	t.rateLimiter.Do(func() error {
		_, ok, err = t.client.GetNATRSIPStatus()
		return err
	})
	return err == nil && ok
}

type upnpClient interface {
	GetExternalIPAddress() (string, error)
	AddPortMapping(string, uint16, string, uint16, string, bool, string, uint32) error
	DeletePortMapping(string, uint16, string) error
	GetNATRSIPStatus() (sip bool, nat bool, err error)
}

func (t *implUpnpService) ExternalIP() (addr net.IP, err error) {
	var ipString string
	t.rateLimiter.Do(func() error {
		ipString, err = t.client.GetExternalIPAddress()
		return err
	})

	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(ipString)
	if ip == nil {
		return nil, errors.New("bad IP in response")
	}
	return ip, nil
}

func (t *implUpnpService) AllowMapping() bool {
	return true
}

func (t *implUpnpService) AddMapping(protocol string, extport, intport int, desc string, lifetime time.Duration) error {
	ip, err := t.internalAddress()
	if err != nil {
		return nil // TODO: Shouldn't we return the error?
	}
	protocol = strings.ToUpper(protocol)
	lifetimeS := uint32(lifetime / time.Second)
	t.DeleteMapping(protocol, extport, intport)

	return t.rateLimiter.Do(func() error {
		return t.client.AddPortMapping("", uint16(extport), protocol, uint16(intport), ip.String(), true, desc, lifetimeS)
	})
}

func (t *implUpnpService) internalAddress() (net.IP, error) {
	devaddr, err := net.ResolveUDPAddr("udp4", t.dev.URLBase.Host)
	if err != nil {
		return nil, err
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if x, ok := addr.(*net.IPNet); ok && x.Contains(devaddr.IP) {
				return x.IP, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find local address in same net as %v", devaddr)
}

func (t *implUpnpService) DeleteMapping(protocol string, extport, intport int) error {
	return t.rateLimiter.Do(func() error {
		return t.client.DeletePortMapping("", uint16(extport), strings.ToUpper(protocol))
	})
}

// discoverUPnP searches for Internet Gateway Devices
// and returns the first one it can find on the local network.
func discoverUPnP() sprint.NatService {
	found := make(chan *implUpnpService, 2)
	// IGDv1
	go discover(found, internetgateway1.URN_WANConnectionDevice_1, func(sc goupnp.ServiceClient) *implUpnpService {
		switch sc.Service.ServiceType {
		case internetgateway1.URN_WANIPConnection_1:
			return &implUpnpService{service: "IGDv1-IP1", client: &internetgateway1.WANIPConnection1{ServiceClient: sc}}
		case internetgateway1.URN_WANPPPConnection_1:
			return &implUpnpService{service: "IGDv1-PPP1", client: &internetgateway1.WANPPPConnection1{ServiceClient: sc}}
		}
		return nil
	})
	// IGDv2
	go discover(found, internetgateway2.URN_WANConnectionDevice_2, func(sc goupnp.ServiceClient) *implUpnpService {
		switch sc.Service.ServiceType {
		case internetgateway2.URN_WANIPConnection_1:
			return &implUpnpService{service: "IGDv2-IP1", client: &internetgateway2.WANIPConnection1{ServiceClient: sc}}
		case internetgateway2.URN_WANIPConnection_2:
			return &implUpnpService{service: "IGDv2-IP2", client: &internetgateway2.WANIPConnection2{ServiceClient: sc}}
		case internetgateway2.URN_WANPPPConnection_1:
			return &implUpnpService{service: "IGDv2-PPP1", client: &internetgateway2.WANPPPConnection1{ServiceClient: sc}}
		}
		return nil
	})
	for i := 0; i < cap(found); i++ {
		if c := <-found; c != nil {
			return c
		}
	}
	return nil
}

// finds devices matching the given target and calls matcher for all
// advertised services of each device. The first non-nil service found
// is sent into out. If no service matched, nil is sent.
func discover(out chan<- *implUpnpService, target string, matcher func(goupnp.ServiceClient) *implUpnpService) {
	devs, err := goupnp.DiscoverDevices(target)
	if err != nil {
		out <- nil
		return
	}
	found := false
	for i := 0; i < len(devs) && !found; i++ {
		if devs[i].Root == nil {
			continue
		}
		devs[i].Root.Device.VisitServices(func(service *goupnp.Service) {
			if found {
				return
			}
			// check for a matching IGD service
			sc := goupnp.ServiceClient{
				SOAPClient: service.NewSOAPClient(),
				RootDevice: devs[i].Root,
				Location:   devs[i].Location,
				Service:    service,
			}
			sc.SOAPClient.HTTPClient.Timeout = soapRequestTimeout
			upnp := matcher(sc)
			if upnp == nil {
				return
			}
			upnp.dev = devs[i].Root

			// check whether port mapping is enabled
			if upnp.natEnabled() {
				out <- upnp
				found = true
			}
		})
	}
	if !found {
		out <- nil
	}
}

