package cooldns

import (
	"net"
	"sync"
)

type DnsDB struct {
	sync.RWMutex
	db    map[string]*Entry
	users map[string]*Auth
}

var (
	DNSDB DnsDB
)

type MxEntry struct {
	ip       net.IP
	priority int
}

type Entry struct {
	Hostname string
	MyIp6    net.IP
	MyIp4    net.IP
	Offline  bool
	Txt      string
	Mx       []MxEntry
	Cnames   []string
}

func init() {

	DNSDB = DnsDB{
		db:    make(map[string]*Entry),
		users: make(map[string]*Auth),
	}
}

func (d *DnsDB) LoadCache(m map[string]*Entry, u map[string]*Auth) {
	d.db = m
	d.users = u
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

func (d *DnsDB) PutUser(a *Auth) {
	d.Lock()
	defer d.Unlock()
	d.users[a.Name] = a
}

func (d *DnsDB) GetUser(name string) *Auth {
	d.RLock()
	defer d.RUnlock()
	return d.users[name]
}
