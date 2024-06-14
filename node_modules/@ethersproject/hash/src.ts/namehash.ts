import { concat, hexlify } from "@ethersproject/bytes";
import { toUtf8Bytes, toUtf8String } from "@ethersproject/strings";
import { keccak256 } from "@ethersproject/keccak256";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

import { ens_normalize } from "./ens-normalize/lib";

const Zeros = new Uint8Array(32);
Zeros.fill(0);

function checkComponent(comp: Uint8Array): Uint8Array {
    if (comp.length === 0) { throw new Error("invalid ENS name; empty component"); }
    return comp;
}

function ensNameSplit(name: string): Array<Uint8Array> {
    const bytes = toUtf8Bytes(ens_normalize(name));
    const comps: Array<Uint8Array> = [ ];

    if (name.length === 0) { return comps; }

    let last = 0;
    for (let i = 0; i < bytes.length; i++) {
        const d = bytes[i];

        // A separator (i.e. "."); copy this component
        if (d === 0x2e) {
            comps.push(checkComponent(bytes.slice(last, i)));
            last = i + 1;
        }
    }

    // There was a stray separator at the end of the name
    if (last >= bytes.length) { throw new Error("invalid ENS name; empty component"); }

    comps.push(checkComponent(bytes.slice(last)));
    return comps;
}

export function ensNormalize(name: string): string {
    return ensNameSplit(name).map((comp) => toUtf8String(comp)).join(".");
}

export function isValidName(name: string): boolean {
    try {
        return (ensNameSplit(name).length !== 0);
    } catch (error) { }
    return false;
}

export function namehash(name: string): string {
    /* istanbul ignore if */
    if (typeof(name) !== "string") {
        logger.throwArgumentError("invalid ENS name; not a string", "name", name);
    }

    let result: string | Uint8Array = Zeros;

    const comps = ensNameSplit(name);
    while (comps.length) {
        result = keccak256(concat([result, keccak256(comps.pop())]));
    }

    return hexlify(result);
}

export function dnsEncode(name: string): string {
    return hexlify(concat(ensNameSplit(name).map((comp) => {
        // DNS does not allow components over 63 bytes in length
        if (comp.length > 63) {
            throw new Error("invalid DNS encoded entry; length exceeds 63 bytes");
        }

        const bytes = new Uint8Array(comp.length + 1);
        bytes.set(comp, 1);
        bytes[0] = bytes.length - 1;
        return bytes;

    }))) + "00";
}
