package api

const Personal_JS = `
web3.extend({
	property: 'personal',
	methods:
	[
		new web3.extend.Method({
			name: 'listAccounts',
			call: 'personal_listAccounts',
			params: 0,
			inputFormatter: [],
			outputFormatter: function(obj) { return obj; }
		}),
		new web3.extend.Method({
			name: 'newAccount',
			call: 'personal_newAccount',
			params: 1,
			inputFormatter: [web3.extend.formatters.formatInputString],
			outputFormatter: web3.extend.formatters.formatOutputString
		}),
		new web3.extend.Method({
			name: 'unlockAccount',
			call: 'personal_unlockAccount',
			params: 3,
			inputFormatter: [web3.extend.formatters.formatInputString,web3.extend.formatters.formatInputString,web3.extend.formatters.formatInputInt],
			outputFormatter: web3.extend.formatters.formatOutputBool
		})
	],
	properties:
	[
	]
});
`
