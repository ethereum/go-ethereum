"use strict";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

export function defineReadOnly<T, K extends keyof T>(object: T, name: K, value: T[K]): void {
    Object.defineProperty(object, name, {
        enumerable: true,
        value: value,
        writable: false,
    });
}

// Crawl up the constructor chain to find a static method
export function getStatic<T>(ctor: any, key: string): T {
    for (let i = 0; i < 32; i++) {
        if (ctor[key]) { return ctor[key]; }
        if (!ctor.prototype || typeof(ctor.prototype) !== "object") { break; }
        ctor = Object.getPrototypeOf(ctor.prototype).constructor;
    }
    return null;
}

export type Deferrable<T> = {
    [ K in keyof T ]: T[K] | Promise<T[K]>;
}


type Result = { key: string, value: any};

export async function resolveProperties<T>(object: Readonly<Deferrable<T>>): Promise<T> {
    const promises: Array<Promise<Result>> = Object.keys(object).map((key) => {
        const value = object[<keyof Deferrable<T>>key];
        return Promise.resolve(value).then((v) => ({ key: key, value: v }));
    });

    const results = await Promise.all(promises);

    return results.reduce((accum, result) => {
        accum[<keyof T>(result.key)] = result.value;
        return accum;
    }, <T>{ });
}

export function checkProperties(object: any, properties: { [ name: string ]: boolean }): void {
    if (!object || typeof(object) !== "object") {
        logger.throwArgumentError("invalid object", "object", object);
    }

    Object.keys(object).forEach((key) => {
        if (!properties[key]) {
            logger.throwArgumentError("invalid object key - " + key, "transaction:" + key, object);
        }
    });
}

export function shallowCopy<T>(object: T): T {
    const result: any = {};
    for (const key in object) { result[key] = object[key]; }
    return result;
}

const opaque: { [key: string]: boolean } = { bigint: true, boolean: true, "function": true, number: true, string: true };

function _isFrozen(object: any): boolean {

    // Opaque objects are not mutable, so safe to copy by assignment
    if (object === undefined || object === null || opaque[typeof(object)]) { return true; }

    if (Array.isArray(object) || typeof(object) === "object") {
        if (!Object.isFrozen(object)) { return false; }

        const keys = Object.keys(object);
        for (let i = 0; i < keys.length; i++) {
            let value: any = null;
            try {
                value = object[keys[i]];
            } catch (error) {
                // If accessing a value triggers an error, it is a getter
                // designed to do so (e.g. Result) and is therefore "frozen"
                continue;
            }

            if (!_isFrozen(value)) { return false; }
        }

        return true;
    }

    return logger.throwArgumentError(`Cannot deepCopy ${ typeof(object) }`, "object", object);
}

// Returns a new copy of object, such that no properties may be replaced.
// New properties may be added only to objects.
function _deepCopy(object: any): any {

    if (_isFrozen(object)) { return object; }

    // Arrays are mutable, so we need to create a copy
    if (Array.isArray(object)) {
        return Object.freeze(object.map((item) => deepCopy(item)));
    }

    if (typeof(object) === "object") {
        const result: { [ key: string ]: any } = {};
        for (const key in object) {
            const value = object[key];
            if (value === undefined) { continue; }
            defineReadOnly(result, key, deepCopy(value));
        }

        return result;
    }

    return logger.throwArgumentError(`Cannot deepCopy ${ typeof(object) }`, "object", object);
}

export function deepCopy<T>(object: T): T {
    return _deepCopy(object);
}

export class Description<T = any> {
    constructor(info: { [ K in keyof T ]: T[K] }) {
        for (const key in info) {
            (<any>this)[key] = deepCopy(info[key]);
        }
    }
}
