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

const Chequebook_JS = `
web3._extend({
  property: 'chequebook',
  methods:
  [
    new web3._extend.Method({
      name: 'deposit',
      call: 'chequebook_deposit',
      params: 1,
      inputFormatter: [null]
    }),
    new web3._extend.Method({
      name: 'info',
      call: 'chequebook_info',
      params: 1,
      inputFormatter: [null]
    }),
    new web3._extend.Method({
      name: 'cash',
      call: 'chequebook_cash',
      params: 1,
      inputFormatter: [null]
    }),
    new web3._extend.Method({
      name: 'issue',
      call: 'chequebook_issue',
      params: 2,
      inputFormatter: [null, null]
    }),
  ]
});
`
