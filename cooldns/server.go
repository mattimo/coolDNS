package cooldns

import (
	"github.com/go-martini/martini"
	"github.com/codegangsta/martini-contrib/render"
	"github.com/martini-contrib/binding"
	"log"
	"fmt"
	"net/http"
	"net"
	"strings"
	"encoding/base64"
)

type Registration struct {
	Hostname	string `form:"hostname"`
	MyIp		string `form:"myip"`
	Offline		string `form:"offline"`
	Txt		string `form:"txt"`
}

func (r *Registration) Validate(errors binding.Errors, req *http.Request) (binding.Errors) {
	if req.Form.Get("hostname") == "" {
		errors = append(errors, binding.Error{
			Classification: binding.RequiredError,
			Message: "hostname Field is empty",
		})
	}
	if req.Form.Get("myip") == "" {
		errors = append(errors, binding.Error{
			Classification: binding.RequiredError,
			Message: "myip Field is empty",
		})
	}
	ip := net.ParseIP(req.Form.Get("myip"))
	if ip == nil {
		errors = append(errors, binding.Error{
			Classification: binding.ContentTypeError,
			Message: "myip is not an IP Address",
		})
	}
	offline := strings.ToLower(req.Form.Get("offline"))
	if offline != "" && offline != "yes" && offline != "no" && offline != "maybe"{
		errors = append(errors, binding.Error{
			Classification: binding.ContentTypeError,
			Message: "offline is neither yes nor no",
		})
	}
	return errors
}

func returnAuthErr(res http.ResponseWriter, errMsg string) {
	res.Header().Set("WWW-Authenticate", "Basic realm=\" "+errMsg+"\"")
	http.Error(res, "Not Authorized", http.StatusUnauthorized)
	return
}

func AuthHandler(res http.ResponseWriter, req *http.Request) {
	// Get name and secret from auth
	rAuthString := req.Header.Get("Authorization")
	if rAuthString == "" {
		returnAuthErr(res, "Authorization Required")
		return
	}
	// Decode base64 auth string, but strip away the "Basic " auth method 
	// decleration
	rAuth , err := base64.StdEncoding.DecodeString(strings.Replace(rAuthString, "Basic ", "", 1))
	if err != nil {
		returnAuthErr(res, "Malfencoded authorization string")
		return
	}
	// Get the two values separeated by a colon.
	// If there is more then one colon the the realm it must be wrong, so 
	// yield an error.
	rAuthArray := strings.Split(string(rAuth), ":")
	if len(rAuthArray) != 2 {
		returnAuthErr(res, "Malformed authorization string")
		return
	}
	// Get the name and secret
	rName, rSecret := rAuthArray[0], rAuthArray[1]

	// Parse the requests Form and return an error on failure
	err = req.ParseForm()
	if err != nil {
		returnAuthErr(res, "Malformed Request")
		return
	}
	// Get the hostname out of the Header and check if it is neither
	// empty nor different from the user name.
	hostname := req.Form.Get("hostname")
	if hostname == "" || hostname != rName{
		returnAuthErr(res, "User does not match request hostname")
		return
	}
	// Get the user name from the user db, if the user doesn't exist we 
	// just return an error stating that the user does not exist. This is 
	// Totally ok because the username equals the domain name that is 
	// public anyway
	a := DNSDB.GetUser(rName)
	if a == nil {
		log.Println("No User for hostname:", rName)
		returnAuthErr(res, "hostname does not exist")
		return
	}

	// Check Authentication realm
	ok, err := a.CheckAuth(rName, rSecret)
	if err != nil || !ok {
		returnAuthErr(res, "Secret is Wrong")
		log.Println("Auth is not Valid, You shall not pass", rName)
		return
	}
}

func Register(db *CoolDB,r render.Render, reg Registration, errors binding.Errors) string {
	if errors != nil {
		return fmt.Sprintf("%v", errors)
	}
	// Set Offline flag, does nothing yet but it's nice to have
	var offline bool
	if strings.ToLower(reg.Offline) == "yes" {
		offline = true
	} else {
		offline = false
	}

	e := &Entry{
		Hostname: reg.Hostname,
		MyIp4: net.ParseIP(reg.MyIp),
		Offline: offline,
		Txt: reg.Txt,
	}
	err := db.SaveEntry(e)
	if err != nil {
		log.Println("Error saving element:", err)
	}
	return fmt.Sprintln(reg)
}

func Run() {
	log.Println("Starting coolDNS Server")

	db, err := NewCoolDB("cool.db")
	if err != nil {
		log.Fatal("Error Loading db:", err)
	}
	dnsCache, err := db.LoadAll()
	if err != nil {
		log.Fatal("Error Loading DNS Cache:", err)
	}
	userCache, err := db.LoadUsers()
	if err != nil {
		log.Fatal("Error Loading User Cache:", err)
	}
	DNSDB.LoadCache(dnsCache, userCache)
	err = createDummyUser("doof.ist.nicht.cool.", "12345678", db)
	if err != nil {
		log.Println("Error adding user:", err)
	}

	m := martini.Classic()
	m.Map(db)
	m.Use(render.Renderer())
	m.Use(martini.Static("assets"))

	// binding registration
	regBinding := binding.Form(Registration{})

	m.Get("/nic/update", AuthHandler, regBinding, Register)
	m.Get("/", binding.Json(WebNewDomain{}), Index)
	go RunDns()

	m.Run()
}

func createDummyUser(name, secret string, db *CoolDB) error {
	a, err := NewAuth(name, secret)
	if err != nil {
		return err
	}

	ok, err := a.CheckAuth(name, secret)
	if err != nil {
		return err
	}
	if !ok {
		log.Println("Woha auth didn't match")
	}
	err = db.SaveAuth(a)
	if err != nil {
		return err
	}
	return nil

}
