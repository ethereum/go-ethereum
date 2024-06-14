"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isBlobEIP4844TxData = exports.isFeeMarketEIP1559TxData = exports.isAccessListEIP2930TxData = exports.isLegacyTxData = exports.isBlobEIP4844Tx = exports.isFeeMarketEIP1559Tx = exports.isAccessListEIP2930Tx = exports.isLegacyTx = exports.TransactionType = exports.isAccessList = exports.isAccessListBytes = exports.Capability = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
/**
 * Can be used in conjunction with {@link Transaction[TransactionType].supports}
 * to query on tx capabilities
 */
var Capability;
(function (Capability) {
    /**
     * Tx supports EIP-155 replay protection
     * See: [155](https://eips.ethereum.org/EIPS/eip-155) Replay Attack Protection EIP
     */
    Capability[Capability["EIP155ReplayProtection"] = 155] = "EIP155ReplayProtection";
    /**
     * Tx supports EIP-1559 gas fee market mechanism
     * See: [1559](https://eips.ethereum.org/EIPS/eip-1559) Fee Market EIP
     */
    Capability[Capability["EIP1559FeeMarket"] = 1559] = "EIP1559FeeMarket";
    /**
     * Tx is a typed transaction as defined in EIP-2718
     * See: [2718](https://eips.ethereum.org/EIPS/eip-2718) Transaction Type EIP
     */
    Capability[Capability["EIP2718TypedTransaction"] = 2718] = "EIP2718TypedTransaction";
    /**
     * Tx supports access list generation as defined in EIP-2930
     * See: [2930](https://eips.ethereum.org/EIPS/eip-2930) Access Lists EIP
     */
    Capability[Capability["EIP2930AccessLists"] = 2930] = "EIP2930AccessLists";
})(Capability = exports.Capability || (exports.Capability = {}));
function isAccessListBytes(input) {
    if (input.length === 0) {
        return true;
    }
    const firstItem = input[0];
    if (Array.isArray(firstItem)) {
        return true;
    }
    return false;
}
exports.isAccessListBytes = isAccessListBytes;
function isAccessList(input) {
    return !isAccessListBytes(input); // This is exactly the same method, except the output is negated.
}
exports.isAccessList = isAccessList;
/**
 * Encompassing type for all transaction types.
 */
var TransactionType;
(function (TransactionType) {
    TransactionType[TransactionType["Legacy"] = 0] = "Legacy";
    TransactionType[TransactionType["AccessListEIP2930"] = 1] = "AccessListEIP2930";
    TransactionType[TransactionType["FeeMarketEIP1559"] = 2] = "FeeMarketEIP1559";
    TransactionType[TransactionType["BlobEIP4844"] = 3] = "BlobEIP4844";
})(TransactionType = exports.TransactionType || (exports.TransactionType = {}));
function isLegacyTx(tx) {
    return tx.type === TransactionType.Legacy;
}
exports.isLegacyTx = isLegacyTx;
function isAccessListEIP2930Tx(tx) {
    return tx.type === TransactionType.AccessListEIP2930;
}
exports.isAccessListEIP2930Tx = isAccessListEIP2930Tx;
function isFeeMarketEIP1559Tx(tx) {
    return tx.type === TransactionType.FeeMarketEIP1559;
}
exports.isFeeMarketEIP1559Tx = isFeeMarketEIP1559Tx;
function isBlobEIP4844Tx(tx) {
    return tx.type === TransactionType.BlobEIP4844;
}
exports.isBlobEIP4844Tx = isBlobEIP4844Tx;
function isLegacyTxData(txData) {
    const txType = Number((0, ethereumjs_util_1.bytesToBigInt)((0, ethereumjs_util_1.toBytes)(txData.type)));
    return txType === TransactionType.Legacy;
}
exports.isLegacyTxData = isLegacyTxData;
function isAccessListEIP2930TxData(txData) {
    const txType = Number((0, ethereumjs_util_1.bytesToBigInt)((0, ethereumjs_util_1.toBytes)(txData.type)));
    return txType === TransactionType.AccessListEIP2930;
}
exports.isAccessListEIP2930TxData = isAccessListEIP2930TxData;
function isFeeMarketEIP1559TxData(txData) {
    const txType = Number((0, ethereumjs_util_1.bytesToBigInt)((0, ethereumjs_util_1.toBytes)(txData.type)));
    return txType === TransactionType.FeeMarketEIP1559;
}
exports.isFeeMarketEIP1559TxData = isFeeMarketEIP1559TxData;
function isBlobEIP4844TxData(txData) {
    const txType = Number((0, ethereumjs_util_1.bytesToBigInt)((0, ethereumjs_util_1.toBytes)(txData.type)));
    return txType === TransactionType.BlobEIP4844;
}
exports.isBlobEIP4844TxData = isBlobEIP4844TxData;
//# sourceMappingURL=types.js.map