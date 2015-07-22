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

const Debug_JS = `
web3._extend({
	property: 'debug',
	methods:
	[
		new web3._extend.Method({
			name: 'printBlock',
			call: 'debug_printBlock',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getBlockRlp',
			call: 'debug_getBlockRlp',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'setHead',
			call: 'debug_setHead',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'processBlock',
			call: 'debug_processBlock',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'seedHash',
			call: 'debug_seedHash',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'dumpBlock',
			call: 'debug_dumpBlock',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'metrics',
			call: 'debug_metrics',
			params: 1,
			inputFormatter: [null]
		})
	],
	properties:
	[
	]
});
`
