package api

const Personal_JS = `
web3._extend({
	property: 'personal',
	methods:
	[
		new web3._extend.Method({
			name: 'newAccount',
			call: 'personal_newAccount',
			params: 1,
			inputFormatter: [null],
			outputFormatter: web3._extend.utils.toAddress
		}),
		new web3._extend.Method({
			name: 'unlockAccount',
			call: 'personal_unlockAccount',
			params: 3,
			inputFormatter: [null, null, null]
		})
	],
	properties:
	[
		new web3._extend.Property({
			name: 'listAccounts',
			getter: 'personal_listAccounts'
		})
	]
});
`
