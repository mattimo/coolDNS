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
	"os"
)

// Server Confiuration object holds instance specific variables
//
// If all values for a specific feature are set, the corrensponding feature is
// activated, These shall not be changed during server Runtime.
type Config struct {
	DbFile       string
	Domain       string // fqdn of the full Domain name
	WebConfig    *WebConfig
	DnsConfig    *DnsServerConfig // Server Configuration
	InfluxConfig *InfluxConfig    // Influx DB configuration
}

func LoadConfig() *Config {
	c := new(Config)
	c.WebConfig = loadWebConfig()
	c.DnsConfig = loadDnsConfig()
	c.InfluxConfig = loadInfluxConfig()
	c.SetDomain(os.Getenv("COOLDNS_SUFFIX"))
	return c
}

func loadWebConfig() *WebConfig {
	w := new(WebConfig)
	w.Resources = os.Getenv("COOLDNS_WEB_RESOURCES")
	if w.Resources == "" {
		w.Resources = "./"
	}
	w.Listen = os.Getenv("COOLDNS_WEB_LISTEN")
	if w.Listen == "" {
		w.Listen = ":3000"
	}
	w.RcPubKey = os.Getenv("COOLDNS_RC_PUB")
	w.RcPrivKey = os.Getenv("COOLDNS_RC_PRIV")
	return w
}

func loadDnsConfig() *DnsServerConfig {
	listen := os.Getenv("COOLDNS_DNS_LISTEN")
	if listen == "" {
		listen = ":8053"
	}
	return &DnsServerConfig{
		Listen:  listen,
		TsigKey: os.Getenv("COOLDNS_TSIG_KEY"),
	}

}

func loadInfluxConfig() *InfluxConfig {
	host := os.Getenv("COOLDNS_INFLUX_HOST")
	database := os.Getenv("COOLDNS_INFLUX_DB")
	user := os.Getenv("COOLDNS_INFLUX_USER")
	password := os.Getenv("COOLDNS_INFLUX_PASS")
	if host == "" ||
		database == "" ||
		user == "" ||
		password == "" {

		return nil
	}
	return &InfluxConfig{
		Host:     host,
		Database: database,
		Username: user,
		Password: password,
	}
}

func (c *Config) SetDomain(d string) {
	c.Domain = d
	c.DnsConfig.Domain = d
	c.WebConfig.Domain = d
}
