/**
 * External Interfaces for other EthereumJS libraries
 */
import type { Account, Address, PrefixedHexString } from '@nomicfoundation/ethereumjs-util';
export interface StorageDump {
    [key: string]: string;
}
/**
 * Object that can contain a set of storage keys associated with an account.
 */
export interface StorageRange {
    /**
     * A dictionary where the keys are hashed storage keys, and the values are
     * objects containing the preimage of the hashed key (in `key`) and the
     * storage key (in `value`). Currently, there is no way to retrieve preimages,
     * so they are always `null`.
     */
    storage: {
        [key: string]: {
            key: string | null;
            value: string;
        };
    };
    /**
     * The next (hashed) storage key after the greatest storage key
     * contained in `storage`.
     */
    nextKey: string | null;
}
export declare type AccountFields = Partial<Pick<Account, 'nonce' | 'balance' | 'storageRoot' | 'codeHash'>>;
export declare type StorageProof = {
    key: PrefixedHexString;
    proof: PrefixedHexString[];
    value: PrefixedHexString;
};
export declare type Proof = {
    address: PrefixedHexString;
    balance: PrefixedHexString;
    codeHash: PrefixedHexString;
    nonce: PrefixedHexString;
    storageHash: PrefixedHexString;
    accountProof: PrefixedHexString[];
    storageProof: StorageProof[];
};
export declare type AccessListItem = {
    address: PrefixedHexString;
    storageKeys: PrefixedHexString[];
};
export declare type AccessListBytesItem = [Uint8Array, Uint8Array[]];
export declare type AccessListBytes = AccessListBytesItem[];
export declare type AccessList = AccessListItem[];
export interface StateManagerInterface {
    getAccount(address: Address): Promise<Account | undefined>;
    putAccount(address: Address, account?: Account): Promise<void>;
    deleteAccount(address: Address): Promise<void>;
    modifyAccountFields(address: Address, accountFields: AccountFields): Promise<void>;
    putContractCode(address: Address, value: Uint8Array): Promise<void>;
    getContractCode(address: Address): Promise<Uint8Array>;
    getContractStorage(address: Address, key: Uint8Array): Promise<Uint8Array>;
    putContractStorage(address: Address, key: Uint8Array, value: Uint8Array): Promise<void>;
    clearContractStorage(address: Address): Promise<void>;
    checkpoint(): Promise<void>;
    commit(): Promise<void>;
    revert(): Promise<void>;
    getStateRoot(): Promise<Uint8Array>;
    setStateRoot(stateRoot: Uint8Array, clearCache?: boolean): Promise<void>;
    getProof?(address: Address, storageSlots: Uint8Array[]): Promise<Proof>;
    hasStateRoot(root: Uint8Array): Promise<boolean>;
    shallowCopy(downlevelCaches?: boolean): StateManagerInterface;
}
export interface EVMStateManagerInterface extends StateManagerInterface {
    originalStorageCache: {
        get(address: Address, key: Uint8Array): Promise<Uint8Array>;
        clear(): void;
    };
    dumpStorage(address: Address): Promise<StorageDump>;
    dumpStorageRange(address: Address, startKey: bigint, limit: number): Promise<StorageRange>;
    generateCanonicalGenesis(initState: any): Promise<void>;
    getProof(address: Address, storageSlots?: Uint8Array[]): Promise<Proof>;
    shallowCopy(downlevelCaches?: boolean): EVMStateManagerInterface;
}
//# sourceMappingURL=interfaces.d.ts.map