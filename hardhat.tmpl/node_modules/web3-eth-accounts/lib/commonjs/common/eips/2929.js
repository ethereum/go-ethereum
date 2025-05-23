"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    name: 'EIP-2929',
    comment: 'Gas cost increases for state access opcodes',
    url: 'https://eips.ethereum.org/EIPS/eip-2929',
    status: 'Final',
    minimumHardfork: 'chainstart',
    gasConfig: {},
    gasPrices: {
        coldsload: {
            v: 2100,
            d: 'Gas cost of the first read of storage from a given location (per transaction)',
        },
        coldaccountaccess: {
            v: 2600,
            d: 'Gas cost of the first read of a given address (per transaction)',
        },
        warmstorageread: {
            v: 100,
            d: "Gas cost of reading storage locations which have already loaded 'cold'",
        },
        sstoreCleanGasEIP2200: {
            v: 2900,
            d: 'Once per SSTORE operation from clean non-zero to something else',
        },
        sstoreNoopGasEIP2200: {
            v: 100,
            d: "Once per SSTORE operation if the value doesn't change",
        },
        sstoreDirtyGasEIP2200: {
            v: 100,
            d: 'Once per SSTORE operation if a dirty value is changed',
        },
        sstoreInitRefundEIP2200: {
            v: 19900,
            d: 'Once per SSTORE operation for resetting to the original zero value',
        },
        sstoreCleanRefundEIP2200: {
            v: 4900,
            d: 'Once per SSTORE operation for resetting to the original non-zero value',
        },
        call: {
            v: 0,
            d: 'Base fee of the CALL opcode',
        },
        callcode: {
            v: 0,
            d: 'Base fee of the CALLCODE opcode',
        },
        delegatecall: {
            v: 0,
            d: 'Base fee of the DELEGATECALL opcode',
        },
        staticcall: {
            v: 0,
            d: 'Base fee of the STATICCALL opcode',
        },
        balance: {
            v: 0,
            d: 'Base fee of the BALANCE opcode',
        },
        extcodesize: {
            v: 0,
            d: 'Base fee of the EXTCODESIZE opcode',
        },
        extcodecopy: {
            v: 0,
            d: 'Base fee of the EXTCODECOPY opcode',
        },
        extcodehash: {
            v: 0,
            d: 'Base fee of the EXTCODEHASH opcode',
        },
        sload: {
            v: 0,
            d: 'Base fee of the SLOAD opcode',
        },
        sstore: {
            v: 0,
            d: 'Base fee of the SSTORE opcode',
        },
    },
    vm: {},
    pow: {},
};
//# sourceMappingURL=2929.js.map