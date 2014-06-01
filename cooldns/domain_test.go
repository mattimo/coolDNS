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
