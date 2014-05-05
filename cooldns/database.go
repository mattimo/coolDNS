package cooldns

import (
	_ "code.google.com/p/gosqlite/sqlite3"
	"database/sql"
	"sync"
)

type CoolDB struct {
	sync.Mutex
	c	*sql.DB
}

const createString string = `
CREATE TABLE if NOT EXISTS cooldns (
  hostname TEXT,
  myip TEXT,
  offline BOOLEAN,
  txt TEXT,
  mx TEXT,
  cnames TEXT
)
`

func createTable(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(createString)
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
	DNSDB.Put(e)
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
			e.MyIp.String(),
			e.Offline,
			e.Txt)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
