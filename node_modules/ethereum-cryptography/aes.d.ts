/// <reference types="node" />
export declare function encrypt(msg: Buffer, key: Buffer, iv: Buffer, mode?: string, pkcs7PaddingEnabled?: boolean): Buffer;
export declare function decrypt(cypherText: Buffer, key: Buffer, iv: Buffer, mode?: string, pkcs7PaddingEnabled?: boolean): Buffer;
//# sourceMappingURL=aes.d.ts.map