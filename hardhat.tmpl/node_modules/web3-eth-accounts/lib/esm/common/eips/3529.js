export default {
    name: 'EIP-3529',
    comment: 'Reduction in refunds',
    url: 'https://eips.ethereum.org/EIPS/eip-3529',
    status: 'Final',
    minimumHardfork: 'berlin',
    requiredEIPs: [2929],
    gasConfig: {
        maxRefundQuotient: {
            v: 5,
            d: 'Maximum refund quotient; max tx refund is min(tx.gasUsed/maxRefundQuotient, tx.gasRefund)',
        },
    },
    gasPrices: {
        selfdestructRefund: {
            v: 0,
            d: 'Refunded following a selfdestruct operation',
        },
        sstoreClearRefundEIP2200: {
            v: 4800,
            d: 'Once per SSTORE operation for clearing an originally existing storage slot',
        },
    },
    vm: {},
    pow: {},
};
//# sourceMappingURL=3529.js.map