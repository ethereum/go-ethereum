// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package api

const Bzz_JS = `
web3._extend({
	property: 'bzz',
	methods:
	[
		new web3._extend.Method({
			name: 'deposit',
			call: 'bzz_deposit',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'info',
			call: 'bzz_info',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'cash',
			call: 'bzz_cash',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'issue',
			call: 'bzz_issue',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'register',
			call: 'bzz_register',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Method({
			name: 'resolve',
			call: 'bzz_resolve',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'download',
			call: 'bzz_download',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'upload',
			call: 'bzz_upload',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'get',
			call: 'bzz_get',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'put',
			call: 'bzz_put',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'modify',
			call: 'bzz_modify',
			params: 4,
			inputFormatter: [null, null, null, null]
		})
	],
	properties:
	[
	]
});
`
