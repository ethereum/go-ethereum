export default {
	name: 'goerli',
	chainId: 5,
	networkId: 5,
	defaultHardfork: 'merge',
	consensus: {
		type: 'poa',
		algorithm: 'clique',
		clique: {
			period: 15,
			epoch: 30000,
		},
	},
	comment: 'Cross-client PoA test network',
	url: 'https://github.com/goerli/testnet',
	genesis: {
		timestamp: '0x5c51a607',
		gasLimit: 10485760,
		difficulty: 1,
		nonce: '0x0000000000000000',
		extraData:
			'0x22466c6578692069732061207468696e6722202d204166726900000000000000e0a2bd4258d2768837baa26a28fe71dc079f84c70000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
	},
	hardforks: [
		{
			name: 'chainstart',
			block: 0,
			forkHash: '0xa3f5ab08',
		},
		{
			name: 'homestead',
			block: 0,
			forkHash: '0xa3f5ab08',
		},
		{
			name: 'tangerineWhistle',
			block: 0,
			forkHash: '0xa3f5ab08',
		},
		{
			name: 'spuriousDragon',
			block: 0,
			forkHash: '0xa3f5ab08',
		},
		{
			name: 'byzantium',
			block: 0,
			forkHash: '0xa3f5ab08',
		},
		{
			name: 'constantinople',
			block: 0,
			forkHash: '0xa3f5ab08',
		},
		{
			name: 'petersburg',
			block: 0,
			forkHash: '0xa3f5ab08',
		},
		{
			name: 'istanbul',
			block: 1561651,
			forkHash: '0xc25efa5c',
		},
		{
			name: 'berlin',
			block: 4460644,
			forkHash: '0x757a1c47',
		},
		{
			name: 'london',
			block: 5062605,
			forkHash: '0xb8c6299d',
		},
		{
			'//_comment':
				'The forkHash will remain same as mergeForkIdTransition is post merge, terminal block: https://goerli.etherscan.io/block/7382818',
			name: 'merge',
			ttd: '10790000',
			block: 7382819,
			forkHash: '0xb8c6299d',
		},
		{
			name: 'mergeForkIdTransition',
			block: null,
			forkHash: null,
		},
		{
			name: 'shanghai',
			block: null,
			forkHash: null,
		},
	],
	bootstrapNodes: [],
	dnsNetworks: [
		'enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.goerli.ethdisco.net',
	],
};
