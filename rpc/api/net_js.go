// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package api

const Net_JS = `
web3._extend({
	property: 'net',
	methods:
	[
		new web3._extend.Method({
			name: 'addPeer',
			call: 'net_addPeer',
			params: 1,
			inputFormatter: [null]
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'version',
			getter: 'net_version'
		})
	]
});
`
