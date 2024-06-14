declare type OnProgressCallback = (progress: number) => void;
export declare function scrypt(password: Uint8Array, salt: Uint8Array, n: number, p: number, r: number, dkLen: number, onProgress?: OnProgressCallback): Promise<Uint8Array>;
export declare function scryptSync(password: Uint8Array, salt: Uint8Array, n: number, p: number, r: number, dkLen: number, onProgress?: OnProgressCallback): Uint8Array;
export {};
