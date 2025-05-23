/* eslint-disable @typescript-eslint/ban-types */

type Callback<T = any> = (value: T, key: string, parent: any) => T;

/**
 * visits all values in a complex object.
 * allows us to perform trasformations on values
 */
export function visit<T extends object = any>(value: T, callback: Callback): T {
    if (Array.isArray(value)) {
        value.forEach((_, index) => visitKey(index, value, callback));
    } else {
        Object.keys(value).forEach((key) => visitKey(key as keyof T, value, callback));
    }

    return value;
}

function visitKey<T extends object, K extends keyof T>(key: K, parent: T, callback: Callback<T[K]>) {
    const keyValue = parent[key];

    parent[key] = callback(keyValue, key as string, parent);

    if (typeof keyValue === 'object') {
        visit(keyValue as any, callback);
    }
}
