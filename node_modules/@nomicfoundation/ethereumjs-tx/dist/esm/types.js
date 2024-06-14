import { bytesToBigInt, toBytes } from '@nomicfoundation/ethereumjs-util';
/**
 * Can be used in conjunction with {@link Transaction[TransactionType].supports}
 * to query on tx capabilities
 */
export var Capability;
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
})(Capability || (Capability = {}));
export function isAccessListBytes(input) {
    if (input.length === 0) {
        return true;
    }
    const firstItem = input[0];
    if (Array.isArray(firstItem)) {
        return true;
    }
    return false;
}
export function isAccessList(input) {
    return !isAccessListBytes(input); // This is exactly the same method, except the output is negated.
}
/**
 * Encompassing type for all transaction types.
 */
export var TransactionType;
(function (TransactionType) {
    TransactionType[TransactionType["Legacy"] = 0] = "Legacy";
    TransactionType[TransactionType["AccessListEIP2930"] = 1] = "AccessListEIP2930";
    TransactionType[TransactionType["FeeMarketEIP1559"] = 2] = "FeeMarketEIP1559";
    TransactionType[TransactionType["BlobEIP4844"] = 3] = "BlobEIP4844";
})(TransactionType || (TransactionType = {}));
export function isLegacyTx(tx) {
    return tx.type === TransactionType.Legacy;
}
export function isAccessListEIP2930Tx(tx) {
    return tx.type === TransactionType.AccessListEIP2930;
}
export function isFeeMarketEIP1559Tx(tx) {
    return tx.type === TransactionType.FeeMarketEIP1559;
}
export function isBlobEIP4844Tx(tx) {
    return tx.type === TransactionType.BlobEIP4844;
}
export function isLegacyTxData(txData) {
    const txType = Number(bytesToBigInt(toBytes(txData.type)));
    return txType === TransactionType.Legacy;
}
export function isAccessListEIP2930TxData(txData) {
    const txType = Number(bytesToBigInt(toBytes(txData.type)));
    return txType === TransactionType.AccessListEIP2930;
}
export function isFeeMarketEIP1559TxData(txData) {
    const txType = Number(bytesToBigInt(toBytes(txData.type)));
    return txType === TransactionType.FeeMarketEIP1559;
}
export function isBlobEIP4844TxData(txData) {
    const txType = Number(bytesToBigInt(toBytes(txData.type)));
    return txType === TransactionType.BlobEIP4844;
}
//# sourceMappingURL=types.js.map