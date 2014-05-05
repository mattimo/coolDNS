package cooldns

import "net"

type MxEntry struct {
	ip	net.IP
	priority int
}

type Entry struct {
	Hostname	string
	MyIp		net.IP
	Offline		bool
	Txt		string
	Mx		[]MxEntry
	Cnames		[]string
}
