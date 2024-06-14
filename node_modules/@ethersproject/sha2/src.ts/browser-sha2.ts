"use strict";

import hash from "hash.js";
//const _ripemd160 = _hash.ripemd160;

import { arrayify, BytesLike } from "@ethersproject/bytes";

import { SupportedAlgorithm } from "./types";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

export function ripemd160(data: BytesLike): string {
    return "0x" + (hash.ripemd160().update(arrayify(data)).digest("hex"));
}

export function sha256(data: BytesLike): string {
    return "0x" + (hash.sha256().update(arrayify(data)).digest("hex"));
}

export function sha512(data: BytesLike): string {
    return "0x" + (hash.sha512().update(arrayify(data)).digest("hex"));
}

export function computeHmac(algorithm: SupportedAlgorithm, key: BytesLike, data: BytesLike): string {
    if (!SupportedAlgorithm[algorithm]) {
        logger.throwError("unsupported algorithm " + algorithm, Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "hmac",
            algorithm: algorithm
        });
    }

    return "0x" + hash.hmac((<any>hash)[algorithm], arrayify(key)).update(arrayify(data)).digest("hex");
}

