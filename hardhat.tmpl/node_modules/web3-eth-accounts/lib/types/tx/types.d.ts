import type { HexString, Numbers } from 'web3-types';
import type { Common } from '../common/common.js';
import type { Uint8ArrayLike, PrefixedHexString } from '../common/types';
import { Address } from './address.js';
/**
 * Can be used in conjunction with {@link Transaction.supports}
 * to query on tx capabilities
 */
export declare enum Capability {
    /**
     * Tx supports EIP-155 replay protection
     * See: [155](https://eips.ethereum.org/EIPS/eip-155) Replay Attack Protection EIP
     */
    EIP155ReplayProtection = 155,
    /**
     * Tx supports EIP-1559 gas fee market mechanism
     * See: [1559](https://eips.ethereum.org/EIPS/eip-1559) Fee Market EIP
     */
    EIP1559FeeMarket = 1559,
    /**
     * Tx is a typed transaction as defined in EIP-2718
     * See: [2718](https://eips.ethereum.org/EIPS/eip-2718) Transaction Type EIP
     */
    EIP2718TypedTransaction = 2718,
    /**
     * Tx supports access list generation as defined in EIP-2930
     * See: [2930](https://eips.ethereum.org/EIPS/eip-2930) Access Lists EIP
     */
    EIP2930AccessLists = 2930
}
/**
 * The options for initializing a {@link Transaction}.
 */
export interface TxOptions {
    /**
     * A {@link Common} object defining the chain and hardfork for the transaction.
     *
     * Object will be internally copied so that tx behavior don't incidentally
     * change on future HF changes.
     *
     * Default: {@link Common} object set to `mainnet` and the default hardfork as defined in the {@link Common} class.
     *
     * Current default hardfork: `istanbul`
     */
    common?: Common;
    /**
     * A transaction object by default gets frozen along initialization. This gives you
     * strong additional security guarantees on the consistency of the tx parameters.
     * It also enables tx hash caching when the `hash()` method is called multiple times.
     *
     * If you need to deactivate the tx freeze - e.g. because you want to subclass tx and
     * add additional properties - it is strongly encouraged that you do the freeze yourself
     * within your code instead.
     *
     * Default: true
     */
    freeze?: boolean;
    /**
     * Allows unlimited contract code-size init while debugging. This (partially) disables EIP-3860.
     * Gas cost for initcode size analysis will still be charged. Use with caution.
     */
    allowUnlimitedInitCodeSize?: boolean;
}
export type AccessListItem = {
    address: PrefixedHexString;
    storageKeys: PrefixedHexString[];
};
export type AccessListUint8ArrayItem = [Uint8Array, Uint8Array[]];
export type AccessListUint8Array = AccessListUint8ArrayItem[];
export type AccessList = AccessListItem[];
export declare function isAccessListUint8Array(input: AccessListUint8Array | AccessList): input is AccessListUint8Array;
export declare function isAccessList(input: AccessListUint8Array | AccessList): input is AccessList;
export interface ECDSASignature {
    v: bigint;
    r: Uint8Array;
    s: Uint8Array;
}
/**
 * Legacy {@link Transaction} Data
 */
export type TxData = {
    /**
     * The transaction's nonce.
     */
    nonce?: Numbers | Uint8Array;
    /**
     * The transaction's gas price.
     */
    gasPrice?: Numbers | Uint8Array | null;
    /**
     * The transaction's gas limit.
     */
    gasLimit?: Numbers | Uint8Array;
    /**
     * The transaction's the address is sent to.
     */
    to?: Address | Uint8Array | HexString;
    /**
     * The amount of Ether sent.
     */
    value?: Numbers | Uint8Array;
    /**
     * This will contain the data of the message or the init of a contract.
     */
    data?: Uint8ArrayLike;
    /**
     * EC recovery ID.
     */
    v?: Numbers | Uint8Array;
    /**
     * EC signature parameter.
     */
    r?: Numbers | Uint8Array;
    /**
     * EC signature parameter.
     */
    s?: Numbers | Uint8Array;
    /**
     * The transaction type
     */
    type?: Numbers;
};
/**
 * {@link AccessListEIP2930Transaction} data.
 */
export interface AccessListEIP2930TxData extends TxData {
    /**
     * The transaction's chain ID
     */
    chainId?: Numbers;
    /**
     * The access list which contains the addresses/storage slots which the transaction wishes to access
     */
    accessList?: AccessListUint8Array | AccessList | null;
}
/**
 * {@link FeeMarketEIP1559Transaction} data.
 */
export interface FeeMarketEIP1559TxData extends AccessListEIP2930TxData {
    /**
     * The transaction's gas price, inherited from {@link Transaction}.  This property is not used for EIP1559
     * transactions and should always be undefined for this specific transaction type.
     */
    gasPrice?: never | null;
    /**
     * The maximum inclusion fee per gas (this fee is given to the miner)
     */
    maxPriorityFeePerGas?: Numbers | Uint8Array;
    /**
     * The maximum total fee
     */
    maxFeePerGas?: Numbers | Uint8Array;
}
/**
 * Uint8Array values array for a legacy {@link Transaction}
 */
export type TxValuesArray = Uint8Array[];
/**
 * Uint8Array values array for an {@link AccessListEIP2930Transaction}
 */
export type AccessListEIP2930ValuesArray = [
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    AccessListUint8Array,
    Uint8Array?,
    Uint8Array?,
    Uint8Array?
];
/**
 * Uint8Array values array for a {@link FeeMarketEIP1559Transaction}
 */
export type FeeMarketEIP1559ValuesArray = [
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    Uint8Array,
    AccessListUint8Array,
    Uint8Array?,
    Uint8Array?,
    Uint8Array?
];
type JsonAccessListItem = {
    address: string;
    storageKeys: string[];
};
/**
 * Generic interface for all tx types with a
 * JSON representation of a transaction.
 *
 * Note that all values are marked as optional
 * and not all the values are present on all tx types
 * (an EIP1559 tx e.g. lacks a `gasPrice`).
 */
export interface JsonTx {
    nonce?: string;
    gasPrice?: string;
    gasLimit?: string;
    to?: string;
    data?: string;
    v?: string;
    r?: string;
    s?: string;
    value?: string;
    chainId?: string;
    accessList?: JsonAccessListItem[];
    type?: string;
    maxPriorityFeePerGas?: string;
    maxFeePerGas?: string;
    maxFeePerDataGas?: string;
    versionedHashes?: string[];
}
export {};
//# sourceMappingURL=types.d.ts.map