import { BytesLike } from "@ethersproject/bytes";
import { SupportedAlgorithm } from "./types";
export declare function ripemd160(data: BytesLike): string;
export declare function sha256(data: BytesLike): string;
export declare function sha512(data: BytesLike): string;
export declare function computeHmac(algorithm: SupportedAlgorithm, key: BytesLike, data: BytesLike): string;
//# sourceMappingURL=sha2.d.ts.map