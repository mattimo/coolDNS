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
