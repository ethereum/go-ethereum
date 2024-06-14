declare type Callback<T = any> = (value: T, key: string, parent: any) => T;
/**
 * visits all values in a complex object.
 * allows us to perform trasformations on values
 */
export declare function visit<T extends object = any>(value: T, callback: Callback): T;
export {};
