package cooldns

import (
	"bytes"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func createTestServer(t *testing.T) (*httptest.Server, *bytes.Buffer, CoolDB) {
	f, err := getTmpFile()
	if err != nil {
		t.Error("setup Failed: Could not create test server")
	}

	// start Server TODO: make stopable
	db, err := getDB(f)
	if err != nil {
		t.Error("Failed to create temporary DB:", err)
	}
	// Setup logger to log to our buffer, print on failure later
	logBuf := new(bytes.Buffer)
	log.SetOutput(logBuf)
	log.Println("##### TESTING LOG BUFFER #####")
	t.Log("New database:", f)

	handler := SetupWeb(db, "../assets", "../templates", NewDummyMetrics())
	return httptest.NewServer(handler), logBuf, db
}

func TestGetIndex(t *testing.T) {
	server, logBuf, _ := createTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Log(logBuf.String())
		t.Error("Failed to connect to server")
	}
	if resp.StatusCode != 200 {
		t.Log(logBuf.String())
		t.Error("Status for \"/\" is not 200")
	}
}

type updateTest struct {
	Domain   string
	Password string
	Ip       string
	Txt      string
	F        bool
}

var updatetests = []updateTest{
	updateTest{
		"hallo",
		"123456789",
		"192.168.0.1",
		"Hallo Welt",
		false,
	},
	updateTest{
		"tadaaa",
		"blablablabla",
		"1.1.1.1",
		"Blab Bla srdgojfg kfjghn",
		false,
	},
	updateTest{
		"lalala",
		"sdgkasgkomasfdkog",
		"6.6.6.6",
		"sgasg awgdasd asdfg dfasg sdfgdf",
		false,
	},
	updateTest{
		"lalalaergsdfkln",
		"sdgkasgkomasfdkog",
		"6.6.6.256",
		"sgasg awgdasd asdfg dfasg sdfgdf",
		true,
	},
}

func TestUpdateDynApi(t *testing.T) {
	server, logBuf, db := createTestServer(t)
	defer server.Close()
	for _, test := range updatetests {
		auth, err := NewAuth(test.Domain+".ist.nicht.cool.", test.Password)
		if err != nil {
			t.Error("Creating new user failed")
		}
		if db.SaveAuth(auth) != nil {
			t.Error("Saving New User failed")
		}
	}

	// Perform updates
	for _, test := range updatetests {
		domain := test.Domain + ".ist.nicht.cool."
		URL, err := url.Parse(server.URL)
		if err != nil {
			t.Error("Error parsing Server URL:", err)
		}
		URL.User = url.UserPassword(domain, test.Password)
		URL.Path = "/nic/update"
		v := url.Values{}
		v.Set("hostname", domain)
		v.Set("myip", test.Ip)
		v.Set("txt", test.Txt)
		URL.RawQuery = v.Encode()
		resp, err := http.Get(URL.String())
		if err != nil || resp.StatusCode != 200 {
			if !test.F {
				t.Log(logBuf.String())
				t.Error("Failed to update URL:", URL.String(), err)
				return
			}
		}
		if test.F && resp.StatusCode != 400 {
			t.Error("This test should have failed", test)
		}

	}

	for _, test := range updatetests {
		domain := test.Domain + ".ist.nicht.cool."
		e := db.GetEntry(domain)
		if e == nil {
			if test.F {
				continue
			}
			t.Log(logBuf.String())
			t.Error("Domain does not exist in DB", test)
			break
		}
		if e.Ip4s[0] == nil || e.Ip4s[0].String() != net.ParseIP(test.Ip).String() {
			t.Log(logBuf.String())
			t.Error("Ips dont match", e, test)
		}
		if e.Txts[0] == "" || e.Txts[0] != test.Txt {
			t.Log(logBuf.String())
			t.Error("Txts dont match", e, test)
		}
	}
}
