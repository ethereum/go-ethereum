"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    name: 'mainnet',
    chainId: 1,
    networkId: 1,
    defaultHardfork: 'merge',
    consensus: {
        type: 'pow',
        algorithm: 'ethash',
        ethash: {},
    },
    comment: 'The Ethereum main chain',
    url: 'https://ethstats.net/',
    genesis: {
        gasLimit: 5000,
        difficulty: 17179869184,
        nonce: '0x0000000000000042',
        extraData: '0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa',
    },
    hardforks: [
        {
            name: 'chainstart',
            block: 0,
            forkHash: '0xfc64ec04',
        },
        {
            name: 'homestead',
            block: 1150000,
            forkHash: '0x97c2c34c',
        },
        {
            name: 'dao',
            block: 1920000,
            forkHash: '0x91d1f948',
        },
        {
            name: 'tangerineWhistle',
            block: 2463000,
            forkHash: '0x7a64da13',
        },
        {
            name: 'spuriousDragon',
            block: 2675000,
            forkHash: '0x3edd5b10',
        },
        {
            name: 'byzantium',
            block: 4370000,
            forkHash: '0xa00bc324',
        },
        {
            name: 'constantinople',
            block: 7280000,
            forkHash: '0x668db0af',
        },
        {
            name: 'petersburg',
            block: 7280000,
            forkHash: '0x668db0af',
        },
        {
            name: 'istanbul',
            block: 9069000,
            forkHash: '0x879d6e30',
        },
        {
            name: 'muirGlacier',
            block: 9200000,
            forkHash: '0xe029e991',
        },
        {
            name: 'berlin',
            block: 12244000,
            forkHash: '0x0eb440f6',
        },
        {
            name: 'london',
            block: 12965000,
            forkHash: '0xb715077d',
        },
        {
            name: 'arrowGlacier',
            block: 13773000,
            forkHash: '0x20c327fc',
        },
        {
            name: 'grayGlacier',
            block: 15050000,
            forkHash: '0xf0afd0e3',
        },
        {
            '//_comment': 'The forkHash will remain same as mergeForkIdTransition is post merge, terminal block: https://etherscan.io/block/15537393',
            name: 'merge',
            ttd: '58750000000000000000000',
            block: 15537394,
            forkHash: '0xf0afd0e3',
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
        'enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@all.mainnet.ethdisco.net',
    ],
};
//# sourceMappingURL=mainnet.js.map