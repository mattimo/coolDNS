package cooldns

import (
	"github.com/miekg/dns"
	"net"
	"strconv"
	"log"
)

const dom string = "ist.nicht.cool."

func handleReflect(w dns.ResponseWriter, r *dns.Msg) {
	var (
		v4 bool
		rr dns.RR
		str string
		a net.IP
	)

	m := new(dns.Msg)
	m.SetReply(r)
	defer func (w dns.ResponseWriter, m *dns.Msg){
		err := w.WriteMsg(m)
		if err != nil {
			log.Println("WOOPS ERRRROOOORR:", err)
		}
	}(w, m)


	entry := DNSDB.Get(r.Question[0].Name)
	if entry == nil {
		return
	}

	// DO QUERY STUFF HERE
	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		str = "Port: " + strconv.Itoa(ip.Port) + " (udp)"
		a = ip.IP
		v4 = a.To4() != nil
	}
	if ip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		str = "Port: " + strconv.Itoa(ip.Port) + " (tcp)"
		a = ip.IP
		v4 = a.To4() != nil
	}

	if v4 {
		rr = new(dns.A)
		rr.(*dns.A).Hdr = dns.RR_Header{Name: entry.Hostname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}
		ip4 := entry.MyIp4
		if ip4 == nil {
			return
		}
		rr.(*dns.A).A = ip4
	} else {
		rr = new(dns.AAAA)
		rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: entry.Hostname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 0}
		ip6 := entry.MyIp6
		if ip6 == nil {
			return
		}
		rr.(*dns.AAAA).AAAA = entry.MyIp6
	}

	t := new(dns.TXT)
	t.Hdr = dns.RR_Header{Name: dom, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 0}
	t.Txt = []string{str}


	switch r.Question[0].Qtype {
	case dns.TypeAAAA, dns.TypeA:
		m.Answer = append(m.Answer, rr)
		m.Extra = append(m.Extra, t)
	case dns.TypeTXT:
		m.Answer = append(m.Answer, t)
		m.Extra = append(m.Extra, rr)
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

func RunDns() {
	dns.HandleFunc("ist.nicht.cool.", handleReflect)
	go serve("udp", "", "")
	go serve("tcp", "", "")
}
