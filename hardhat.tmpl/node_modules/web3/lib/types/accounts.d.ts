import { EthExecutionAPI, Bytes, Transaction, KeyStore } from 'web3-types';
import { Web3Context } from 'web3-core';
import { Wallet } from 'web3-eth-accounts';
/**
 * Initialize the accounts module for the given context.
 *
 * To avoid multiple package dependencies for `web3-eth-accounts` we are creating
 * this function in `web3` package. In future the actual `web3-eth-accounts` package
 * should be converted to context aware.
 */
export declare const initAccountsForContext: (context: Web3Context<EthExecutionAPI>) => {
    signTransaction: (transaction: Transaction, privateKey: Bytes) => Promise<import("web3-types").SignTransactionResult>;
    create: () => {
        signTransaction: (transaction: Transaction) => Promise<import("web3-types").SignTransactionResult>;
        address: import("web3-types").HexString;
        privateKey: import("web3-types").HexString;
        sign: (data: Record<string, unknown> | string) => import("web3-types").SignResult;
        encrypt: (password: string, options?: Record<string, unknown>) => Promise<KeyStore>;
    };
    privateKeyToAccount: (privateKey: Uint8Array | string) => {
        signTransaction: (transaction: Transaction) => Promise<import("web3-types").SignTransactionResult>;
        address: import("web3-types").HexString;
        privateKey: import("web3-types").HexString;
        sign: (data: Record<string, unknown> | string) => import("web3-types").SignResult;
        encrypt: (password: string, options?: Record<string, unknown>) => Promise<KeyStore>;
    };
    decrypt: (keystore: KeyStore | string, password: string, options?: Record<string, unknown>) => Promise<{
        signTransaction: (transaction: Transaction) => Promise<import("web3-types").SignTransactionResult>;
        address: import("web3-types").HexString;
        privateKey: import("web3-types").HexString;
        sign: (data: Record<string, unknown> | string) => import("web3-types").SignResult;
        encrypt: (password: string, options?: Record<string, unknown>) => Promise<KeyStore>;
    }>;
    recoverTransaction: (rawTransaction: import("web3-types").HexString) => import("web3-types").Address;
    hashMessage: (message: string, skipPrefix?: boolean) => string;
    sign: (data: string, privateKey: Bytes) => import("web3-types").SignResult;
    recover: (data: string | import("web3-types").SignatureObject, signatureOrV?: string, prefixedOrR?: boolean | string, s?: string, prefixed?: boolean) => import("web3-types").Address;
    encrypt: (privateKey: Bytes, password: string | Uint8Array, options?: import("web3-types").CipherOptions) => Promise<KeyStore>;
    wallet: Wallet<{
        signTransaction: (transaction: Transaction) => Promise<import("web3-types").SignTransactionResult>;
        address: import("web3-types").HexString;
        privateKey: import("web3-types").HexString;
        sign: (data: Record<string, unknown> | string) => import("web3-types").SignResult;
        encrypt: (password: string, options?: Record<string, unknown>) => Promise<KeyStore>;
    }>;
    privateKeyToAddress: (privateKey: Bytes) => string;
    parseAndValidatePrivateKey: (data: Bytes, ignoreLength?: boolean) => Uint8Array;
    privateKeyToPublicKey: (privateKey: Bytes, isCompressed: boolean) => string;
};
//# sourceMappingURL=accounts.d.ts.map