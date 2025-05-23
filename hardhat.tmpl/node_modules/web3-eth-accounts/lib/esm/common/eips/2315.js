export default {
    name: 'EIP-2315',
    number: 2315,
    comment: 'Simple subroutines for the EVM',
    url: 'https://eips.ethereum.org/EIPS/eip-2315',
    status: 'Draft',
    minimumHardfork: 'istanbul',
    gasConfig: {},
    gasPrices: {
        beginsub: {
            v: 2,
            d: 'Base fee of the BEGINSUB opcode',
        },
        returnsub: {
            v: 5,
            d: 'Base fee of the RETURNSUB opcode',
        },
        jumpsub: {
            v: 10,
            d: 'Base fee of the JUMPSUB opcode',
        },
    },
    vm: {},
    pow: {},
};
//# sourceMappingURL=2315.js.map