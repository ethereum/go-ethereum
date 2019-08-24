// Copyright 2019 The go-ethereum Authors
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
	"net"
	"time"

	"github.com/apilayer/freegeoip"
)

// geoDBInfo contains all the geographical information we could extract based on an IP
// address.
type geoDBInfo struct {
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

// geoLocation contains geographical information.
type geoLocation struct {
	Country   string  `json:"country,omitempty"`
	City      string  `json:"city,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

// geoDB represents a geoip database that can be queried for IP to geographical
// information conversions.
type geoDB struct {
	geodb *freegeoip.DB
}

// Open creates a new geoip database with an up-to-date database from the internet.
func openGeoDB() (*geoDB, error) {
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
	return &geoDB{geodb: db}, nil
}

// Close terminates the database background updater.
func (db *geoDB) close() error {
	db.geodb.Close()
	return nil
}

// Lookup converts an IP address to a geographical location.
func (db *geoDB) lookup(ip net.IP) *geoDBInfo {
	result := new(geoDBInfo)
	db.geodb.Lookup(ip, result)
	return result
}

// Location retrieves the geographical location of the given IP address.
func (db *geoDB) location(ip string) *geoLocation {
	location := db.lookup(net.ParseIP(ip))
	return &geoLocation{
		Country:   location.Country.Names.English,
		City:      location.City.Names.English,
		Latitude:  location.Location.Latitude,
		Longitude: location.Location.Longitude,
	}
}
