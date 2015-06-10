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
			name: 'peers',
			call: 'admin_peers',
			params: 0,
			inputFormatter: [],
			outputFormatter: function(obj) { return obj; }
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
			name: 'syncStatus',
			call: 'admin_syncStatus',
			params: 1,
			inputFormatter: [web3._extend.utils.formatInputInt],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3._extend.Method({
			name: 'setSolc',
			call: 'admin_setSolc',
			params: 1,
			inputFormatter: [web3._extend.utils.formatInputString],
			outputFormatter: web3._extend.formatters.formatOutputString
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo',
			outputFormatter: web3._extend.formatters.formatOutputString
		})
	]
});
`
