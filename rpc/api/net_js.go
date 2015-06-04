package api

const Net_JS = `
web3.extend({
	property: 'miner',
	methods:
	[
		new web3.extend.Method({
			name: 'version',
			call: 'net_version',
			params: 0,
			inputFormatter: [],
			outputFormatter: web3.extend.formatters.formatOutputString
		}),
		new web3.extend.Method({
			name: 'stop',
			call: 'net_getPeerCount',
			params: 0,
			inputFormatter: [],
			outputFormatter: web3.extend.formatters.formatOutputString
		}),
		new web3.extend.Method({
			name: 'stop',
			call: 'miner_stop',
			params: 0,
			inputFormatter: [],
			outputFormatter: web3.extend.formatters.formatOutputBool
		})
	],
	properties:
	[
		new web3.extend.Property({
			name: 'listening',
			getter: 'net_listening',
			outputFormatter: web3.extend.formatters.formatOutputBool
		}),
		new web3.extend.Property({
			name: 'peerCount',
			getter: 'net_getPeerCount',
			outputFormatter: web3.extend.utils.toDecimal
		})
	]
});
`
