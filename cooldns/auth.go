package cooldns

import (
	"code.google.com/p/go.crypto/scrypt"
	"os"
	"crypto/subtle"
)

const (
	ScryptN int = 16384
	Scryptr int = 8
	Scryptp int = 1
	ScryptKeyLen int = 32
)

type Auth struct {
	Name	string
	Salt	[]byte
	Key	[]byte
}

func NewAuth(name, secret string) (*Auth, error) {
	rand, err := os.Open("/dev/urandom")
	if err != nil {
		return nil, err
	}

	salt := make([]byte, 8)
	c, err := rand.Read(salt)
	if err != nil || c < 8 {
		return nil, err
	}

	passname := []byte(name+secret)
	key, err := scrypt.Key(passname, salt, ScryptN, Scryptr, Scryptp, ScryptKeyLen)
	if err != nil {
		return nil, err
	}

	return &Auth{
		Name: name,
		Salt: salt,
		Key: key,
	}, nil
}

// Well let's just hope this happens in constant time or something
func (a *Auth) CheckAuth(name, secret string) (bool, error) {
	passname := []byte(name+secret)
	salt := a.Salt
	sKey := a.Key
	key, err := scrypt.Key(passname, salt, ScryptN, Scryptr, Scryptp, ScryptKeyLen)
	ok := subtle.ConstantTimeCompare(sKey, key)
	return ok == 1, err
}


