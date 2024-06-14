"use strict";

import { arrayify, BytesLike } from "@ethersproject/bytes";


export function decode(textData: string): Uint8Array {
    return arrayify(new Uint8Array(Buffer.from(textData, "base64")));
};

export function encode(data: BytesLike): string {
    return Buffer.from(arrayify(data)).toString("base64");
}
