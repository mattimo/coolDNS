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
	"code.google.com/p/go.net/idna"
	"github.com/miekg/dns"
	"log"
	"time"
)

// Configuration for the DNS server
type DnsServerConfig struct {
	Domain string // fqdn of the full Domain name
	Listen string // Dns server listener Interface <interface>:<port>. default is ":8053"
	// Tsig Key according to the tsig spec (base64 string)
	// If not set, tsig will not be activated.
	TsigKey string
}

// Hold a pointer to the actual DnsDB within the CoolDB object
type dnsHandler struct {
	db                      CoolDB
	domain, listen, tsigkey string
	// Metrics Handle
	metric MetricsHandle
}

func (h *dnsHandler) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	h.metric.DnsEvent()
	m := new(dns.Msg)
	m.SetReply(r)
	defer func(w dns.ResponseWriter, m *dns.Msg) {
		err := w.WriteMsg(m)
		if err != nil {
			log.Println("WOOPS ERRRROOOORR:", err)
		}
	}(w, m)
	if h.tsigkey != "" {
		m.SetTsig(h.domain, dns.HmacSHA256, 300, time.Now().Unix())
	}

	for _, question := range r.Question {
		var entry *Entry
		// try to convert to puny code
		qName, err := idna.ToUnicode(question.Name)
		if err == nil {
			entry = h.db.GetEntry(qName)
		} else {
			entry = h.db.GetEntry(question.Name)
		}
		if entry == nil {
			return
		}
		// if CNAME exists use it and return. Do not resolve alias
		// address
		if entry.Cname != "" {
			cname := new(dns.CNAME)
			cname.Hdr = dns.RR_Header{Name: entry.Hostname,
				Rrtype: dns.TypeCNAME,
				Class:  dns.ClassINET,
				Ttl:    0}
			cname.Target = entry.Cname
			m.Answer = append(m.Answer, cname)
			return
		}

		switch question.Qtype {
		case dns.TypeAAAA:
			for _, ip6 := range entry.Ip6s {
				rr := new(dns.AAAA)
				rr.Hdr = dns.RR_Header{Name: entry.Hostname,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    0}
				rr.AAAA = ip6
				m.Answer = append(m.Answer, rr)
			}
		case dns.TypeA:
			for _, ip4 := range entry.Ip4s {
				rr := new(dns.A)
				rr.Hdr = dns.RR_Header{Name: entry.Hostname,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    0}
				rr.A = ip4
				m.Answer = append(m.Answer, rr)
			}
		case dns.TypeTXT:
			t := new(dns.TXT)
			t.Hdr = dns.RR_Header{Name: entry.Hostname,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    0}
			if len(entry.Txts) == 0 {
				break
			}
			t.Txt = entry.Txts
			m.Answer = append(m.Answer, t)
		case dns.TypeMX:
			for _, emx := range entry.Mxs {
				mx := new(dns.MX)
				mx.Hdr = dns.RR_Header{Name: entry.Hostname,
					Rrtype: dns.TypeMX,
					Class:  dns.ClassINET,
					Ttl:    0}
				mx.Mx = emx.ip
				mx.Preference = uint16(emx.priority)
				m.Answer = append(m.Answer, mx)
			}
		}
	}
	return

}

// Takes Either tcp or udp string
func (h *dnsHandler) serve(net string) {
	server := &dns.Server{Pool: false,
		Addr: h.listen,
		Net:  net,
		TsigSecret: map[string]string{
			h.domain: h.tsigkey,
		},
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to setup the "+net+" server: %s\n", err)
	}
}

// Run DNS Server with given config.
//
// A configuration is sufficient if it contains a Domain name, a db mustbe
// supplied but the metricsHandle can be nil
func RunDns(config *DnsServerConfig, db CoolDB, metric MetricsHandle) {
	h := new(dnsHandler)
	if db == nil {
		log.Fatal("No database supplied")
	}
	h.db = db

	if metric != nil {
		h.metric = metric
	} else {
		h.metric = NewDummyMetrics()
	}

	if config.Domain == "" {
		log.Fatal("No Domain supplied")
	}
	h.domain = config.Domain

	if config.Listen != "" {
		h.listen = config.Listen
	} else {
		h.listen = ":8053"
	}

	h.tsigkey = config.TsigKey

	dns.HandleFunc(config.Domain, h.handleRequest)
	go h.serve("udp")
	go h.serve("tcp")
}
