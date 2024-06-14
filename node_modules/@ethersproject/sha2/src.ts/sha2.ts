"use strict";

import { createHash, createHmac } from "crypto";

import hash from "hash.js";

import { arrayify, BytesLike } from "@ethersproject/bytes";

import { SupportedAlgorithm } from "./types";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

export function ripemd160(data: BytesLike): string {
    return "0x" + (hash.ripemd160().update(arrayify(data)).digest("hex"));
}

export function sha256(data: BytesLike): string {
    return "0x" + createHash("sha256").update(Buffer.from(arrayify(data))).digest("hex")
}

export function sha512(data: BytesLike): string {
    return "0x" + createHash("sha512").update(Buffer.from(arrayify(data))).digest("hex")
}

export function computeHmac(algorithm: SupportedAlgorithm, key: BytesLike, data: BytesLike): string {
    /* istanbul ignore if */
    if (!SupportedAlgorithm[algorithm]) {
        logger.throwError("unsupported algorithm - " + algorithm, Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "computeHmac",
            algorithm: algorithm
        });
    }

    return "0x" + createHmac(algorithm, Buffer.from(arrayify(key))).update(Buffer.from(arrayify(data))).digest("hex");
}

