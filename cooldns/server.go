package cooldns

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/auth"
	"log"
	"fmt"
	"net/http"
	"net"
	"strings"
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

func Register(db *CoolDB,reg Registration, errors binding.Errors) string {
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
		MyIp: net.ParseIP(reg.MyIp),
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


	m := martini.Classic()
	m.Use(auth.Basic("root", "123"))
	m.Map(db)

	// binding registration
	regBinding := binding.Form(Registration{})

	m.Get("/", regBinding, Register)


	m.Run()
}
