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
	"code.google.com/p/go.crypto/scrypt"
	"crypto/subtle"
	"errors"
	"os"
	"unicode/utf8"
)

const (
	ScryptN      int = 16384
	Scryptr      int = 8
	Scryptp      int = 1
	ScryptKeyLen int = 32
)

var (
	AuthConstraintsNotMet error = errors.New("Constraints do not apply")
)

type Auth struct {
	Name string
	Salt []byte
	Key  []byte
}

func checkConstraints(name, secret string) bool {
	// Name string shall not be empty
	if name == "" {
		return false
	}
	// secret has to be longer then 8 unicode chars
	if utf8.RuneCountInString(secret) < 8 {
		return false
	}
	return true
}

// New Auth takes a name and secret of type string and generated an Auth out
// of them. An 8 Byte salt is randomly generated and added to the Auth.
//
// Some standard input constraints are applied:
//  *No Empty strings
//  *Minimum of 8 unicode Runes for the secret (more is recomended)
//
// We use /dev/urandom as a rand source and if you wish to argue about this,
// argue with a knife because I am not intersted. Oh, and don't trust this on
// *BSD because they got it all wrong.
func NewAuth(name, secret string) (*Auth, error) {
	if !checkConstraints(name, secret) {
		return nil, AuthConstraintsNotMet
	}
	rand, err := os.Open("/dev/urandom")
	if err != nil {
		return nil, err
	}

	salt := make([]byte, 8)
	c, err := rand.Read(salt)
	if err != nil || c < 8 {
		return nil, err
	}

	passname := []byte(name + secret)
	key, err := scrypt.Key(passname, salt, ScryptN, Scryptr, Scryptp, ScryptKeyLen)
	if err != nil {
		return nil, err
	}

	return &Auth{
		Name: name,
		Salt: salt,
		Key:  key,
	}, nil
}

// CheckAuth Checks if a name, secret touple is identical to the one used for
// the initial key. We return ok=true if the touple matches, else ok=false.
//
// Well let's just hope this happens in constant time or something
func (a *Auth) CheckAuth(name, secret string) (bool, error) {
	passname := []byte(name + secret)
	salt := a.Salt
	sKey := a.Key
	key, err := scrypt.Key(passname, salt, ScryptN, Scryptr, Scryptp, ScryptKeyLen)
	ok := subtle.ConstantTimeCompare(sKey, key)
	return ok == 1, err
}
