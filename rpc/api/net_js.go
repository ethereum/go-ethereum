package api

const Net_JS = `
web3._extend({
	property: 'net',
	methods:
	[
		new web3._extend.Method({
			name: 'addPeer',
			call: 'net_addPeer',
			params: 1,
			inputFormatter: [null]
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'peers',
			getter: 'net_peers'
		}),
		new web3._extend.Property({
			name: 'version',
			getter: 'net_version'
		})
	]
});
`
