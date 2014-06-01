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
