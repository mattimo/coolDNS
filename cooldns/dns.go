package cooldns

import (
	"github.com/miekg/dns"
	"log"
	"time"
	"os"
)

var TsigKey string

func init() {
	TsigKey = os.Getenv("COOLDNS_TSIG_KEY")
}

// Hold a pointer to the actual DnsDB within the CoolDB object
type DnsHandler struct {
	db *DnsDB
}

func (h *DnsHandler) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	defer func(w dns.ResponseWriter, m *dns.Msg) {
		err := w.WriteMsg(m)
		if err != nil {
			log.Println("WOOPS ERRRROOOORR:", err)
		}
	}(w, m)
	if TsigKey != "" {
		m.SetTsig(domainsuffix, dns.HmacSHA256, 300, time.Now().Unix())
	}

	for _, question := range r.Question {
		entry := h.db.Get(question.Name)
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

func serve(net, name, secret string) {
	switch name {
	case "":
		server := &dns.Server{Pool: false, Addr: ":8053", Net: net, TsigSecret: nil}
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("Failed to setup the "+net+" server: %s\n", err.Error())
		}
	default:
		server := &dns.Server{Pool: false, Addr: ":8053", Net: net, TsigSecret: map[string]string{name: secret}}
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("Failed to setup the "+net+" server: %s\n", err.Error())
		}
	}
}

func RunDns(db *DnsDB) {
	h := &DnsHandler{db : db}
	dns.HandleFunc(domainsuffix, h.handleRequest)
	go serve("udp", domainsuffix, TsigKey )
	go serve("tcp", domainsuffix, TsigKey)
}
