/// <reference types="node" />
import "scrypt-js/thirdparty/setImmediate";
export declare function scrypt(password: Buffer, salt: Buffer, n: number, p: number, r: number, dklen: number): Promise<Buffer>;
export declare function scryptSync(password: Buffer, salt: Buffer, n: number, p: number, r: number, dklen: number): Buffer;
//# sourceMappingURL=scrypt.d.ts.map