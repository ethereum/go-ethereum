export default {
    name: 'EIP-2565',
    number: 2565,
    comment: 'ModExp gas cost',
    url: 'https://eips.ethereum.org/EIPS/eip-2565',
    status: 'Final',
    minimumHardfork: 'byzantium',
    gasConfig: {},
    gasPrices: {
        modexpGquaddivisor: {
            v: 3,
            d: 'Gquaddivisor from modexp precompile for gas calculation',
        },
    },
    vm: {},
    pow: {},
};
//# sourceMappingURL=2565.js.map