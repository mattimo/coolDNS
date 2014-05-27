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
