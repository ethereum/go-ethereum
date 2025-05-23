export default {
    name: 'sepolia',
    chainId: 11155111,
    networkId: 11155111,
    defaultHardfork: 'merge',
    consensus: {
        type: 'pow',
        algorithm: 'ethash',
        ethash: {},
    },
    comment: 'PoW test network to replace Ropsten',
    url: 'https://github.com/ethereum/go-ethereum/pull/23730',
    genesis: {
        timestamp: '0x6159af19',
        gasLimit: 30000000,
        difficulty: 131072,
        nonce: '0x0000000000000000',
        extraData: '0x5365706f6c69612c20417468656e732c204174746963612c2047726565636521',
    },
    hardforks: [
        {
            name: 'chainstart',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'homestead',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'tangerineWhistle',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'spuriousDragon',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'byzantium',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'constantinople',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'petersburg',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'istanbul',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'muirGlacier',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'berlin',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'london',
            block: 0,
            forkHash: '0xfe3366e7',
        },
        {
            '//_comment': 'The forkHash will remain same as mergeForkIdTransition is post merge, terminal block: https://sepolia.etherscan.io/block/1450408',
            name: 'merge',
            ttd: '17000000000000000',
            block: 1450409,
            forkHash: '0xfe3366e7',
        },
        {
            name: 'mergeForkIdTransition',
            block: 1735371,
            forkHash: '0xb96cbd13',
        },
        {
            name: 'shanghai',
            block: null,
            timestamp: '1677557088',
            forkHash: '0xf7f9bc08',
        },
    ],
    bootstrapNodes: [],
    dnsNetworks: [
        'enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.sepolia.ethdisco.net',
    ],
};
//# sourceMappingURL=sepolia.js.map