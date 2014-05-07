package cooldns

import (
	"github.com/codegangsta/martini-contrib/render"
	"github.com/martini-contrib/binding"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	domainsuffix string = ".ist.nicht.cool."
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
	CNames   string `form:"cname"`
	Ips      string `form:"ip"`
	Mxs      string `form:"mx"`
	TXTs     string `form:"txt"`
}

type WebErrorHandler func(int, []string, interface{})

func init() {
	newsuffix := os.Getenv("COOLDNS_SUFFIX")
	if newsuffix != "" {
		domainsuffix = newsuffix
	}
}

func Index(db *CoolDB, r render.Render) {
	r.HTML(200, "index", map[string]string{"Rcpublic": rcPublicKey})
}

func Update(db *CoolDB, r render.Render) {
	r.HTML(200, "update", map[string]string{"rcpublic": rcPublicKey})
}

func checkNewDomain(n *WebNewDomain) (ok bool, errors []string) {
	ok = false
	// Check if Hostname is fqdn with needed suffix and a minimum of two
	// characters as a sub domain
	if !strings.HasSuffix(n.Hostname, domainsuffix) {
		n.Hostname = n.Hostname + domainsuffix
		//		errors = append(errors, "Hostname has Wrong suffix")
		//		return
	}
	if len(strings.TrimSuffix(n.Hostname, domainsuffix)) < 2 {
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
	if !strings.HasSuffix(n.Hostname, domainsuffix) {
		n.Hostname = n.Hostname + domainsuffix
	}
	if len(strings.TrimSuffix(n.Hostname, domainsuffix)) < 2 {
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
	Rcpublic string
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
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, domainsuffix)
		view := &newView{
			Rcpublic: rcPublicKey,
			Err:      errors,
			F:        vContent,
		}
		r.HTML(errCode, "index", view)
	}
	newDomain(db, r, n, errors, req, errHandler)
}

type updateView struct {
	Err []string         // Occured Errors
	F   *WebUpdateDomain // Prefilled items
}

func FormApiDomainUpdate(db *CoolDB,
	r render.Render,
	n WebUpdateDomain,
	errors binding.Errors,
	req *http.Request) {

	errHandler := func(errCode int, errors []string, content interface{}) {
		vContent := content.(*WebUpdateDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, domainsuffix)
		view := &updateView{
			Err: errors,
			F:   vContent,
		}
		r.HTML(errCode, "update", view)
	}
	UpdateDomain(db, r, n, errors, req, errHandler)
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
	errHandler WebErrorHandler) {

	// Check object for sanity
	ok, nerrors := checkUpdateDomain(&n)
	if !ok {
		errHandler(200, nerrors, &n)
		return
	}

	// Get Auth
	a := DNSDB.GetUser(n.Hostname)
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
	// Look for cnames
	exists, cnames := extractRecords(n.CNames)
	if exists {
		entry.Cnames = cnames
	}
	// Look for Ips
	exists, Ips := extractRecords(n.Ips)
	if exists {
		// TODO: Well this is pretty lame, we have to find a way
		// to macht A and AA Entries
		entry.MyIp4 = net.ParseIP(Ips[0])
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
			entry.Mx = append(entry.Mx, MxEntry{
				ip:       net.ParseIP(mxa[1]),
				priority: int(prio),
			})
		}
	}
	// Look for Txts
	exists, txts := extractRecords(n.TXTs)
	if exists {
		entry.Txt = txts[0]
	}
	err = db.SaveEntry(entry)
	if err != nil {
		log.Println("New Domain: Entry could not be saved", err)
		errHandler(500, []string{"Internal Server Error"}, &n)
		return
	}
	errHandler(200, []string{}, &n)

}

func newDomain(db *CoolDB,
	r render.Render,
	n WebNewDomain,
	errors binding.Errors,
	req *http.Request,
	errHandler WebErrorHandler) {

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
		Hostname: n.Hostname,
		Secret:   n.Secret,
	}
	uView := updateView{
		Err: []string{},
		F:   update,
	}
	r.HTML(200, "update", &uView)
}
