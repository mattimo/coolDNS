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
	_ "code.google.com/p/gosqlite/sqlite3"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

type SqliteCoolDB struct {
	sync.Mutex
	c     *sql.DB
	cache *DnsDB
}

const createCoolDNS string = `
CREATE TABLE if NOT EXISTS cooldns (
  hostname TEXT,
  ip4 TEXT,
  ip6 TEXT,
  offline BOOLEAN,
  txt TEXT,
  mx TEXT,
  cname TEXT,
UNIQUE (hostname) ON CONFLICT REPLACE
);
`
const createUsers string = `
CREATE TABLE if NOT EXISTS users (
  name TEXT,
  salt VARCHAR(8),
  key VARCHAR(32),
UNIQUE (name) ON CONFLICT REPLACE
) 
`

func createTable(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(createCoolDNS)
	if err != nil {
		return err
	}
	_, err = tx.Exec(createUsers)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func NewSqliteCoolDB(filename string) (*SqliteCoolDB, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	err = createTable(db)
	if err != nil {
		return nil, err
	}
	cooldb := &SqliteCoolDB{c: db}

	// create and load Cache
	cache := NewCache()
	dnsCache, err := cooldb.loadAll()
	if err != nil {
		log.Fatal("Error Loading DNS Cache:", err)
	}
	userCache, err := cooldb.loadUsers()
	if err != nil {
		log.Fatal("Error Loading User Cache:", err)
	}
	cache.LoadCache(dnsCache, userCache)

	cooldb.cache = cache
	return cooldb, nil
}

func (db *SqliteCoolDB) Close() error {
	db.Lock()
	defer db.Unlock()
	return db.c.Close()
}

const dbRecSep = "\x1f"

func (db *SqliteCoolDB) SaveEntry(e *Entry) error {
	if !e.Offline {
		db.cache.Put(e)
	}
	db.Lock()
	defer db.Unlock()

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	// Check if values exist otherwise fill them with blanks
	var (
		hostname string
		cname    string
		ip4s     string
		ip6s     string
		offline  bool
		txts     string
		mxs      string
	)

	hostname = e.Hostname

	cname = e.Cname

	var ip4a []string
	for _, ip4 := range e.Ip4s {
		ip4a = append(ip4a, ip4.String())
	}
	ip4s = strings.Join(ip4a, dbRecSep)

	var ip6a []string
	for _, ip6 := range e.Ip6s {
		ip6a = append(ip6a, ip6.String())
	}
	ip6s = strings.Join(ip6a, dbRecSep)

	txts = strings.Join(e.Txts, dbRecSep)
	offline = e.Offline

	var mxa []string
	for _, mx := range e.Mxs {
		mxa = append(mxa, fmt.Sprintf("%d %s", mx.priority, mx.ip))
	}
	mxs = strings.Join(mxa, dbRecSep)

	// TODO remove logging
	// log.Printf("hostname: %s, cname :%s, ip4: %s, ip6: %s, txts: %s, mxs: %v", hostname, cname, ip4, ip6, txts, mxs)

	_, err = tx.Exec(`
	INSERT OR REPLACE INTO cooldns 
	 (hostname, cname, ip4, ip6, offline, mx, txt)
	VALUES (?, ?, ?, ?, ?, ?, ?);
			`,
		hostname,
		cname,
		ip4s,
		ip6s,
		offline,
		mxs,
		txts)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (db *SqliteCoolDB) SaveAuth(auth *Auth) error {
	db.cache.PutUser(auth)
	db.Lock()
	defer db.Unlock()

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
	INSERT OR REPLACE INTO users
	 (name, salt, key)
	VALUES (?, ?, ?);
		`,
		auth.Name,
		auth.Salt,
		auth.Key)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (db *SqliteCoolDB) loadAll() (map[string]*Entry, error) {
	db.Lock()
	defer db.Unlock()

	rows, err := db.c.Query("SELECT hostname, cname, ip4, ip6, offline, mx, txt FROM cooldns")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]*Entry)
	for rows.Next() {
		e := Entry{}
		var (
			ip4s string
			ip6s string
			txts string
			mxs  string
		)
		err = rows.Scan(
			&e.Hostname,
			&e.Cname,
			&ip4s,
			&ip6s,
			&e.Offline,
			&mxs,
			&txts)
		if err != nil {
			break
		}
		// unmarshal ip6 address
		for _, ip4 := range strings.Split(ip4s, dbRecSep) {
			ip := net.ParseIP(ip4)
			if ip == nil {
				continue
			}
			e.Ip4s = append(e.Ip4s, ip)
		}
		// unmarshal ip6 addresses
		for _, ip6 := range strings.Split(ip6s, dbRecSep) {
			ip := net.ParseIP(ip6)
			if ip == nil {
				continue
			}
			e.Ip6s = append(e.Ip6s, ip)
		}

		e.Txts = strings.Split(txts, dbRecSep)
		// unmarshal MX entries
		for _, mx := range strings.Split(mxs, dbRecSep) {
			mxSubA := strings.Fields(mx)
			if len(mxSubA) != 2 {
				continue
			}
			prio, err := strconv.ParseInt(mxSubA[0], 10, 0)
			if err != nil {
				log.Println("Warning: loadAll: Malformatted mx entry in database:", mxSubA)
				continue
			}
			e.Mxs = append(e.Mxs, MxEntry{
				ip:       mxSubA[1],
				priority: int(prio),
			})
		}

		m[e.Hostname] = &e
	}
	return m, err
}

func (db *SqliteCoolDB) loadUsers() (map[string]*Auth, error) {
	db.Lock()
	defer db.Unlock()

	rows, err := db.c.Query("SELECT name, salt, key FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	u := make(map[string]*Auth)
	for rows.Next() {
		a := Auth{}
		err = rows.Scan(
			&a.Name,
			&a.Salt,
			&a.Key)
		if err != nil {
			break
		}
		u[a.Name] = &a
	}
	return u, err

}

func (db *SqliteCoolDB) GetAuth(name string) *Auth {
	return db.cache.GetUser(name)
}

func (db *SqliteCoolDB) GetEntry(name string) *Entry {
	return db.cache.Get(name)
}
