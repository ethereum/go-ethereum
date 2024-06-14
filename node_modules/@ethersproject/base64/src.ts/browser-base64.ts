"use strict";

import { arrayify, BytesLike } from "@ethersproject/bytes";

export function decode(textData: string): Uint8Array {
    textData = atob(textData);
    const data = [];
    for (let i = 0; i < textData.length; i++) {
        data.push(textData.charCodeAt(i));
    }
    return arrayify(data);
}

export function encode(data: BytesLike): string {
    data = arrayify(data);
    let textData = "";
    for (let i = 0; i < data.length; i++) {
        textData += String.fromCharCode(data[i]);
    }
    return btoa(textData);
}


