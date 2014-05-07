package cooldns

import (
	"net/http"
	"crypto/x509"
	"log"
	"io/ioutil"
	"crypto/tls"
	"os"
	"errors"
	"net/url"
	"strings"
)

var (
	pool *x509.CertPool
	// locations to search for bundled ssl certfiles
	certSearch []string = []string{"/etc/ssl/cert.pem", // Recomended by the go doc
				// Found this one under fedora 19, seems to be
				// part of the one central cert pool for 
				// everything project.
				"/etc/ssl/certs/ca-bundle.crt",
			}
	tlsConfig *tls.Config
	trans *http.Transport
	client *http.Client
)

func init() {
	pool = x509.NewCertPool()
	loadCertPool(pool)

	tlsConfig = &tls.Config{
			RootCAs: pool,
		}
	trans = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	client = &http.Client{
			Transport: trans,
		}

}

func searchCerts() ([]byte, error) {
	for _, cFile := range certSearch {
		r, err := os.Open(cFile)
		if err == nil {
			defer r.Close()
			return ioutil.ReadAll(r)
		}
	}
	return nil, errors.New("No certificate bundle found")
}

func loadCertPool(pool *x509.CertPool) {
	// search for cert pool
	certBundle, err := searchCerts()
	if err != nil {
		log.Fatal("Error Loading certificates:", err)
	}
	if !pool.AppendCertsFromPEM(certBundle){
		log.Fatal("Could not load Certs")
	}
}
const (
	rcPublicKey string = "publicKeyAABBCCDD"
	rcPrivateKey string = "privateAABBCCDD"

	reCaptchaURL = "http://www.google.com/recaptcha/api/verify"

)

func checkReCaptcha(challenge, response string) (string, error){
	res, err := client.PostForm(reCaptchaURL,
				url.Values{
					"privatekey": {rcPublicKey},
					"remoteip": {rcPrivateKey},
					"challenge": {challenge},
					"response": {response}})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func ReCaptcha(challenge, response string) (bool, error) {
	answer, err := checkReCaptcha(challenge, response)
	if err != nil {
		return false, err
	}
	// Check for errors
	if !strings.HasPrefix(answer, "true") {
		return false, nil
	}
	return true, nil
}
