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

type webTestServer struct {
	S     *httptest.Server
	Log   *bytes.Buffer
	Db    CoolDB
	TmpDb string
}

func createTestServer(t *testing.T) *webTestServer {
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
	t.Log("Database:", f)

	handler := SetupWeb(db, "../assets", "../templates", NewDummyMetrics())
	return &webTestServer{httptest.NewServer(handler), logBuf, db, f}
}

func TestGetIndex(t *testing.T) {
	server := createTestServer(t)
	defer server.S.Close()

	resp, err := http.Get(server.S.URL + "/")
	if err != nil {
		t.Log(server.Log.String())
		t.Error("Failed to connect to server")
	}
	if resp.StatusCode != 200 {
		t.Log(server.Log.String())
		t.Error("Status for \"/\" is not 200")
	}
}

type updateTest struct {
	Domain   string
	Password string
	Ip       string
	Txt      string
	F        bool
	V6       bool
}

var updatetests = []updateTest{
	updateTest{
		"hallo",
		"123456789",
		"192.168.0.1",
		"Hallo Welt",
		false,
		false,
	},
	updateTest{
		"tadaaa",
		"blablablabla",
		"1.1.1.1",
		"Blab Bla srdgojfg kfjghn",
		false,
		false,
	},
	updateTest{
		"lalala",
		"sdgkasgkomasfdkog",
		"6.6.6.6",
		"sgasg awgdasd asdfg dfasg sdfgdf",
		false,
		false,
	},
	updateTest{
		"lalalaergsdfkln",
		"sdgkasgkomasfdkog",
		"6.6.6.256",
		"sgasg awgdasd asdfg dfasg sdfgdf",
		true,
		false,
	},
	updateTest{
		"lalalaipv6",
		"sdgkasgkomasfdkog",
		"fe80::92e6:baff:feca:2fc1",
		"sgasg awgdasd asdfg dfasg sdfgdf",
		false,
		true,
	},
}

func getUpdateURL(domain, password, server string, v url.Values) *url.URL {
	URL, err := url.Parse(server)
	if err != nil {
		return nil
	}
	URL.User = url.UserPassword(domain, password)
	URL.Path = "/nic/update"
	URL.RawQuery = v.Encode()

	return URL
}

func TestUpdateDynApi(t *testing.T) {
	server := createTestServer(t)
	defer server.S.Close()
	for _, test := range updatetests {
		auth, err := NewAuth(test.Domain+".ist.nicht.cool.", test.Password)
		if err != nil {
			t.Fatal("Creating new user failed")
		}
		if server.Db.SaveAuth(auth) != nil {
			t.Fatal("Saving New User failed")
		}

		// Setup URL
		domain := test.Domain + ".ist.nicht.cool."
		v := url.Values{}
		v.Set("hostname", domain)
		v.Set("myip", test.Ip)
		v.Set("txt", test.Txt)
		URL := getUpdateURL(domain, test.Password, server.S.URL, v)

		resp, err := http.Get(URL.String())
		if err != nil || resp.StatusCode != 200 {
			if !test.F {
				t.Log(server.Log.String())
				t.Fatal("Failed to update URL:", URL.String(), err)
				return
			}
		}
		if test.F && resp.StatusCode != 400 {
			t.Log(server.Log.String())
			t.Fatal("This test should have failed", test)
		}

		// Check in db if values weere actually set.
		e := server.Db.GetEntry(domain)
		if e == nil {
			if test.F {
				continue
			}
			t.Log(server.Log.String())
			t.Fatal("Domain does not exist in DB", test)
			break
		}
		if test.V6 {
			if e.Ip6s[0] == nil || e.Ip6s[0].String() != net.ParseIP(test.Ip).String() {
				t.Log(server.Log.String())
				t.Fatal("Ips dont match", e, test)
			}
		} else {
			if e.Ip4s[0] == nil || e.Ip4s[0].String() != net.ParseIP(test.Ip).String() {
				t.Log(server.Log.String())
				t.Fatal("Ips dont match", e, test)
			}
		}

		if e.Txts[0] == "" || e.Txts[0] != test.Txt {
			t.Log(server.Log.String())
			t.Fatal("Txts dont match", e, test)
		}
	}
}

