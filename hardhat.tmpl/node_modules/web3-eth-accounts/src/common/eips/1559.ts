export default {
	name: 'EIP-1559',
	number: 1559,
	comment: 'Fee market change for ETH 1.0 chain',
	url: 'https://eips.ethereum.org/EIPS/eip-1559',
	status: 'Final',
	minimumHardfork: 'berlin',
	requiredEIPs: [2930],
	gasConfig: {
		baseFeeMaxChangeDenominator: {
			v: 8,
			d: 'Maximum base fee change denominator',
		},
		elasticityMultiplier: {
			v: 2,
			d: 'Maximum block gas target elasticity',
		},
		initialBaseFee: {
			v: 1000000000,
			d: 'Initial base fee on first EIP1559 block',
		},
	},
	gasPrices: {},
	vm: {},
	pow: {},
};
