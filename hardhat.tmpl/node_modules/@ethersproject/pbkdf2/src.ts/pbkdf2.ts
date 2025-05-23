"use strict";

import { pbkdf2Sync as _pbkdf2 } from "crypto";

import { arrayify, BytesLike, hexlify } from "@ethersproject/bytes";


function bufferify(value: BytesLike): Buffer {
    return Buffer.from(arrayify(value));
}

export function pbkdf2(password: BytesLike, salt: BytesLike, iterations: number, keylen: number, hashAlgorithm: string): string {
    return hexlify(_pbkdf2(bufferify(password), bufferify(salt), iterations, keylen, hashAlgorithm));
}
