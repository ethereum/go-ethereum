export declare function encrypt(msg: Uint8Array, key: Uint8Array, iv: Uint8Array, mode?: string, pkcs7PaddingEnabled?: boolean): Promise<Uint8Array>;
export declare function decrypt(cypherText: Uint8Array, key: Uint8Array, iv: Uint8Array, mode?: string, pkcs7PaddingEnabled?: boolean): Promise<Uint8Array>;
