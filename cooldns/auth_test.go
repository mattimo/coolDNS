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
	"testing"
)

type authCheckTest struct {
	Name, Secret   string
	CName, CSecret string
	Ok             bool
	FailNew        bool
}

var authchecktests = []authCheckTest{
	{"TestUser", "TestPass", "TestUser", "TestPass", true, false},
	{"TestUser\n", "TestPass", "TestUser", "TestPass", false, false},
	{"TestUser\n\t", "TestPass\t\n", "TestUser\n\t", "TestPass\t\n", true, false},
	{"TestUseröäüöäüöäüöäüöäüöäüöü^t", "TestPass", "TestUseröäüöäüöäüöäüöäüöäüöü^t", "TestPass", true, false},
	{"TestUser\n\t", "TestPass\t\n", "TestUser\n\t", "TestPass\t", false, false},
	{"TestUser\n\t", "TestPass\t\n", "TestUser\n\t", "", false, false},
	{"", "", "", "", false, true},
}

func TestCheckAuth(t *testing.T) {
	for _, act := range authchecktests {
		a, err := NewAuth(act.Name, act.Secret)
		if err != nil {
			if !act.FailNew {
				t.Error("NewAuth Returned Error: ", err)
			}
			continue
		}
		ok, err := a.CheckAuth(act.CName, act.CSecret)
		if err != nil {
			t.Error("AuchCheck Returned Error:", err)
			continue
		}
		if ok != act.Ok {
			if act.Ok {
				t.Errorf("CheckAuth should not have failed. key: %X\n", a.Key)
			} else {
				t.Errorf("CheckAuth should have failed. Key: %X\n", a.Key)
			}
		}
	}
}

type authNewTests struct {
	Name, Secret   string
	ConstraintFail bool
}

var authnewtests = []authNewTests{
	{"totally.new.domain.", "12345678", false},
	// This should make clear that we know unicode
	{"totally.new.domain.", "1234567löäöpüöäöü??\n&瞬ಠ_ಠ", false},
	// Should fail becasue password is to short (3 runes)
	{"totally.new.domain.", "瞬ಠ_ಠ", true},
	{"totally.new.domain.", "1234567", true},
	{"", "123456789", true},
	{"", "", true},
}

func TestNewAuth(t *testing.T) {
	for _, act := range authnewtests {
		_, err := NewAuth(act.Name, act.Secret)
		if err != nil && err != AuthConstraintsNotMet {
			t.Errorf("Unexpected Error:", err)
		}
		if err != AuthConstraintsNotMet && act.ConstraintFail {
			t.Errorf("Should have failed: %v, Err: %v, len:%d\n", act, err, len(act.Secret))
		}
		if err == AuthConstraintsNotMet && !act.ConstraintFail {
			t.Errorf("Should not Have failed: %v, Err: %v, len:%d\n", act, err, len(act.Secret))
		}
	}
}
