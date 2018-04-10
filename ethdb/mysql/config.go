// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package mysql

import (
	"fmt"
)

type Config struct {
	Protocol             string `toml:",omitempty"`
	Address              string `toml:",omitempty"`
	Port                 string `toml:",omitempty"`
	User                 string `toml:",omitempty"`
	Password             string `toml:",omitempty"`
	Database             string `toml:",omitempty"`
	AllowNativePasswords bool   `toml:",omitempty"`
}

var DefaultConfig = Config{
	Protocol:             "tcp",
	Address:              "localhost",
	Password:             "my-pw",
	Port:                 "3306",
	User:                 "root",
	Database:             "db0",
	AllowNativePasswords: true,
}

func (o *Config) String() string {
	return fmt.Sprintf(
		"%s:%s@%s(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local&allowNativePasswords=%v",
		o.User,
		o.Password,
		o.Protocol,
		o.Address,
		o.Port,
		o.Database,
		o.AllowNativePasswords)
}
