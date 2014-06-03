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
	"testing"
)

type domainValidationTest struct {
	InDomain, OutDomain string
	Validates           bool
}

var domainvalidationtests = []domainValidationTest{
	domainValidationTest{"hello", "hello.domain.name.", true},
	domainValidationTest{"hello.domain.name", "hello.domain.name.", true},
	domainValidationTest{"hello.", "hello.domain.name.", true},
	domainValidationTest{"hello.hello.domain.name.", "hello.hello.domain.name.", true},
	domainValidationTest{"hello.domain.name", "hello.domain.name.", true},
	domainValidationTest{"h", "h.domain.name.", false},
	domainValidationTest{"g.hello.", "g.hello.domain.name.", true},
	domainValidationTest{"hello..", "hello.domain.name.", true},
}

func TestDomainValidation(t *testing.T) {
	const domain = "domain.name."
	for _, test := range domainvalidationtests {
		fqdn, validates := ValidateDomain(test.InDomain, domain)
		if test.Validates {
			if !validates {
				t.Error("Should Validate")
			}
			if fqdn != test.OutDomain || !validates {
				t.Errorf("Should match: In: %#v, Out=%#v, Expected=%#v, Validates=%#v",
					test.InDomain,
					fqdn,
					test.OutDomain,
					validates)
				continue
			}
		} else {
			if validates {
				t.Error("Should not Validate")
			}
		}
		t.Logf("Compared: In: %#v, Out=%#v, Expected=%#v, Validates=%#v",
			test.InDomain,
			fqdn,
			test.OutDomain,
			validates)
	}
}
