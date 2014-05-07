package cooldns

import (
	"github.com/codegangsta/martini-contrib/render"
	"github.com/martini-contrib/binding"
	"strings"
	"log"
	"os"
	"net/http"
)

var (
	domainsuffix string = ".ist.nicht.cool."
)

type WebNewDomain struct {
	Hostname string `json:"hostname" form:"domain"`
	Mx	string `json:"mx" form:""`
	Txt	string `json:"txt" form:""`
	Secret	string `json:"secret" form:"secret"`
	RcChal	string `json:"rcchal" form:"recaptcha_challenge_field"`
	RcResp	string `json:"rcresp" form:"recaptcha_response_field"`
}

func init() {
	newsuffix := os.Getenv("COOLDNS_SUFFIX")
	if newsuffix != "" {
		domainsuffix = newsuffix
	}
}

func Index(db *CoolDB, r render.Render) {
	r.HTML(200, "index", map[string]string{"rcpublic": rcPublicKey})
}

func Update(db *CoolDB, r render.Render) {
	r.HTML(200, "update", map[string]string{"rcpublic": rcPublicKey})
}

func checkNewDomain(n *WebNewDomain)  (ok bool, errors []string){
	ok = false
	// Check if Hostname is fqdn with needed suffix and a minimum of two 
	// characters as a sub domain
	if !strings.HasSuffix(n.Hostname, domainsuffix) {
		n.Hostname = n.Hostname+domainsuffix
//		errors = append(errors, "Hostname has Wrong suffix")
//		return
	}
	if len(strings.TrimSuffix(n.Hostname, domainsuffix)) < 2 {
		errors = append(errors, "Sub domain to short")
	}
	// Ignore mx if not set
	if n.Mx != "" {
		// Check if mx is fqdn and has at least three chars long and 
		// has a Dot at the end
		if len(n.Mx) < 3 || strings.HasSuffix(n.Mx, ".") {
			errors = append(errors, "Malformed MX")
		}
	}
	// Check Txt element if it isn't insanely large
	if len(n.Txt) > 4096 {
		errors = append(errors, "TXT too long")
	}
	// Check if secret exists
	if n.Secret == "" {
		errors = append(errors, "Secret Missing")
	}
	// Check if reCAPTCHA Challenge exists
	if n.RcChal == "" {
		errors =  append(errors, "reCAPTCHA challenge missing")
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

func FormApiDomainNew(db *CoolDB,
			r render.Render,
			n WebNewDomain,
			errors binding.Errors,
			req *http.Request) {
	NewDomain(db, r, n, errors, req)
}

func NewDomain(db *CoolDB, r render.Render, n WebNewDomain, errors binding.Errors, req *http.Request) {
	ok, nerrors := checkNewDomain(&n)
	if !ok {
		r.JSON(200, nerrors)
	}
	remoteip := strings.Split(req.RemoteAddr, ":")[0]
	ok, err := ReCaptcha(remoteip, n.RcChal, n.RcResp)
	if err != nil {
		log.Println("NewDomain: Failed to verify reCAPTCHA:", err)
		r.JSON(500, "")
		return
	}
	if !ok {
		r.JSON(200, map[string]string{"Err":"captcha"})
		return
	}
	// Create Authentication object
	auth, err := NewAuth(n.Hostname, n.Secret)
	if err != nil && err != AuthConstraintsNotMet {
		log.Println("New Domain: Failed to create Auth:", err)
		r.JSON(500, "")
		return
	}
	// Check if contrainsts were met
	if err == AuthConstraintsNotMet {
		r.JSON(200, map[string]string{"Err":"authContraints"})
		return
	}
	// Save Auth object
	err = db.SaveAuth(auth)
	if err != nil {
		log.Println("New Domain: Auth could not be saved", err)
		r.JSON(500, "")
		return
	}
	// Create and save entry
	entry := &Entry{
		Hostname: n.Hostname,
		Txt:	n.Txt,
	}
	err = db.SaveEntry(entry)
	if err != nil {
		log.Println("New Domain: Entry could not be saved", err)
		r.JSON(500, "")
		return
	}
	r.HTML(200, "update", nil)
}


