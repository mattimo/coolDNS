# coolDNS

Fully Qualified Cool Domain Names

## Config

* `COOLDNS_RC_PUB` The reCAPTCHA public Key
* `COOLDNS_RC_PRIV` The reCAPTCHA private Key

* `COOLDNS_SUFFIX` The cool dns domain suffix

InfluxDB specific configuration, sending metrics to Influx only works if all 
of the following values are set.

* `COOLDNS_INFLUX_HOST` Influx Hostname with <addredd>:<port>
* `COOLDNS_INFLUX_DB` Database name
* `COOLDNS_INFLUX_USER` User name
* `COOLDNS_INFLUX_PASS` Password

## Testing

use curl to test
---
curl --basic http://doof.ist.nicht.cool.:12345678@localhost:3000/nic/update\?hostname\=doof.ist.nicht.cool.\&myip\=192.168.45.200\&txt\=Doller%20geht%20nich
---
