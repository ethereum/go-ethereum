import { HexString, SignatureObject, SignResult, SignTransactionResult, Web3BaseWalletAccount } from 'web3-types';
import { AccessListEIP2930Transaction, FeeMarketEIP1559Transaction, Transaction } from './tx';
export { SignatureObject, SignResult, SignTransactionResult };
export interface Web3Account extends Web3BaseWalletAccount {
    address: HexString;
    privateKey: HexString;
}
/** This Web Storage API interface provides access to a particular domain's session or local storage. It allows, for example, the addition, modification, or deletion of stored data items. */
export interface WebStorage {
    /** Returns the number of key/value pairs. */
    readonly length: number;
    /**
     * Removes all key/value pairs, if there are any.
     *
     * Dispatches a storage event on Window objects holding an equivalent Storage object.
     */
    clear(): void;
    /** Returns the current value associated with the given key, or null if the given key does not exist. */
    getItem(key: string): string | null;
    /** Returns the name of the nth key, or null if n is greater than or equal to the number of key/value pairs. */
    key(index: number): string | null;
    /**
     * Removes the key/value pair with the given key, if a key/value pair with the given key exists.
     *
     * Dispatches a storage event on Window objects holding an equivalent Storage object.
     */
    removeItem(key: string): void;
    /**
     * Sets the value of the pair identified by key to value, creating a new key/value pair if none existed for key previously.
     *
     * Throws a "QuotaExceededError" DOMException exception if the new value couldn't be set. (Setting could fail if, e.g., the user has disabled storage for the site, or if the quota has been exceeded.)
     *
     * Dispatches a storage event on Window objects holding an equivalent Storage object.
     */
    setItem(key: string, value: string): void;
    [name: string]: any;
}
export type TypedTransaction = Transaction | AccessListEIP2930Transaction | FeeMarketEIP1559Transaction;
//# sourceMappingURL=types.d.ts.map