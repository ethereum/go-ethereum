/**
 *  Each state-changing operation on Ethereum requires a transaction.
 *
 *  @_section api/transaction:Transactions  [about-transactions]
 */
/**
 *  A single [[AccessList]] entry of storage keys (slots) for an address.
 */
export type AccessListEntry = {
    address: string;
    storageKeys: Array<string>;
};
/**
 *  An ordered collection of [[AccessList]] entries.
 */
export type AccessList = Array<AccessListEntry>;
/**
 *  Any ethers-supported access list structure.
 */
export type AccessListish = AccessList | Array<[string, Array<string>]> | Record<string, Array<string>>;
export { accessListify } from "./accesslist.js";
export { computeAddress, recoverAddress } from "./address.js";
export { Transaction } from "./transaction.js";
export type { Blob, BlobLike, KzgLibrary, TransactionLike } from "./transaction.js";
//# sourceMappingURL=index.d.ts.map