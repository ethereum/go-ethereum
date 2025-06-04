import { Bytes, BytesLike } from "@ethersproject/bytes";
export declare function looseArrayify(hexString: string): Uint8Array;
export declare function zpad(value: String | number, length: number): String;
export declare function getPassword(password: Bytes | string): Uint8Array;
export declare function searchPath(object: any, path: string): string;
export declare function uuidV4(randomBytes: BytesLike): string;
//# sourceMappingURL=utils.d.ts.map