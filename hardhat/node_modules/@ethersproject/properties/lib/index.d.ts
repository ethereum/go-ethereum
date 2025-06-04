export declare function defineReadOnly<T, K extends keyof T>(object: T, name: K, value: T[K]): void;
export declare function getStatic<T>(ctor: any, key: string): T;
export declare type Deferrable<T> = {
    [K in keyof T]: T[K] | Promise<T[K]>;
};
export declare function resolveProperties<T>(object: Readonly<Deferrable<T>>): Promise<T>;
export declare function checkProperties(object: any, properties: {
    [name: string]: boolean;
}): void;
export declare function shallowCopy<T>(object: T): T;
export declare function deepCopy<T>(object: T): T;
export declare class Description<T = any> {
    constructor(info: {
        [K in keyof T]: T[K];
    });
}
//# sourceMappingURL=index.d.ts.map