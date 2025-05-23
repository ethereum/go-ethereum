export default {
	name: 'muirGlacier',
	comment: 'HF to delay the difficulty bomb',
	url: 'https://eips.ethereum.org/EIPS/eip-2384',
	status: 'Final',
	gasConfig: {},
	gasPrices: {},
	vm: {},
	pow: {
		difficultyBombDelay: {
			v: 9000000,
			d: 'the amount of blocks to delay the difficulty bomb with',
		},
	},
};
