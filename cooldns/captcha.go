// The CoolDNS Project. The simple dynamic dns server and update service.
// Copyright (C) 2014 The CoolDNS Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.package main

package cooldns

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	reCaptchaURL = "http://www.google.com/recaptcha/api/verify"
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
	trans     *http.Transport
	client    *http.Client
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
	if !pool.AppendCertsFromPEM(certBundle) {
		log.Fatal("Could not load Certs")
	}
}

func checkReCaptcha(remoteip, challenge, response, privKey string) (string, error) {
	res, err := client.PostForm(reCaptchaURL,
		url.Values{
			"privatekey": {privKey},
			"remoteip":   {remoteip},
			"challenge":  {challenge},
			"response":   {response}})
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

func ReCaptcha(remoteip, challenge, response, privKey string) (bool, error) {
	answer, err := checkReCaptcha(remoteip, challenge, response, privKey)
	if err != nil {
		return false, err
	}
	// Check for errors
	if !strings.HasPrefix(answer, "true") {
		return false, nil
	}
	return true, nil
}
