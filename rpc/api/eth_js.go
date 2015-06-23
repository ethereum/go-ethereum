package api

// JS api provided by web3.js
// eth_sign not standard

const Eth_JS = `
web3._extend({
	property: 'eth',
	methods:
	[
		new web3._extend.Method({
			name: 'sign',
			call: 'eth_sign',
			params: 2,
			inputFormatter: [web3._extend.formatters.formatInputString,web3._extend.formatters.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputString
		})
	]
});
`
