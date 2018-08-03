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

package dashboard

import (
	"github.com/apilayer/freegeoip"
	"time"
	"net"
)
// Package geoip contains utility methods for converting IPs to geographical data.

// GeoDBInfo contains all the geographical information we could extract based on an IP
// address.
type GeoDBInfo struct {
	Country struct {
		Names struct {
			English string `maxminddb:"en" json:"en,omitempty"`
		} `maxminddb:"names" json:"names,omitempty"`
	} `maxminddb:"country" json:"country,omitempty"`
	City struct {
		Names struct {
			English string `maxminddb:"en" json:"en,omitempty"`
		} `maxminddb:"names" json:"names,omitempty"`
	} `maxminddb:"city" json:"city,omitempty"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude" json:"latitude,omitempty"`
		Longitude float64 `maxminddb:"longitude" json:"longitude,omitempty"`
	} `maxminddb:"location" json:"location,omitempty"`
}

// GeoDB represents a geoip database that can be queried for IP to geographical
// information conversions.
type GeoDB struct {
	geodb *freegeoip.DB
}

// Open creats a new geoip database with an up-to-date database from the internet.
func OpenGeoDB() (*GeoDB, error) {
	// Initiate a geoip database to cross reference locations
	db, err := freegeoip.OpenURL(freegeoip.MaxMindDB, 24*time.Hour, time.Hour)
	if err != nil {
		return nil, err
	}
	// Wait until the database is updated to the latest data
	select {
	case <-db.NotifyOpen():
	case err := <-db.NotifyError():
		return nil, err
	}
	// Assemble and return our custom wrapper
	return &GeoDB{geodb: db}, nil
}

// Close terminates the database background updater.
func (db *GeoDB) Close() error {
	db.geodb.Close()
	return nil
}

// Lookup converts an IP address to a geographical location.
func (db *GeoDB) Lookup(ip net.IP) *GeoDBInfo {
	result := new(GeoDBInfo)
	db.geodb.Lookup(ip, result)
	return result
}
