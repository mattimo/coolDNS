package cooldns

import (
	"testing"
	"io/ioutil"
	"io"
	"net"
	"os"
	"fmt"
	"strings"
	"reflect"
)

func getTmpFile() (string, error) {
	f, err := ioutil.TempFile("/tmp", "cooldnstest")
	if err != nil {
		return "", err
	}
	defer f.Close()

	return f.Name(), nil
}

func getDB(filename string) (CoolDB, error) {
	return NewSqliteCoolDB(filename)
}

func getTmpDB() (CoolDB, error) {
	tmpFileName, err := getTmpFile()
	if err != nil {
		return nil, err
	}
	return getDB(tmpFileName)
}

var entries = []*Entry{
		&Entry{Hostname: "test1.ist.nicht.cool.",
			Ip4s: []net.IP{net.ParseIP("192.168.0.1")},
			Offline: false,
			Txts: []string{"Hallo Welt", "Zweite Zeile"},
			Mxs: []MxEntry{
				MxEntry{"mail.deine.mutter.de", 1000},
			},
		},
		&Entry{Hostname: "test2.ist.nicht.cool.",
			Ip4s: []net.IP{net.ParseIP("192.168.0.2")},
			Offline: false,
			Txts: []string{"Hallo Welt", "Zweite Zeile"},
			Mxs: []MxEntry{
				MxEntry{"mail.deine.mutter.de", 1000},
			},
		},
}

func TestCacheCreation(t *testing.T) {
	db, err := getTmpDB()
	if err != nil {
		t.Error("Failed to create temporary DB:", err)
	}
	defer db.Close()

	for _, e := range entries {
		err = db.SaveEntry(e)
		if err != nil {
			t.Error("failed to save:", e)
		}
	}

	for _, e := range entries {
		dbE := db.GetEntry(e.Hostname)
		if dbE == nil {
			t.Error("Could not find DB entry:", e.Hostname)
		}
		if !reflect.DeepEqual(e, dbE) {
			t.Error("database Entry and randomly generated did not match")
		}
	}

}

var randomSrc io.Reader

func getRandom() io.Reader {
	if randomSrc == nil {
		var err error
		randomSrc, err = os.Open("/dev/urandom")
		if err != nil {
			return nil
		}
	}
	return randomSrc
}

func getRandIP(b []byte) net.IP {
	var ipA []string
	var ip string
	if len(b) == 4 {
		for _, b4 := range b {
			ipA = append(ipA, fmt.Sprintf("%d", b4))
		}
		ip = strings.Join(ipA, ".")
	} else if len(b) == 16 {
		for _, b6 := range b {
			ipA = append(ipA, fmt.Sprintf("%x", b6))
		}
		ipPA := make([]string, 8)
		for i, _ := range ipPA {
			ipPA[i] = ipA[2*i]+ipA[2*1+1]
		}
		ip = strings.Join(ipPA, ":")
	}
	return net.ParseIP(ip)
}

func genRandEntry() *Entry {
	rand := getRandom()
	// random hostname
	hostnameRand := make([]byte, 32)
	rand.Read(hostnameRand)
	// random cname
	cnameRand := make([]byte, 16)
	rand.Read(cnameRand)
	// random ipv4
	ipv4Rand := make([]byte, 4)
	rand.Read(ipv4Rand)
	// random ipv6
	ipv6Rand := make([]byte, 16)
	rand.Read(ipv6Rand)

	return &Entry{
		Hostname: fmt.Sprintf("%x.ist.nicht.cool.", hostnameRand),
		Cname: fmt.Sprintf("%x", cnameRand),
		Ip4s: []net.IP{getRandIP(ipv4Rand)},
		Ip6s: []net.IP{getRandIP(ipv6Rand)},
		Offline: false,
		Txts: []string{"Hallo Welt", "Zweite Zeile"},
		Mxs: []MxEntry{
			MxEntry{"mail.deine.mutter.de", 1000},
		},
	}

}

const randEntryC = 128

func TestCacheRandomEntry(t *testing.T) {
	db, err := getTmpDB()
	if err != nil {
		t.Error("Failed to create temporary DB:", err)
	}
	defer db.Close()

	var randEntries []*Entry
	for i:=0; i<randEntryC; i++ {
		randEntries = append(randEntries, genRandEntry())
	}

	for _, e := range randEntries {
		err := db.SaveEntry(e)
		if err != nil {
			t.Errorf("Failed to insert: %v", e)
		}
	}

	for _, e := range randEntries {
		dbE := db.GetEntry(e.Hostname)
		if dbE == nil {
			t.Error("Could not find entry in cache:", e.Hostname)
		}
		if !reflect.DeepEqual(e, dbE) {
			t.Error("cache Entry and randomly generated did not match")
		}
	}

}


func TestDatabaseRandomEntry(t *testing.T) {
	tmpFile, err := getTmpFile()
	if err != nil {
		t.Error("Failed to create tmp file")
	}
	db, err := getDB(tmpFile)
	if err != nil {
		t.Error("Failed to create temporary DB:", err)
	}

	var randEntries []*Entry
	for i:=0; i<randEntryC; i++ {
		randEntries = append(randEntries, genRandEntry())
	}

	for _, e := range randEntries {
		err := db.SaveEntry(e)
		if err != nil {
			t.Errorf("Failed to insert: %v", e)
		}
	}
	// Close database
	db.Close()

	rdb, err := getDB(tmpFile)
	if err != nil {
		t.Error("Failed to reopen temporary DB:", err)
	}

	for _, e := range randEntries {
		dbE := rdb.GetEntry(e.Hostname)
		if dbE == nil {
			t.Error("Could not find entry in DB:", e.Hostname)
		}
		if !reflect.DeepEqual(e, dbE) {
			t.Logf("\n%v\n%v\n", e, dbE)
			t.Error("database Entry and randomly generated did not match")
		}
	}
}
