/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package netlify_test

import (
	"fmt"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/stretchr/testify/require"
	"github.com/sprintframework/sprintframework/pkg/core/dns/netlify"
	"github.com/sprintframework/sprint"
	"os"
	"strings"
	"testing"
)

func TestGetZone(t *testing.T) {

	domain := "www.example.com."

	fqdn := dns01.ToFqdn(domain)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	require.NoError(t, err)

	secondLevel := dns01.UnFqdn(zone)
	println(secondLevel)

}

func noTestNetlifyMxChange(t *testing.T) {

	domain := "example.com"

	token := os.Getenv("NETLIFY_TOKEN")
	require.True(t, token != "")

	client := netlify.NewClient(token)

	publicIP, err := client.GetPublicIP()
	require.NoError(t, err)

	println(publicIP)

	fqdn := dns01.ToFqdn(domain)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	require.NoError(t, err)

	fmt.Printf("zone=%v\n", zone)

	zone = dns01.UnFqdn(zone)

	list, err := client.GetRecords(zone)
	require.NoError(t, err)

	mxHostname := fmt.Sprintf("mx.%s", zone)

	createARecord := true
	createMXRecord := true
	for _, rec := range list {
		deleteRecord := false

		switch rec.Type {
		case "A":
			if strings.EqualFold(rec.Hostname, mxHostname) {
				if rec.Value == publicIP {
					createARecord = false
				} else {
					deleteRecord = true
				}
			}

		case "MX":
			if strings.EqualFold(rec.Value, mxHostname) {
				createMXRecord = false
			} else {
				deleteRecord = true
			}
		}

		if deleteRecord {
			fmt.Printf("DeleteRecord %v\n", rec)
			err = client.RemoveRecord(zone, rec.ID)
			require.NoError(t, err)
		}
	}

	if createARecord {

		record := &sprint.DNSRecord{
			Hostname: "mx",
			TTL:      300,
			Type:     "A",
			Value:    publicIP,
		}

		record, err = client.CreateRecord(zone, record)
		require.NoError(t, err)

		fmt.Printf("Created Record %v\n", record)
	}

	if createMXRecord {

		record := &sprint.DNSRecord{
			Hostname: zone,
			TTL:      300,
			Type:     "MX",
			Priority:  10,
			Value:    mxHostname,
		}

		record, err = client.CreateRecord(zone, record)
		require.NoError(t, err)

		fmt.Printf("Created Record %v\n", record)
	}


}


