package api

const TxPool_JS = `
web3._extend({
	property: 'txpool',
	methods:
	[
	],
	properties:
	[
		new web3._extend.Property({
			name: 'status',
			getter: 'txpool_status',
			outputFormatter: function(obj) { return obj; }
		})
	]
});
`
