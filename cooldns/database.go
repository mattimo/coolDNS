package cooldns

import (
	_ "code.google.com/p/gosqlite/sqlite3"
	"database/sql"
	"sync"
	"net"
)

type CoolDB struct {
	sync.Mutex
	c	*sql.DB
}

const createCoolDNS string = `
CREATE TABLE if NOT EXISTS cooldns (
  hostname TEXT,
  myip TEXT,
  myip6 TEXT,
  offline BOOLEAN,
  txt TEXT,
  mx TEXT,
  cnames TEXT,
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
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(createCoolDNS)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(createUsers)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func NewCoolDB(filename string) (*CoolDB, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	err = createTable(db)
	if err != nil {
		return nil, err
	}
	return &CoolDB{c: db}, nil
}

func (db *CoolDB) SaveEntry(e *Entry) error {
	if !e.Offline {
		DNSDB.Put(e)
	}
	db.Lock()
	defer db.Unlock()

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	INSERT OR REPLACE INTO cooldns 
	 (hostname, myip, offline, txt) 
	VALUES (?, ?, ?, ?)
			`,
			e.Hostname,
			e.MyIp4.String(),
			e.Offline,
			e.Txt)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (db *CoolDB) SaveAuth(auth *Auth) error {
	DNSDB.PutUser(auth)
	db.Lock()
	defer db.Unlock()

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	INSERT OR REPLACE INTO users
	 (name, salt, key)
	VALUES (?, ?, ?)
		`,
		auth.Name,
		auth.Salt,
		auth.Key)
	if err != nil {
		return err
	}
	tx.Commit()
	return nil
}

func (db *CoolDB) LoadAll() (map[string]*Entry, error) {
	db.Lock()
	defer db.Unlock()

	// TODO get all the other stuff as well
	rows, err := db.c.Query("SELECT hostname, myip, offline, txt FROM cooldns")
	if err != nil {
		return nil, err
	}
	m := make(map[string]*Entry)
	for rows.Next() {
		e := Entry{}
		var ip4 string
		err = rows.Scan(
			&e.Hostname,
			&ip4,
			&e.Offline,
			&e.Txt)
		if err != nil {
			break
		}
		e.MyIp4 = net.ParseIP(ip4)
		if e.MyIp4 == nil {
			break
		}
		m[e.Hostname] = &e
	}
	return m, err
}

func (db *CoolDB) LoadUsers() (map[string]*Auth, error) {
	db.Lock()
	defer db.Unlock()

	// TODO get all the other stuff as well
	rows, err := db.c.Query("SELECT name, salt, key FROM users")
	if err != nil {
		return nil, err
	}
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
