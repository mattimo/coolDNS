package cooldns

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestGetUpdate(t *testing.T) {
	server := createTestServer(t)
	defer server.S.Close()

	resp, err := http.Get(server.S.URL + "/update")
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
		"hallo.ist.nicht.cool.",
		"123456789",
		"192.168.0.1",
		"Hallo Welt",
		false,
		false,
	},
	updateTest{
		"tadaaa.ist.nicht.cool.",
		"blablablabla",
		"1.1.1.1",
		"Blab Bla srdgojfg kfjghn",
		false,
		false,
	},
	updateTest{
		"lalala.ist.nicht.cool.",
		"sdgkasgkomasfdkog",
		"6.6.6.6",
		"sgasg awgdasd asdfg dfasg sdfgdf",
		false,
		false,
	},
	updateTest{
		"lalalaergsdfkln.ist.nicht.cool.",
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
		auth, err := NewAuth(test.Domain, test.Password)
		if err != nil {
			t.Fatal("Creating new user failed")
		}
		if server.Db.SaveAuth(auth) != nil {
			t.Fatal("Saving New User failed")
		}

		// Setup URL
		domain := test.Domain
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
	updateErrorFieldsTest{"testtest.ist.nicht.cool.noexist", "192.168.0.1", "", 401},
}

type updateErrorAuthTest struct {
	Fields *updateErrorFieldsTest
	User   string
	Pass   string
}

var updatecorrectfields = updateErrorFieldsTest{"testtest.ist.nicht.cool.", "192.168.0.1", "Hallo Welt", 401}

var updatenotexistfields = updateErrorFieldsTest{"testtest.ist.nicht.cool.not.exist.", "192.168.0.1", "Hallo Welt", 401}

var updateerrorauthtests = []updateErrorAuthTest{
	// No User name
	updateErrorAuthTest{&updatecorrectfields, "", "123456789"},
	// No password
	updateErrorAuthTest{&updatecorrectfields, "testtest.ist.nicht.cool.", ""},
	// Neither password nor user name
	updateErrorAuthTest{&updatecorrectfields, "", ""},
	// Wrong Username
	updateErrorAuthTest{&updatecorrectfields, "testtest.jsagjcool.", "123456789"},
	// Wrong Password
	updateErrorAuthTest{&updatecorrectfields, "testtest.ist.nicht.cool.", "wrongwrong"},
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
		if test.Domain != "" {
			v.Set("hostname", test.Domain)
		}
		if test.Ip != "" {
			v.Set("myip", test.Ip)
		}
		if test.Txt != "" {
			v.Set("txt", test.Txt)
		}
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

	// check for no auth
	v := url.Values{}
	v.Set("hostname", domain)
	v.Set("myip", "1.1.1.1")
	v.Set("txt", "Hallololololol")
	URL := getUpdateURL(domain, password, server.S.URL, v)
	URL.User = nil

	resp, err := http.Get(URL.String())
	if err != nil {
		t.Log(server.Log.String())
		t.Fatal("Failed to update URL:", URL.String(), err)
	}
	if resp.StatusCode != 401 {
		t.Log(server.Log.String())
		t.Errorf("AuthCheck: Wrong return Code: Got %d, expected 401. \n\tTest: %v",
			resp.StatusCode,
			"Check for no auth")
	}

	// check totaly not existing entry
	v = url.Values{}
	v.Set("hostname", updatenotexistfields.Domain)
	v.Set("myip", updatenotexistfields.Ip)
	v.Set("txt", updatenotexistfields.Txt)
	URL = getUpdateURL(updatenotexistfields.Domain, password, server.S.URL, v)
	resp, err = http.Get(URL.String())
	if err != nil {
		t.Log(server.Log.String())
		t.Fatal("Failed to update URL:", URL.String(), err)
	}
	if resp.StatusCode != 401 {
		t.Log(server.Log.String())
		t.Errorf("AuthCheck: Wrong return Code: Got %d, expected 401. \n\tTest: %v",
			resp.StatusCode,
			"Check for not existing auth")
	}
}

