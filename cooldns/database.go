package cooldns

import (
	_ "code.google.com/p/gosqlite/sqlite3"
	"database/sql"
	"net"
	"sync"
	"fmt"
	"log"
	"strings"
	"strconv"
)

type CoolDB struct {
	sync.Mutex
	c *sql.DB
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
	defer func(){
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

func NewCoolDB(filename string) (*CoolDB, error) {
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
	return &CoolDB{c: db}, nil
}

func (db *CoolDB) Close() error {
	db.Lock()
	defer db.Unlock()
	return db.c.Close()
}

const dbRecSep = "\x1f"

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
	defer func(){
		if err != nil {
			tx.Rollback()
		}
	}()
	// Check if values exist otherwise fill them with blanks
	var (
		hostname string
		cname string
		ip4s string
		ip6s string
		offline bool
		txts string
		mxs string
	)

	hostname = e.Hostname

	cname = e.Cname

	var ip4a []string
	for _, ip4 := range e.Ip4s {
		ip4a =  append(ip4a, ip4.String())
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
		mxa = append(mxa, fmt.Sprintf("%i %s", mx.priority, mx.ip))
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

func (db *CoolDB) SaveAuth(auth *Auth) error {
	DNSDB.PutUser(auth)
	db.Lock()
	defer db.Unlock()

	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer func(){
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

func (db *CoolDB) LoadAll() (map[string]*Entry, error) {
	db.Lock()
	defer db.Unlock()

	// TODO get all the other stuff as well
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
			mxs string
		)
		err = rows.Scan(
			&e.Hostname,
			&e.Cname,
			&ip4s,
			&ip6s,
			&e.Offline,
			&mxs,
			&txts,)
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
				log.Println("Warning: LoadAll: Malformatted mx entry in database")
				continue
			}
			e.Mxs = append(e.Mxs, MxEntry{
						ip: mxSubA[1],
						priority: int(prio),
					})
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
