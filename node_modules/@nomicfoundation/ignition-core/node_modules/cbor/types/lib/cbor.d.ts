import Commented = require("./commented");
import Diagnose = require("./diagnose");
import Decoder = require("./decoder");
import Encoder = require("./encoder");
import Simple = require("./simple");
import Tagged = require("./tagged");
import Map = require("./map");
import SharedValueEncoder = require("./sharedValueEncoder");
export declare const comment: typeof import("./commented").comment;
export declare const decodeAll: typeof import("./decoder").decodeAll;
export declare const decodeFirst: typeof import("./decoder").decodeFirst;
export declare const decodeAllSync: typeof import("./decoder").decodeAllSync;
export declare const decodeFirstSync: any;
export declare const diagnose: typeof import("./diagnose").diagnose;
export declare const encode: typeof import("./encoder").encode;
export declare const encodeCanonical: typeof import("./encoder").encodeCanonical;
export declare const encodeOne: typeof import("./encoder").encodeOne;
export declare const encodeAsync: typeof import("./encoder").encodeAsync;
export declare const decode: typeof import("./decoder").decodeFirstSync;
export declare namespace leveldb {
    const decode_1: typeof Decoder.decodeFirstSync;
    export { decode_1 as decode };
    const encode_1: typeof Encoder.encode;
    export { encode_1 as encode };
    export const buffer: boolean;
    export const name: string;
}
/**
 * Reset everything that we can predict a plugin might have altered in good
 * faith.  For now that includes the default set of tags that decoding and
 * encoding will use.
 */
export declare function reset(): void;
export { Commented, Diagnose, Decoder, Encoder, Simple, Tagged, Map, SharedValueEncoder };