func getFormNewURL(server string) *url.URL {
	URL, err := url.Parse(server)
	if err != nil {
		return nil
	}
	URL.Path = "/"

	return URL
}

type formdomainnewtest struct {
	Domain   string
	Secret   string
	ErrCount int
}

var formdomainnewtests = []formdomainnewtest{
	formdomainnewtest{"test1.ist.nicht.cool", "GeheimGeheim", 0},
	formdomainnewtest{"withdot.ist.nicht.cool.", "GeheimGeheim", 0},
	formdomainnewtest{"test2.ist.nicht.cool", "toshort", 1},
	formdomainnewtest{"", "", 2},
	formdomainnewtest{"1.ist.nicht.cool", "", 2},
	formdomainnewtest{"", "strangeöäü?", 1},
	formdomainnewtest{"2.ist.nicht.cool", "123", 1},
	formdomainnewtest{"3.ist.nicht.cool", "strangeöäü?", 1},
	formdomainnewtest{"exists.ist.nicht.cool", "blablablabla", 1},
}

func TestFormDomainNew(t *testing.T) {
	server := createTestServer(t)
	defer server.S.Close()
	// Add Exists entry
	domain := "exists.ist.nicht.cool."
	secret := "supersecretsecret"
	auth, _ := NewAuth(domain, secret)
	server.Db.SaveAuth(auth)
	server.Db.SaveEntry(&Entry{
		Hostname: domain,
		Offline:  false,
	})

	for _, test := range formdomainnewtests {
		v := url.Values{}
		if test.Domain != "" {
			v.Set("domain", test.Domain)
		}
		if test.Secret != "" {
			v.Set("secret", test.Secret)
		}
		URL := getFormNewURL(server.S.URL)

		resp, err := http.PostForm(URL.String(), v)
		if err != nil {
			t.Log(server.Log.String())
			t.Fatal("Failed to update URL:", URL.String(), err)
		}
		if resp.StatusCode != 200 {
			t.Log(server.Log.String())
			t.Errorf("FormNew: Wrong return Code: Got %d, expected %d. \n\tTest: %v",
				resp.StatusCode,
				200,
				test)
		}
		doc, err := html.Parse(resp.Body)
		if err != nil {
			t.Fatal("Error parsing response Bopy")
		}
		errMsgs := checkForAlerts(doc)
		if test.ErrCount != len(errMsgs) {
			t.Errorf("Should have %d alert-warnings found: %d \nErrors: &v", test.ErrCount, len(errMsgs), errMsgs)
		}

		// Post Db check if user was actually created
		if test.ErrCount == 0 {
			// Check auth
			var d string
			if strings.HasSuffix(test.Domain, ".") {
				d = test.Domain
			} else {
				d = test.Domain + "."
			}
			a := server.Db.GetAuth(d)
			if a == nil {
				t.Error("User was not created:", d)
				continue
			}
			ok, err := a.CheckAuth(d, test.Secret)
			if err != nil {
				t.Fatal("Failed to check authenticity")
			}
			if !ok {
				t.Error("Saved Auth does not match input", test, a)
			}

			// Check entry
			e := server.Db.GetEntry(d)
			if e == nil || e.Hostname != d {
				t.Error("Domain does not match entry", test, e)
			}
		}
		if t.Failed() {
			t.Logf("TEST: %v\n", test)
			t.Log("\tErrMsgs:", errMsgs)
			t.Log("\t", resp)
			t.Log(server.Log.String())
		}
	}
}

