package api

const Net_JS = `
web3._extend({
	property: 'network',
	methods:
	[
		new web3._extend.Method({
			name: 'addPeer',
			call: 'net_addPeer',
			params: 1,
			inputFormatter: [web3._extend.utils.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
		new web3._extend.Method({
			name: 'id',
			call: 'net_id',
			params: 0,
			inputFormatter: [],
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Method({
			name: 'getPeerCount',
			call: 'net_peerCount',
			params: 0,
			inputFormatter: [],
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Method({
			name: 'peers',
			call: 'net_peers',
			params: 0,
			inputFormatter: [],
			outputFormatter: function(obj) { return obj; }
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'listening',
			getter: 'net_listening',
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
		new web3._extend.Property({
			name: 'peerCount',
			getter: 'net_peerCount',
			outputFormatter: web3._extend.utils.toDecimal
		})
	]
});
`
