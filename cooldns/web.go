package cooldns

import (
	"github.com/codegangsta/martini-contrib/render"
	"github.com/martini-contrib/binding"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type WebNewDomain struct {
	Hostname string `json:"hostname" form:"domain"`
	Secret   string `json:"secret" form:"secret"`
	RcChal   string `json:"rcchal" form:"recaptcha_challenge_field"`
	RcResp   string `json:"rcresp" form:"recaptcha_response_field"`
}

type WebUpdateDomain struct {
	Hostname string `form:"domain"`
	Secret   string `form:"secret"`
	CName    string `form:"cname"`
	Ips      string `form:"ip"`
	Mxs      string `form:"mx"`
	TXTs     string `form:"txt"`
}

// Web Error Handler function signature. Helps you interface with errors
type WebErrorHandler func(int, []string, interface{})

// Web Success Handler function signature. Helps interface with success messages
type WebSuccessHandler func([]string, interface{})

func Index(db *CoolDB, r render.Render) {
	r.HTML(200, "index", map[string]string{
		"Rcpublic": rcPublicKey,
		"Domain":   domainsuffix})
}

func Update(db *CoolDB, r render.Render) {
	r.HTML(200, "update", map[string]string{
		"Rcpublic": rcPublicKey,
		"Domain":   domainsuffix})
}

func checkNewDomain(n *WebNewDomain) (ok bool, errors []string) {
	ok = false
	// Check if Hostname is fqdn with needed suffix and a minimum of two
	// characters as a sub domain
	if !strings.HasSuffix(n.Hostname, "."+domainsuffix) {
		n.Hostname = n.Hostname + "." + domainsuffix
	}
	if len(strings.TrimSuffix(n.Hostname, "."+domainsuffix)) < 2 {
		errors = append(errors, "Sub domain to short")
	}
	// Check if secret exists
	if n.Secret == "" {
		errors = append(errors, "Secret Missing")
	}
	// Check if reCAPTCHA Challenge exists
	if n.RcChal == "" {
		errors = append(errors, "reCAPTCHA challenge missing")
	}
	// Check if reCAPTCHA response exists
	if n.RcResp == "" {
		errors = append(errors, "reCAPTCHA response missing")
	}

	// conclusion
	if len(errors) == 0 {
		ok = true
	}
	return
}

func checkUpdateDomain(n *WebUpdateDomain) (ok bool, errors []string) {
	ok = false
	// Check if Hostname is fqdn with needed suffix and a minimum of two
	// characters as a sub domain
	if !strings.HasSuffix(n.Hostname, "."+domainsuffix) {
		n.Hostname = n.Hostname + "." + domainsuffix
	}
	if len(strings.TrimSuffix(n.Hostname, "."+domainsuffix)) < 2 {
		errors = append(errors, "Sub domain to short")
	}
	// Check if secret exists
	if n.Secret == "" {
		errors = append(errors, "Secret Missing")
	}
	// conclusion
	if len(errors) == 0 {
		ok = true
	}
	return
}

type newView struct {
	Domain   string        // Domain base name
	Rcpublic string        // reCaptcha Public Key
	Err      []string      // Occured Errors
	F        *WebNewDomain // Prefilled items
}

func FormApiDomainNew(db *CoolDB,
	r render.Render,
	n WebNewDomain,
	errors binding.Errors,
	req *http.Request) {

	errHandler := func(errCode int, errors []string, content interface{}) {
		vContent := content.(*WebNewDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, "."+domainsuffix)
		view := &newView{
			Domain:   domainsuffix,
			Rcpublic: rcPublicKey,
			Err:      errors,
			F:        vContent,
		}
		r.HTML(errCode, "index", view)
	}

	success := func(success []string, content interface{}) {
		vContent := content.(*WebUpdateDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, "."+domainsuffix)
		view := &updateView{
			Domain:  domainsuffix,
			Success: success,
			F:       vContent,
		}
		r.HTML(200, "update", view)
	}

	newDomain(db, r, n, errors, req, errHandler, success)
}

type updateView struct {
	Domain  string           // Domain base name
	Err     []string         // Occured Errors
	F       *WebUpdateDomain // Prefilled items
	Success []string         // Success string
}

func FormApiDomainUpdate(db *CoolDB,
	r render.Render,
	n WebUpdateDomain,
	errors binding.Errors,
	req *http.Request) {

	errHandler := func(errCode int, errors []string, content interface{}) {
		vContent := content.(*WebUpdateDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, "."+domainsuffix)
		view := &updateView{
			Domain: domainsuffix,
			Err:    errors,
			F:      vContent,
		}
		r.HTML(errCode, "update", view)
	}
	success := func(success []string, content interface{}) {
		vContent := content.(*WebUpdateDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, "."+domainsuffix)
		view := &updateView{
			Domain:  domainsuffix,
			Success: success,
			F:       vContent,
		}
		r.HTML(200, "update", view)
	}

	UpdateDomain(db, r, n, errors, req, errHandler, success)
}

func fqdn(s string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	if s[l-1] != '.' {
		return s + "."
	}
	return s
}

func extractRecords(input string) (exist bool, records []string) {
	records = strings.Split(input, "\n")
	for i, r := range records {
		records[i] = strings.TrimSpace(r)
	}
	return len(records) != 0, records
}

func UpdateDomain(db *CoolDB,
	r render.Render,
	n WebUpdateDomain,
	errors binding.Errors,
	req *http.Request,
	errHandler WebErrorHandler,
	successHandler WebSuccessHandler) {

	// Check object for sanity
	ok, nerrors := checkUpdateDomain(&n)
	if !ok {
		errHandler(200, nerrors, &n)
		return
	}

	// Get Auth
	a := db.Cache.GetUser(n.Hostname)
	if a == nil {
		errHandler(200, []string{"Hostname and Secret do not match"}, &n)
		return
	}
	// Check Authentication realm
	ok, err := a.CheckAuth(n.Hostname, n.Secret)
	if err != nil || !ok {
		errHandler(200, []string{"Hostname and Secret do not match"}, &n)
		return
	}

	// Create entry
	entry := &Entry{
		Hostname: n.Hostname,
		Offline:  false,
	}
	// Look for cname (abusing extractRecord function)
	exists, cname := extractRecords(n.CName)
	if exists {
		entry.Cname = fqdn(cname[0])
	}
	// Look for Ips
	exists, Ips := extractRecords(n.Ips)
	if exists {
		// TODO: Well this is pretty lame, we have to find a way
		// to match A and AAAA Entries
		for _, ipString := range Ips {
			ip := net.ParseIP(ipString)
			if ip == nil {
				continue //TODO fail louder!
			}
			if strings.Contains(ipString, ":") {
				entry.Ip6s = append(entry.Ip6s, ip)
			} else {
				entry.Ip4s = append(entry.Ip4s, ip)
			}
		}
	}
	// Look for MXs
	exists, mxs := extractRecords(n.Mxs)
	if exists {
		for _, mx := range mxs {
			mxa := strings.Split(mx, " ")
			prio, err := strconv.ParseInt(mxa[0], 10, 0)
			if err != nil {
				break
			}
			mxName := fqdn(mxa[1])
			entry.Mxs = append(entry.Mxs, MxEntry{
				ip:       mxName,
				priority: int(prio),
			})
		}
	}
	// Look for Txts
	exists, txts := extractRecords(n.TXTs)
	if exists {
		entry.Txts = txts

	}
	err = db.SaveEntry(entry)
	if err != nil {
		log.Println("New Domain: Entry could not be saved", err)
		errHandler(500, []string{"Internal Server Error"}, &n)
		return
	}
	n.Secret = ""
	successHandler([]string{"Domain was Successfully updated"}, &n)

}

func newDomain(db *CoolDB,
	r render.Render,
	n WebNewDomain,
	errors binding.Errors,
	req *http.Request,
	errHandler WebErrorHandler,
	successHandler WebSuccessHandler) {

	ok, nerrors := checkNewDomain(&n)
	if !ok {
		errHandler(200, nerrors, &n)
		return
	}
	remoteip := strings.Split(req.RemoteAddr, ":")[0]
	ok, err := ReCaptcha(remoteip, n.RcChal, n.RcResp)
	if err != nil {
		log.Println("NewDomain: Failed to verify reCAPTCHA:", err)
		errHandler(500, []string{"Internal Server Error"}, nil)
		return
	}
	if !ok {
		errHandler(200, []string{"reCAPTCHA is wrong"}, &n)
		return
	}
	// Create Authentication object
	auth, err := NewAuth(n.Hostname, n.Secret)
	if err != nil && err != AuthConstraintsNotMet {
		log.Println("New Domain: Failed to create Auth:", err)
		errHandler(500, []string{"Internal Server Error"}, &n)
		return
	}
	// Check if contrainsts were met
	if err == AuthConstraintsNotMet {
		errHandler(200, []string{"Auth Constaints not met"}, &n)
		return
	}
	// Save Auth object
	err = db.SaveAuth(auth)
	if err != nil {
		log.Println("New Domain: Auth could not be saved", err)
		errHandler(500, []string{"Internal Server Error"}, &n)
		return
	}
	// Create and save entry
	entry := &Entry{
		Hostname: n.Hostname,
		Offline:  false,
	}
	err = db.SaveEntry(entry)
	if err != nil {
		log.Println("New Domain: Entry could not be saved", err)
		errHandler(500, []string{"Internal Server Error"}, &n)
		return
	}
	update := &WebUpdateDomain{
		Hostname: strings.TrimSuffix(n.Hostname, domainsuffix),
		Secret:   n.Secret,
	}
	successHandler([]string{"Creation of new domain " + update.Hostname + " was successful"}, update)
}
