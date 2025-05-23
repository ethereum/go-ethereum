export default {
	name: 'EIP-2930',
	comment: 'Optional access lists',
	url: 'https://eips.ethereum.org/EIPS/eip-2930',
	status: 'Final',
	minimumHardfork: 'istanbul',
	requiredEIPs: [2718, 2929],
	gasConfig: {},
	gasPrices: {
		accessListStorageKeyCost: {
			v: 1900,
			d: 'Gas cost per storage key in an Access List transaction',
		},
		accessListAddressCost: {
			v: 2400,
			d: 'Gas cost per storage key in an Access List transaction',
		},
	},
	vm: {},
	pow: {},
};
