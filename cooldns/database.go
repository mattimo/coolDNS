package cooldns

import (
	"fmt"
	"net"
)

type MxEntry struct {
	ip       string
	priority int
}

type Entry struct {
	Hostname string
	Ip6s     []net.IP
	Ip4s     []net.IP
	Offline  bool
	Txts     []string
	Mxs      []MxEntry
	Cname    string
}

func (e *Entry) String() string {
	return fmt.Sprintf("%s\n\tIpv6: %v\n\tIpv4: %v\n\tOffline: %v\n\tTxt: %v\n\tMxs: %v\n\tCname: %s",
		e.Hostname, e.Ip6s, e.Ip4s, e.Offline, e.Txts, e.Mxs, e.Cname)
}

// Specifies the four methods that are needed from a DB
// All methods shall be callable from sevferal goroutines at a time.
type CoolDB interface {
	GetEntry(string) *Entry
	SaveEntry(*Entry) error
	GetAuth(string) *Auth
	SaveAuth(*Auth) error

	Close() error
}
