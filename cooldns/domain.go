// The CoolDNS Project. The simple dynamic dns server and update service.
// Copyright (C) 2014 The CoolDNS Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.package main

package cooldns

import (
	"github.com/miekg/dns"
	"strings"
)

// Validates a sub domain, if validation fails we try to make it a valid sub
// fqdn of the subdomain. The following things are checked and mitigated:
//  - Sub-domain is actually a sub-domain if not we try to append the domain
//  - multiple dots are deleted
//  - if not an fqdn a dot is appended
//  - Everything is lowercased
// If the URL has one of the following features we just return false and do
// not recover
//  - URL to short (below 2 characters)
//  - contains illeageal characters.
func ValidateDomain(subdomain, domain string) (fqdn string, valid bool) {
	domain = strings.ToLower(domain)
	// To lower case
	subdomain = strings.ToLower(subdomain)
	// get rid of double dots
	subdomain = trimDots(subdomain)
	// make and fqdn out of it
	subdomain = dns.Fqdn(subdomain)

	if !dns.IsSubDomain(domain, subdomain) {
		subdomain = subdomain + domain
	}
	if !dns.IsSubDomain(domain, subdomain) {
		return "", false
	}

	// Check for domain length constraint is met (length greater then 2)
	subLabels := dns.SplitDomainName(subdomain)
	domainCount := dns.CountLabel(domain)
	if len(strings.Join(subLabels[0:len(subLabels)-domainCount], "")) < 2 {
		return "", false
	}

	_, isDomain := dns.IsDomainName(subdomain)
	return subdomain, isDomain
}

func trimDots(h string) string {
	for {
		hnew := strings.Replace(h, "..", ".", -1)
		if h == hnew {
			return hnew
		}
		h = hnew
	}
}
