package cooldns

import (
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

// Specifies the four methods that are needed from a DB
// All methods shall be callable from sevferal goroutines at a time.
type CoolDB interface {
	GetEntry(string) *Entry
	SaveEntry(*Entry) error
	GetAuth(string) *Auth
	SaveAuth(*Auth) error

	Close() error
}
