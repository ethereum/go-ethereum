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
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'exportChain',
			call: 'admin_exportChain',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'importChain',
			call: 'admin_importChain',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'verbosity',
			call: 'admin_verbosity',
			params: 1,
			inputFormatter: [web3._extend.utils.toDecimal]
		}),
		new web3._extend.Method({
			name: 'setSolc',
			call: 'admin_setSolc',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'startRPC',
			call: 'admin_startRPC',
			params: 4,
			inputFormatter: [null, web3._extend.utils.toDecimal, null, null]
		}),
		new web3._extend.Method({
			name: 'stopRPC',
			call: 'admin_stopRPC',
			params: 0,
			inputFormatter: []
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo'
		}),
		new web3._extend.Property({
			name: 'peers',
			getter: 'admin_peers'
		}),
		new web3._extend.Property({
			name: 'datadir',
			getter: 'admin_datadir'
		}),
		new web3._extend.Property({
			name: 'chainSyncStatus',
			getter: 'admin_chainSyncStatus'
		})
	]
});
`
