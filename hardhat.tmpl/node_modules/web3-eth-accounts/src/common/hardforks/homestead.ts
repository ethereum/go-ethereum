export default {
	name: 'homestead',
	comment: 'Homestead hardfork with protocol and network changes',
	url: 'https://eips.ethereum.org/EIPS/eip-606',
	status: 'Final',
	gasConfig: {},
	gasPrices: {
		delegatecall: {
			v: 40,
			d: 'Base fee of the DELEGATECALL opcode',
		},
	},
	vm: {},
	pow: {},
};
