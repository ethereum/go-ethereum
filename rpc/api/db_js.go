package api

const Db_JS = `
web3._extend({
	property: 'db',
	methods:
	[
		new web3._extend.Method({
			name: 'getString',
			call: 'db_getString',
			params: 2,
			inputFormatter: [web3._extend.formatters.formatInputString, web3._extend.formatters.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Method({
			name: 'putString',
			call: 'db_putString',
			params: 3,
			inputFormatter: [web3._extend.formatters.formatInputString, web3._extend.formatters.formatInputString, web3._extend.formatters.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
		new web3._extend.Method({
			name: 'getHex',
			call: 'db_getHex',
			params: 2,
			inputFormatter: [web3._extend.formatters.formatInputString, web3._extend.formatters.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Method({
			name: 'putHex',
			call: 'db_putHex',
			params: 3,
			inputFormatter: [web3._extend.formatters.formatInputString, web3._extend.formatters.formatInputString, web3._extend.formatters.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
	],
	properties:
	[
	]
});
`
