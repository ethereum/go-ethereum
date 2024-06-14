/// <reference types="node" />
export declare function pbkdf2(password: Buffer, salt: Buffer, iterations: number, keylen: number, digest: string): Promise<Buffer>;
export declare function pbkdf2Sync(password: Buffer, salt: Buffer, iterations: number, keylen: number, digest: string): Buffer;
//# sourceMappingURL=pbkdf2.d.ts.map