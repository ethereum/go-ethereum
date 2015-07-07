package api

const Miner_JS = `
web3._extend({
	property: 'miner',
	methods:
	[
		new web3._extend.Method({
			name: 'start',
			call: 'miner_start',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'stop',
			call: 'miner_stop',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'setEtherbase',
			call: 'miner_setEtherbase',
			params: 1,
			inputFormatter: [web3._extend.formatters.formatInputInt],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
		new web3._extend.Method({
			name: 'setExtra',
			call: 'miner_setExtra',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'setGasPrice',
			call: 'miner_setGasPrice',
			params: 1,
			inputFormatter: [web3._extend.utils.fromDecial]
		}),
		new web3._extend.Method({
			name: 'startAutoDAG',
			call: 'miner_startAutoDAG',
			params: 0,
			inputFormatter: []
		}),
		new web3._extend.Method({
			name: 'stopAutoDAG',
			call: 'miner_stopAutoDAG',
			params: 0,
			inputFormatter: []
		}),
		new web3._extend.Method({
			name: 'makeDAG',
			call: 'miner_makeDAG',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputDefaultBlockNumberFormatter]
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'hashrate',
			getter: 'miner_hashrate',
			outputFormatter: web3._extend.utils.toDecimal
		})
	]
});
`
