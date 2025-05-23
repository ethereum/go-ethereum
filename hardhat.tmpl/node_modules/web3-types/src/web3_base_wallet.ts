/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
import { Transaction } from './eth_types.js';
import { HexString } from './primitives_types.js';

export type Cipher = 'aes-128-ctr' | 'aes-128-cbc' | 'aes-256-cbc';

export type CipherOptions = {
	salt?: Uint8Array | string;
	iv?: Uint8Array | string;
	kdf?: 'scrypt' | 'pbkdf2';
	dklen?: number;
	c?: number; // iterrations
	n?: number; // cpu/memory cost
	r?: number; // block size
	p?: number; // parallelization cost
};

export type ScryptParams = {
	dklen: number;
	n: number;
	p: number;
	r: number;
	salt: Uint8Array | string;
};
export type PBKDF2SHA256Params = {
	c: number; // iterations
	dklen: number;
	prf: 'hmac-sha256';
	salt: Uint8Array | string;
};

export type KeyStore = {
	crypto: {
		cipher: Cipher;
		ciphertext: string;
		cipherparams: {
			iv: string;
		};
		kdf: 'pbkdf2' | 'scrypt';
		kdfparams: ScryptParams | PBKDF2SHA256Params;
		mac: HexString;
	};
	id: string;
	version: 3;
	address: string;
};

export type SignatureObject = {
	messageHash: string;
	r: string;
	s: string;
	v: string;
};

export type SignTransactionResult = SignatureObject & {
	rawTransaction: string;
	transactionHash: string;
};

export type SignResult = SignatureObject & {
	message?: string;
	signature: string;
};

export interface Web3BaseWalletAccount {
	[key: string]: unknown;
	readonly address: string;
	readonly privateKey: string;
	readonly signTransaction: (tx: Transaction) => Promise<SignTransactionResult>;
	readonly sign: (data: Record<string, unknown> | string) => SignResult;
	readonly encrypt: (password: string, options?: Record<string, unknown>) => Promise<KeyStore>;
}

export interface Web3AccountProvider<T> {
	privateKeyToAccount: (privateKey: string) => T;
	create: () => T;
	decrypt: (
		keystore: KeyStore | string,
		password: string,
		options?: Record<string, unknown>,
	) => Promise<T>;
}

export abstract class Web3BaseWallet<T extends Web3BaseWalletAccount> extends Array<T> {
	protected readonly _accountProvider: Web3AccountProvider<T>;

	public constructor(accountProvider: Web3AccountProvider<T>) {
		super();
		this._accountProvider = accountProvider;
	}

	public abstract create(numberOfAccounts: number): this;
	public abstract add(account: T | string): this;
	public abstract get(addressOrIndex: string | number): T | undefined;
	public abstract remove(addressOrIndex: string | number): boolean;
	public abstract clear(): this;
	public abstract encrypt(
		password: string,
		options?: Record<string, unknown>,
	): Promise<KeyStore[]>;
	public abstract decrypt(
		encryptedWallet: KeyStore[],
		password: string,
		options?: Record<string, unknown>,
	): Promise<this>;
	public abstract save(password: string, keyName?: string): Promise<boolean | never>;
	public abstract load(password: string, keyName?: string): Promise<this | never>;
}
