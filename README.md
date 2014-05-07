# coolDNS

Fully Qualified Cool Domain Names

## Config

* `COOLDNS_RC_PUB` The reCAPTCHA public Key
* `COOLDNS_RC_PRIV` The reCAPTCHA private Key

* `COOLDNS_SUFFIX` The cool dns domain suffix

## Testing

use curl to test
---
curl --basic http://doof.ist.nicht.cool.:12345678@localhost:3000/nic/update\?hostname\=doof.ist.nicht.cool.\&myip\=192.168.45.200\&txt\=Doller%20geht%20nich
---
