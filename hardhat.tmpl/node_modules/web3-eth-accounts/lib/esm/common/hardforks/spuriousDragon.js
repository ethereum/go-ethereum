export default {
    name: 'spuriousDragon',
    comment: 'HF with EIPs for simple replay attack protection, EXP cost increase, state trie clearing, contract code size limit',
    url: 'https://eips.ethereum.org/EIPS/eip-607',
    status: 'Final',
    gasConfig: {},
    gasPrices: {
        expByte: {
            v: 50,
            d: 'Times ceil(log256(exponent)) for the EXP instruction',
        },
    },
    vm: {
        maxCodeSize: {
            v: 24576,
            d: 'Maximum length of contract code',
        },
    },
    pow: {},
};
//# sourceMappingURL=spuriousDragon.js.map