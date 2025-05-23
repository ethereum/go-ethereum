export var Commented: typeof import("./commented");
export var Diagnose: typeof import("./diagnose");
export var Decoder: typeof import("./decoder");
export var Encoder: typeof import("./encoder");
export var Simple: typeof import("./simple");
export var Tagged: typeof import("./tagged");
export var Map: typeof import("./map");
export namespace leveldb {
    const decode: typeof import("./decoder").decodeFirstSync;
    const encode: typeof import("./encoder").encode;
    const buffer: boolean;
    const name: string;
}
export function reset(): void;
export var comment: typeof import("./commented").comment;
export var decodeAll: typeof import("./decoder").decodeAll;
export var decodeAllSync: typeof import("./decoder").decodeAllSync;
export var decodeFirst: typeof import("./decoder").decodeFirst;
export var decodeFirstSync: typeof import("./decoder").decodeFirstSync;
export var decode: typeof import("./decoder").decodeFirstSync;
export var diagnose: typeof import("./diagnose").diagnose;
export var encode: typeof import("./encoder").encode;
export var encodeCanonical: typeof import("./encoder").encodeCanonical;
export var encodeOne: typeof import("./encoder").encodeOne;
export var encodeAsync: typeof import("./encoder").encodeAsync;