type updateErrorFieldsTest struct {
	Domain string
	Ip     string
	Txt    string
	Status int
}

var updateerrorfieldtests = []updateErrorFieldsTest{
	// All fields Empty
	updateErrorFieldsTest{"", "", "", 401},
	updateErrorFieldsTest{"testtest.ist.nicht.cool.", "", "", 400},
	updateErrorFieldsTest{"", "192.168.0.1", "", 401},
	updateErrorFieldsTest{"", "", "Hallo Welt", 401},
	updateErrorFieldsTest{"testtest.ist.nicht.cool.", "", "Hallo Welt", 400},
	updateErrorFieldsTest{"", "192.168.0.1", "Hallo Welt", 401},
	// Not an IP
	updateErrorFieldsTest{"testtest.ist.nicht.cool.", "192.168.0.bla1", "", 400},
}

type updateErrorAuthTest struct {
	Fields *updateErrorFieldsTest
	Pass   string
	User   string
}

var updatecorrectfields = updateErrorFieldsTest{"testtest.ist.nicht.cool.", "192.168.0.1", "Hallo Welt", 401}

var updateerrorauthtests = []updateErrorAuthTest{
	// No User name
	updateErrorAuthTest{&updatecorrectfields, ".", "123456789"},
	// No password
	updateErrorAuthTest{&updatecorrectfields, "testtest.ist.nicht.cool.", ""},
	// Neither password nor user name
	updateErrorAuthTest{&updatecorrectfields, "", ""},
	// Wrong Username
	updateErrorAuthTest{&updatecorrectfields, "testtest.jsagjcool.", "123456789"},
	// Wrong Password
	updateErrorAuthTest{&updatecorrectfields, "testtest.ist.nicht.cool.", "wrong wrong"},
}

func TestUpdateDynApiError(t *testing.T) {
	server := createTestServer(t)
	defer server.S.Close()
	// Setup User
	domain := "testtest.ist.nicht.cool."
	password := "123456789"
	auth, err := NewAuth(domain, password)
	if err != nil {
		t.Fatal("Creating new user failed")
	}
	if server.Db.SaveAuth(auth) != nil {
		t.Fatal("Saving New User failed")
	}

	// Check for malformatted fields
	for _, test := range updateerrorfieldtests {
		v := url.Values{}
		v.Set("hostname", test.Domain)
		v.Set("myip", test.Ip)
		v.Set("txt", test.Txt)
		URL := getUpdateURL(domain, password, server.S.URL, v)

		resp, err := http.Get(URL.String())
		if err != nil {
			t.Log(server.Log.String())
			t.Fatal("Failed to update URL:", URL.String(), err)
		}
		if resp.StatusCode != test.Status {
			t.Log(server.Log.String())
			t.Errorf("FieldCheck: Wrong return Code: Got %d, expected %d. \n\tTest: %v",
				resp.StatusCode,
				test.Status,
				test)
		}
	}

	// Check for invalid auth
	for _, test := range updateerrorauthtests {
		v := url.Values{}
		v.Set("hostname", test.Fields.Domain)
		v.Set("myip", test.Fields.Ip)
		v.Set("txt", test.Fields.Txt)
		URL := getUpdateURL(test.User, test.Pass, server.S.URL, v)

		resp, err := http.Get(URL.String())
		if err != nil {
			t.Log(server.Log.String())
			t.Fatal("Failed to update URL:", URL.String(), err)
		}
		if resp.StatusCode != 401 {
			t.Log(server.Log.String())
			t.Errorf("AuthCheck: Wrong return Code: Got %d, expected 401. \n\tTest: %v",
				resp.StatusCode,
				test)
		}
	}

}
