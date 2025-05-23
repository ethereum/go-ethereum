export default {
	name: 'tangerineWhistle',
	comment: 'Hardfork with gas cost changes for IO-heavy operations',
	url: 'https://eips.ethereum.org/EIPS/eip-608',
	status: 'Final',
	gasConfig: {},
	gasPrices: {
		sload: {
			v: 200,
			d: 'Once per SLOAD operation',
		},
		call: {
			v: 700,
			d: 'Once per CALL operation & message call transaction',
		},
		extcodesize: {
			v: 700,
			d: 'Base fee of the EXTCODESIZE opcode',
		},
		extcodecopy: {
			v: 700,
			d: 'Base fee of the EXTCODECOPY opcode',
		},
		balance: {
			v: 400,
			d: 'Base fee of the BALANCE opcode',
		},
		delegatecall: {
			v: 700,
			d: 'Base fee of the DELEGATECALL opcode',
		},
		callcode: {
			v: 700,
			d: 'Base fee of the CALLCODE opcode',
		},
		selfdestruct: {
			v: 5000,
			d: 'Base fee of the SELFDESTRUCT opcode',
		},
	},
	vm: {},
	pow: {},
};
