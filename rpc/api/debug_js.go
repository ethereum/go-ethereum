package api

const Debug_JS = `
web3.extend({
	property: 'debug',
	methods:
	[
		new web3.extend.Method({
			name: 'printBlock',
			call: 'debug_printBlock',
			params: 1,
			inputFormatter: [web3.extend.formatters.formatInputInt],
			outputFormatter: web3.extend.formatters.formatOutputString
		}),
		new web3.extend.Method({
			name: 'getBlockRlp',
			call: 'debug_getBlockRlp',
			params: 1,
			inputFormatter: [web3.extend.formatters.formatInputInt],
			outputFormatter: web3.extend.formatters.formatOutputString
		}),
		new web3.extend.Method({
			name: 'setHead',
			call: 'debug_setHead',
			params: 1,
			inputFormatter: [web3.extend.formatters.formatInputInt],
			outputFormatter: web3.extend.formatters.formatOutputBool
		}),
		new web3.extend.Method({
			name: 'processBlock',
			call: 'debug_processBlock',
			params: 1,
			inputFormatter: [web3.extend.formatters.formatInputInt],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3.extend.Method({
			name: 'seedHash',
			call: 'debug_seedHash',
			params: 1,
			inputFormatter: [web3.extend.formatters.formatInputInt],
			outputFormatter: web3.extend.formatters.formatOutputString
		})
	],
	properties:
	[
	]
});
`
