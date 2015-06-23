package api

const Admin_JS = `
web3._extend({
	property: 'admin',
	methods:
	[
		new web3._extend.Method({
			name: 'addPeer',
			call: 'admin_addPeer',
			params: 1,
			inputFormatter: [web3._extend.utils.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
		new web3._extend.Method({
			name: 'exportChain',
			call: 'admin_exportChain',
			params: 1,
			inputFormatter: [web3._extend.utils.formatInputString],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3._extend.Method({
			name: 'importChain',
			call: 'admin_importChain',
			params: 1,
			inputFormatter: [web3._extend.utils.formatInputString],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3._extend.Method({
			name: 'verbosity',
			call: 'admin_verbosity',
			params: 1,
			inputFormatter: [web3._extend.utils.formatInputInt],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
		new web3._extend.Method({
			name: 'setSolc',
			call: 'admin_setSolc',
			params: 1,
			inputFormatter: [web3._extend.utils.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Method({
			name: 'startRPC',
			call: 'admin_startRPC',
			params: 4,
			inputFormatter: [web3._extend.utils.formatInputString,web3._extend.utils.formatInputInteger,web3._extend.utils.formatInputString,web3._extend.utils.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
		new web3._extend.Method({
			name: 'stopRPC',
			call: 'admin_stopRPC',
			params: 0,
			inputFormatter: [],
			outputFormatter: web3._extend.formatters.formatOutputBool
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo',
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Property({
			name: 'peers',
			getter: 'admin_peers',
			outputFormatter: function(obj) { return obj; }
		}),
		new web3._extend.Property({
			name: 'datadir',
			getter: 'admin_datadir',
			outputFormatter: web3._extend.formatters.formatOutputString
		}),
		new web3._extend.Property({
			name: 'chainSyncStatus',
			getter: 'admin_chainSyncStatus',
			outputFormatter: function(obj) { return obj; }
		})
	]
});
`
