package cooldns

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
)

// start a server on a random interface with a db instance and return tcp and udp port
func startDnsServer(db CoolDB, key string) string {
	tcptest, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP: net.ParseIP("0.0.0.0"),
	})
	if err != nil {
		return "0"
	}
	_, tcpPort, _ := net.SplitHostPort(tcptest.Addr().String())
	fmt.Println("tcpPort:", tcpPort)
	tcptest.Close()

	conf := &DnsServerConfig{
		Domain:  "ist.nicht.cool.",
		Listen:  ":" + tcpPort,
		TsigKey: key,
	}
	RunDns(conf, db, nil)
	return tcpPort
}

var dnstestentrylist = []Entry{
	Entry{
		Hostname: "domain.ist.nicht.cool.",
		Ip4s: []net.IP{
			net.ParseIP("1.1.1.1"),
			net.ParseIP("1.1.1.2"),
			net.ParseIP("1.1.1.3"),
		},
		Ip6s: []net.IP{
			net.ParseIP("fe80::92e6:baff:feca:2fc1"),
			net.ParseIP("fe80::92e6:baff:feca:2fc2"),
			net.ParseIP("fe80::92e6:baff:feca:2fc3"),
		},
		Txts: []string{"Hello World", "Cruel World"},
		Mxs: []MxEntry{
			MxEntry{"mail.server.bla.com.", 20},
			MxEntry{"mail1.server.bla.com.", 30},
			MxEntry{"mail2.server.bla.com.", 40},
		},
	},
}

func testDnsReq(port, rrType string, test *Entry) (string, error) {
	cmd := exec.Command("dig", "+short", "@localhost", port, test.Hostname, rrType)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Could not execute command:", err, cmd)
	}
	return fmt.Sprintf("%s", out), nil
}

func splitLines(out string) []string {
	fields := strings.FieldsFunc(out, func(f rune) bool {
		return f == '\n' || f == '\r'
	})
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}
	return fields
}

// Compares arrays to check if all elements in a are also contained in b
func stringArrayCompare(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	var matches int
	for _, e := range a {
		for _, eb := range b {
			if e == eb {
				matches++
				break
			}
		}
	}
	return matches == len(a)
}

func TestStringArrayCompare(t *testing.T) {
	a := []string{"aa", "bb", "ccc", "ddd"}
	b := []string{"aa", "bb", "ccc", "ddd"}
	c := []string{"aa", "bb", "ccc", "zzz"}
	if !stringArrayCompare(a, b) {
		t.Errorf("%v should be equal to %v", a, b)
	}
	if stringArrayCompare(a, c) {
		t.Errorf("%v should NOT be equal to %v", a, c)
	}
}

func TestDnsA(t *testing.T) {
	db, err := getTmpDB()
	if err != nil {
		t.Error("Failed to create temporary DB")
	}
	port := fmt.Sprintf("-p%s", startDnsServer(db, ""))

	for _, test := range dnstestentrylist {
		err := db.SaveEntry(&test)
		if err != nil {
			t.Error("Error saving entry")
		}
		out, err := testDnsReq(port, "A", &test)
		if err != nil {
			t.Error("Failed:", err)
		}
		var ips []string
		for _, ip := range test.Ip4s {
			ips = append(ips, ip.String())
		}
		if !stringArrayCompare(splitLines(out), ips) {
			t.Errorf("Entry does not match response: \n\tTest: %#v\n\tOut:%#v",
				ips,
				splitLines(out),
			)
		}
	}
}

func TestDnsAAAA(t *testing.T) {
	db, err := getTmpDB()
	if err != nil {
		t.Error("Failed to create temporary DB")
	}
	port := fmt.Sprintf("-p%s", startDnsServer(db, ""))

	for _, test := range dnstestentrylist {
		err := db.SaveEntry(&test)
		if err != nil {
			t.Error("Error saving entry")
		}
		out, err := testDnsReq(port, "AAAA", &test)
		if err != nil {
			t.Error("Failed:", err)
		}
		var ips []string
		for _, ip := range test.Ip6s {
			ips = append(ips, ip.String())
		}
		if !stringArrayCompare(splitLines(out), ips) {
			t.Errorf("Entry does not match response: \n\tTest: %#v\n\tOut:%#v",
				ips,
				out,
			)
		}
	}
}

func TestDnsTXT(t *testing.T) {
	db, err := getTmpDB()
	if err != nil {
		t.Error("Failed to create temporary DB")
	}
	port := fmt.Sprintf("-p%s", startDnsServer(db, ""))

	for _, test := range dnstestentrylist {
		err := db.SaveEntry(&test)
		if err != nil {
			t.Error("Error saving entry")
		}
		out, err := testDnsReq(port, "TXT", &test)
		if err != nil {
			t.Error("Failed:", err)
		}
		var txts []string
		for _, t := range test.Txts {
			txts = append(txts, "\""+t+"\"")
		}
		if strings.Trim(out, "\n") != strings.Join(txts, " ") {
			t.Errorf("Entry does not match response: \n\tTest: %#v\n\tOut:%#v",
				strings.Join(txts, " "),
				out,
			)
		}
	}
}

func TestDnsMX(t *testing.T) {
	db, err := getTmpDB()
	if err != nil {
		t.Error("Failed to create temporary DB")
	}
	port := fmt.Sprintf("-p%s", startDnsServer(db, ""))

	for _, test := range dnstestentrylist {
		err := db.SaveEntry(&test)
		if err != nil {
			t.Error("Error saving entry")
		}
		out, err := testDnsReq(port, "MX", &test)
		if err != nil {
			t.Error("Failed:", err)
		}
		var mxs []string
		for _, mx := range test.Mxs {
			mxs = append(mxs, fmt.Sprintf("%d %s", mx.priority, mx.ip))
		}
		if !stringArrayCompare(splitLines(out), mxs) {
			t.Errorf("Entry does not match response: \n\tTest: %v\n\tOut:%s",
				test.Mxs,
				out,
			)
		}
	}
}
