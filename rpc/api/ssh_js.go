package api

const Shh_JS = `
web3._extend({
	property: 'shh',
	methods:
	[
		new web3._extend.Method({
			name: 'post',
			call: 'shh_post',
			params: 6,
			inputFormatter: [web3._extend.formatters.formatInputString,
							  web3._extend.formatters.formatInputString,
							  web3._extend.formatters.formatInputString,
							,
							, web3._extend.formatters.formatInputInt
							, web3._extend.formatters.formatInputInt],
			outputFormatter: web3._extend.formatters.formatOutputBool
		}),
	],
	properties:
	[
		new web3._extend.Property({
			name: 'version',
			getter: 'shh_version',
			outputFormatter: web3._extend.formatters.formatOutputInt
		})
	]
});
`
