export default {
	name: 'EIP-3074',
	number: 3074,
	comment: 'AUTH and AUTHCALL opcodes',
	url: 'https://eips.ethereum.org/EIPS/eip-3074',
	status: 'Review',
	minimumHardfork: 'london',
	gasConfig: {},
	gasPrices: {
		auth: {
			v: 3100,
			d: 'Gas cost of the AUTH opcode',
		},
		authcall: {
			v: 0,
			d: 'Gas cost of the AUTHCALL opcode',
		},
		authcallValueTransfer: {
			v: 6700,
			d: 'Paid for CALL when the value transfer is non-zero',
		},
	},
	vm: {},
	pow: {},
};
