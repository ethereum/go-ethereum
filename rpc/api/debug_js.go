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
			inputFormatter: [web3._extend.formatters.formatInputInt],
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Method({
			name: 'getBlockRlp',
			call: 'debug_getBlockRlp',
			params: 1,
			inputFormatter: [web3._extend.formatters.formatInputInt],
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Method({
			name: 'setHead',
			call: 'debug_setHead',
			params: 1,
			inputFormatter: [web3._extend.formatters.formatInputInt],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
		new web3._extend.Method({
			name: 'processBlock',
			call: 'debug_processBlock',
			params: 1,
			inputFormatter: [web3._extend.formatters.formatInputInt],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3._extend.Method({
			name: 'seedHash',
			call: 'debug_seedHash',
			params: 1,
			inputFormatter: [web3._extend.formatters.formatInputInt],
			outputFormatter: web3._extend.formatters.formatOutputString
		})		,
		new web3._extend.Method({
			name: 'dumpBlock',
			call: 'debug_dumpBlock',
			params: 1,
			inputFormatter: [web3._extend.formatters.formatInputInt],
			outputFormatter: function(obj) { return obj; }
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'metrics',
			getter: 'debug_metrics',
			outputFormatter: function(obj) { return obj; }
		})
	]
});
`
