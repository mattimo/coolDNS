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

type Web struct {
	Domain    string
	RcPubKey  string
	RcPrivKey string
}

func NewWeb(c *WebConfig) *Web {
	return &Web{
		Domain:    c.Domain,
		RcPubKey:  c.RcPubKey,
		RcPrivKey: c.RcPrivKey,
	}
}

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

func (w *Web) Index(db CoolDB, r render.Render) {
	r.HTML(200, "index", map[string]string{
		"Rcpublic": w.RcPubKey,
		"Domain":   "." + w.Domain})
}

func (w *Web) Update(db CoolDB, r render.Render) {
	r.HTML(200, "update", map[string]string{
		"Rcpublic": w.RcPubKey,
		"Domain":   "." + w.Domain})
}

func (w *Web) checkNewDomain(n *WebNewDomain) (ok bool, errors []string) {
	ok = false
	// Check if new domain is valid
	hok := false
	n.Hostname, hok = ValidateDomain(n.Hostname, w.Domain)
	if !hok {
		errors = append(errors, "Hostname not Valid")
	}
	// Check if secret exists
	if n.Secret == "" {
		errors = append(errors, "Secret Missing")
	}
	// Check if reCAPTCHA Challenge exists
	if w.RcPubKey != "" && n.RcChal == "" {
		errors = append(errors, "reCAPTCHA challenge missing")
	}
	// Check if reCAPTCHA response exists
	if w.RcPubKey != "" && n.RcResp == "" {
		errors = append(errors, "reCAPTCHA response missing")
	}

	// conclusion
	if len(errors) == 0 {
		ok = true
	}
	return
}

func (w *Web) checkUpdateDomain(n *WebUpdateDomain) (ok bool, errors []string) {
	ok = false
	// Check if new domain is valid
	hok := false
	n.Hostname, hok = ValidateDomain(n.Hostname, w.Domain)
	if !hok {
		errors = append(errors, "Hostname not Valid")
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

func (w *Web) FormApiDomainNew(db CoolDB,
	r render.Render,
	n WebNewDomain,
	errors binding.Errors,
	req *http.Request) {

	errHandler := func(errCode int, errors []string, content interface{}) {
		vContent := content.(*WebNewDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, "."+w.Domain)
		view := &newView{
			Domain:   "." + w.Domain,
			Rcpublic: w.RcPubKey,
			Err:      errors,
			F:        vContent,
		}
		r.HTML(errCode, "index", view)
	}

	success := func(success []string, content interface{}) {
		vContent := content.(*WebUpdateDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, "."+w.Domain)
		view := &updateView{
			Domain:  "." + w.Domain,
			Success: success,
			F:       vContent,
		}
		r.HTML(200, "update", view)
	}

	w.newDomain(db, r, n, errors, req, errHandler, success)
}

type updateView struct {
	Domain  string           // Domain base name
	Err     []string         // Occured Errors
	F       *WebUpdateDomain // Prefilled items
	Success []string         // Success string
}

func (w *Web) FormApiDomainUpdate(db CoolDB,
	r render.Render,
	n WebUpdateDomain,
	errors binding.Errors,
	req *http.Request) {

	errHandler := func(errCode int, errors []string, content interface{}) {
		vContent := content.(*WebUpdateDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, "."+w.Domain)
		view := &updateView{
			Domain: "." + w.Domain,
			Err:    errors,
			F:      vContent,
		}
		r.HTML(errCode, "update", view)
	}
	success := func(success []string, content interface{}) {
		vContent := content.(*WebUpdateDomain)
		vContent.Hostname = strings.TrimSuffix(vContent.Hostname, "."+w.Domain)
		view := &updateView{
			Domain:  "." + w.Domain,
			Success: success,
			F:       vContent,
		}
		r.HTML(200, "update", view)
	}

	w.UpdateDomain(db, r, n, errors, req, errHandler, success)
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

func extractRecords(input string) (bool, []string) {
	records := strings.Split(input, "\n")
	var newRec []string
	for _, r := range records {
		rec := strings.TrimSpace(r)
		if rec != "" {
			newRec = append(newRec, rec)
		}
	}
	return len(newRec) != 0, newRec
}

func (w *Web) UpdateDomain(db CoolDB,
	r render.Render,
	n WebUpdateDomain,
	errors binding.Errors,
	req *http.Request,
	errHandler WebErrorHandler,
	successHandler WebSuccessHandler) {

	// Check object for sanity
	ok, nerrors := w.checkUpdateDomain(&n)
	if !ok {
		errHandler(200, nerrors, &n)
		return
	}

	// Get Auth
	a := db.GetAuth(n.Hostname)
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
				errHandler(200, []string{"Malformatted Ip Address"}, &n)
				return
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
				errHandler(200, []string{"Malformatted MX Entry"}, &n)
				return
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

func (w *Web) newDomain(db CoolDB,
	r render.Render,
	n WebNewDomain,
	errors binding.Errors,
	req *http.Request,
	errHandler WebErrorHandler,
	successHandler WebSuccessHandler) {

	ok, nerrors := w.checkNewDomain(&n)
	if !ok {
		errHandler(200, nerrors, &n)
		return
	}
	remoteip := strings.Split(req.RemoteAddr, ":")[0]
	if w.RcPubKey != "" {
		ok, err := ReCaptcha(remoteip, n.RcChal, n.RcResp, w.RcPrivKey)
		if err != nil {
			log.Println("NewDomain: Failed to verify reCAPTCHA:", err)
			errHandler(500, []string{"Internal Server Error"}, nil)
			return
		}
		if !ok {
			errHandler(200, []string{"reCAPTCHA is wrong"}, &n)
			return
		}
	}
	// Check if Domain already exists
	if db.GetEntry(n.Hostname) != nil {
		errHandler(200, []string{"Sorry, Domain already in use"}, &n)
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
		errHandler(200, []string{"Auth Constraints not met"}, &n)
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
		Hostname: strings.TrimSuffix(n.Hostname, "."+w.Domain),
		Secret:   n.Secret,
	}
	successHandler([]string{"Creation of new domain " + update.Hostname + " was successful"}, update)
}