// This is where it gets funny. We implement a little html parser to see how
// many alert-warnings there are. Returns a String containing all error
// messages.
func checkForAlerts(doc *html.Node) []string {
	var errors []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, a := range n.Attr {
				if a.Key == "class" {
					if strings.Contains(a.Val, "alert-warning") {
						errors = append(errors, n.FirstChild.Data)
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return errors
}

func getFormUpdateURL(server string) *url.URL {
	URL, err := url.Parse(server)
	if err != nil {
		return nil
	}
	URL.Path = "/update"

	return URL
}

type formdomainupdatetest struct {
	Domain   string
	Secret   string
	Ips      string
	Cname    string
	Mxs      string
	Txts     string
	ErrCount int
	ExSecret string
	Entry    Entry
}

var formdomainupdatetests = []formdomainupdatetest{
	// Simple example
	formdomainupdatetest{
		Domain:   "test1.ist.nicht.cool",
		Secret:   "123456789",
		Ips:      "192.168.0.1\n 2a00:1450:4008:c01::65",
		Cname:    "",
		Mxs:      "10 mail.provider.tld",
		Txts:     "",
		ErrCount: 0,
		ExSecret: "123456789",
		Entry: Entry{
			Hostname: "test1.ist.nicht.cool.",
			Ip6s: []net.IP{
				net.ParseIP("2a00:1450:4008:c01::65"),
			},
			Ip4s: []net.IP{
				net.ParseIP("192.168.0.1"),
			},
			Offline: false,
			Txts:    nil,
			Mxs: []MxEntry{
				MxEntry{
					ip:       "mail.provider.tld.",
					priority: 10,
				},
			},
		},
	},
	// More complicated example
	formdomainupdatetest{
		Domain:   "test2.ist.nicht.cool",
		Secret:   "123456789",
		Ips:      "192.168.0.1 \n192.168.0.2 \n192.168.0.3 \n2a00:1450:4008:c01::64 \n192.168.0.4 \n2a00:1450:4008:c01::65",
		Cname:    "",
		Mxs:      "10 mail.provider.tld \n100 mail2.provider.tld",
		Txts:     "",
		ErrCount: 0,
		ExSecret: "123456789",
		Entry: Entry{
			Hostname: "test2.ist.nicht.cool.",
			Ip6s: []net.IP{
				net.ParseIP("2a00:1450:4008:c01::64"),
				net.ParseIP("2a00:1450:4008:c01::65"),
			},
			Ip4s: []net.IP{
				net.ParseIP("192.168.0.1"),
				net.ParseIP("192.168.0.2"),
				net.ParseIP("192.168.0.3"),
				net.ParseIP("192.168.0.4"),
			},
			Offline: false,
			Txts:    []string{},
			Mxs: []MxEntry{
				MxEntry{
					ip:       "mail.provider.tld.",
					priority: 10,
				},
				MxEntry{
					ip:       "mail2.provider.tld.",
					priority: 100,
				},
			},
			Cname: "",
		},
	},
	// Empty example
	formdomainupdatetest{
		Domain:   "",
		Secret:   "",
		Ips:      "",
		Cname:    "",
		Mxs:      "",
		Txts:     "",
		ErrCount: 2,
		Entry:    Entry{},
	},
	// Wrong password
	formdomainupdatetest{
		Domain:   "wrongpw.ist.nicht.cool",
		Secret:   "wrong",
		Ips:      "192.168.0.1\n 2a00:1450:4008:c01::65",
		Cname:    "",
		Mxs:      "10 mail.provider.tld",
		Txts:     "",
		ErrCount: 1,
		ExSecret: "123456789",
		Entry: Entry{
			Hostname: "wrongpw.ist.nicht.cool.",
			Ip6s: []net.IP{
				net.ParseIP("2a00:1450:4008:c01::65"),
			},
			Ip4s: []net.IP{
				net.ParseIP("192.168.0.1"),
			},
			Offline: false,
			Txts:    nil,
			Mxs: []MxEntry{
				MxEntry{
					ip:       "mail.provider.tld.",
					priority: 10,
				},
			},
		},
	},
	// non existing Domain
	formdomainupdatetest{
		Domain:   "noexist.ist.nicht.cool",
		Secret:   "wrong",
		Ips:      "192.168.0.1\n 2a00:1450:4008:c01::65",
		Cname:    "",
		Mxs:      "10 mail.provider.tld",
		Txts:     "",
		ErrCount: 1,
		ExSecret: "123456789",
		Entry:    Entry{},
	},
	// Errornous Ips but with fqdn
	formdomainupdatetest{
		Domain:   "sillyip.ist.nicht.cool.",
		Secret:   "123456789",
		Ips:      "192.168.300.1\n 2a00:1450:40x8:c01::65",
		Cname:    "",
		Mxs:      "10 mail.provider.tld",
		Txts:     "",
		ErrCount: 1,
		ExSecret: "123456789",
		Entry: Entry{
			Hostname: "sillyip.ist.nicht.cool.",
			Ip6s:     []net.IP{},
			Ip4s:     []net.IP{},
			Offline:  false,
			Txts:     nil,
			Mxs: []MxEntry{
				MxEntry{
					ip:       "mail.provider.tld.",
					priority: 10,
				},
			},
		},
	},
	// Errornous MX but with fqdn
	formdomainupdatetest{
		Domain:   "sillymx.ist.nicht.cool.",
		Secret:   "123456789",
		Ips:      "192.168.0.1\n 2a00:1450:4008:c01::65",
		Cname:    "",
		Mxs:      "1a mail.provider.tld",
		Txts:     "",
		ErrCount: 1,
		ExSecret: "123456789",
		Entry: Entry{
			Hostname: "sillymx.ist.nicht.cool.",
			Ip6s: []net.IP{
				net.ParseIP("2a00:1450:4008:c01::65"),
			},
			Ip4s: []net.IP{
				net.ParseIP("192.168.0.1"),
			},
			Offline: false,
			Txts:    nil,
			Mxs: []MxEntry{
				MxEntry{
					ip:       "mail.provider.tld.",
					priority: 10,
				},
			},
		},
	},
}

func TestFormDomainUpdate(t *testing.T) {
	server := createTestServer(t)
	defer server.S.Close()

	// Add entries based on entry element
	for _, test := range formdomainupdatetests {
		if test.Entry.Hostname == "" {
			continue
		}
		auth, _ := NewAuth(test.Entry.Hostname, test.ExSecret)
		err := server.Db.SaveAuth(auth)
		if err != nil {
			t.Fatal("Failed to create Users")
		}
		err = server.Db.SaveEntry(&Entry{
			Hostname: test.Entry.Hostname,
			Offline:  false,
		})
		if err != nil {
			t.Fatal("Failed to save Entry")
		}
	}

	for _, test := range formdomainupdatetests {
		v := url.Values{}
		if test.Domain != "" {
			v.Set("domain", test.Domain)
		}
		if test.Secret != "" {
			v.Set("secret", test.Secret)
		}
		if test.Cname != "" {
			v.Set("cname", test.Cname)
		}
		if test.Ips != "" {
			v.Set("ip", test.Ips)
		}
		if test.Mxs != "" {
			v.Set("mx", test.Mxs)
		}
		if test.Txts != "" {
			v.Set("txt", test.Txts)
		}
		URL := getFormUpdateURL(server.S.URL)

		resp, err := http.PostForm(URL.String(), v)
		if err != nil {
			t.Log(server.Log.String())
			t.Fatal("Failed to update URL:", URL.String(), err)
		}
		if resp.StatusCode != 200 {
			t.Log(server.Log.String())
			t.Errorf("FormNew: Wrong return Code: Got %d, expected %d. \n\tTest: %v",
				resp.StatusCode,
				200,
				test)
		}
		doc, err := html.Parse(resp.Body)
		if err != nil {
			t.Fatal("Error parsing response Bopy")
		}
		errMsgs := checkForAlerts(doc)
		if test.ErrCount != len(errMsgs) {
			t.Errorf("Should have %d alert-warnings found: %d \nErrors: &v", test.ErrCount, len(errMsgs), errMsgs)
		}

		// Post Db check if user was actually created
		if test.ErrCount == 0 {
			// Check entry
			e := server.Db.GetEntry(test.Entry.Hostname)
			if e == nil {
				t.Error("Domain does not match exist", test, e)
			}
			if e.String() != test.Entry.String() {
				t.Errorf("Entries do not match: \nIs:\t %v\nEx:\t %v",
					e,
					&test.Entry)
			}
		}
		if t.Failed() {
			t.Logf("TEST: %v\n", test)
			t.Log("\tErrMsgs:", errMsgs)
			t.Log("\t", resp)
			t.Log(server.Log.String())
		}
	}
}
