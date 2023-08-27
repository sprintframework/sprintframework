/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package util

import (
	"github.com/go-acme/lego/v4/challenge/dns01"
)

func ToZone(domain string) (string, error) {

	fqdn := ToFqdn(domain)
	zone, err := dns01.FindZoneByFqdn(fqdn)
	if err != nil {
		return "", err
	}

	return UnFqdn(zone), nil
}

// ToFqdn converts the name into a fqdn appending a trailing dot.
func ToFqdn(name string) string {
	n := len(name)
	if n == 0 || name[n-1] == '.' {
		return name
	}
	return name + "."
}

// UnFqdn converts the fqdn into a name removing the trailing dot.
func UnFqdn(name string) string {
	n := len(name)
	if n != 0 && name[n-1] == '.' {
		return name[:n-1]
	}
	return name
}
