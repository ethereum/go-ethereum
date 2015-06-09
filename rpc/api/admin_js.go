package api

const Admin_JS = `
web3.extend({
	property: 'admin',
	methods:
	[
		new web3.extend.Method({
			name: 'addPeer',
			call: 'admin_addPeer',
			params: 1,
			inputFormatter: [web3.extend.utils.formatInputString],
			outputFormatter: web3.extend.formatters.formatOutputBool
		}),
		new web3.extend.Method({
			name: 'peers',
			call: 'admin_peers',
			params: 0,
			inputFormatter: [],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3.extend.Method({
			name: 'exportChain',
			call: 'admin_exportChain',
			params: 1,
			inputFormatter: [web3.extend.utils.formatInputString],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3.extend.Method({
			name: 'importChain',
			call: 'admin_importChain',
			params: 1,
			inputFormatter: [web3.extend.utils.formatInputString],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3.extend.Method({
			name: 'verbosity',
			call: 'admin_verbosity',
			params: 1,
			inputFormatter: [web3.extend.utils.formatInputInt],
			outputFormatter: web3.extend.formatters.formatOutputBool
		}),
		new web3.extend.Method({
			name: 'syncStatus',
			call: 'admin_syncStatus',
			params: 1,
			inputFormatter: [web3.extend.utils.formatInputInt],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3.extend.Method({
			name: 'setSolc',
			call: 'admin_setSolc',
			params: 1,
			inputFormatter: [web3.extend.utils.formatInputString],
			outputFormatter: web3.extend.formatters.formatOutputString
		})
	],
	properties:
	[
		new web3.extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo',
			outputFormatter: web3.extend.formatters.formatOutputString
		})
	]
});
`
