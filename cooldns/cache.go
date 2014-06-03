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
	"sync"
)

type DnsDB struct {
	sync.RWMutex
	db    map[string]*Entry
	users map[string]*Auth
}

func NewCache() *DnsDB {
	return &DnsDB{
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
