package cooldns

import (
	"net"
	"sync"
	)

type DnsDB struct {
	sync.RWMutex
	db	map[string]*Entry
}

var (
	DNSDB	DnsDB
)

type MxEntry struct {
	ip	net.IP
	priority int
}

type Entry struct {
	Hostname	string
	MyIp6		net.IP
	MyIp4		net.IP
	Offline		bool
	Txt		string
	Mx		[]MxEntry
	Cnames		[]string
}

func init() {

	DNSDB = DnsDB{
		db: make(map[string]*Entry),
	}
}

func (d *DnsDB) LoadCache(m map[string]*Entry) {
	d.db = m
}

func (d *DnsDB) Put(e *Entry) {
	d.Lock()
	defer d.Unlock()
	d.db[e.Hostname] = e
}

func (d *DnsDB) Get(name string) *Entry {
	d.RLock()
	defer d.RUnlock()
	return d.db[name]
}
