declare module "util" {
    interface InspectOptions extends NodeJS.InspectOptions { }
    function format(format: any, ...param: any[]): string;
    function formatWithOptions(inspectOptions: InspectOptions, format: string, ...param: any[]): string;
    /** @deprecated since v0.11.3 - use `console.error()` instead. */
    function debug(string: string): void;
    /** @deprecated since v0.11.3 - use `console.error()` instead. */
    function error(...param: any[]): void;
    /** @deprecated since v0.11.3 - use `console.log()` instead. */
    function puts(...param: any[]): void;
    /** @deprecated since v0.11.3 - use `console.log()` instead. */
    function print(...param: any[]): void;
    /** @deprecated since v0.11.3 - use a third party module instead. */
    function log(string: string): void;
    function inspect(object: any, showHidden?: boolean, depth?: number | null, color?: boolean): string;
    function inspect(object: any, options: InspectOptions): string;
    namespace inspect {
        let colors: {
            [color: string]: [number, number] | undefined
        };
        const custom: unique symbol;
        let styles: {
            [style: string]: string | undefined
        };
        let defaultOptions: InspectOptions;
    }
    /** @deprecated since v4.0.0 - use `Array.isArray()` instead. */
    function isArray(object: any): object is any[];
    /** @deprecated since v4.0.0 - use `util.types.isRegExp()` instead. */
    function isRegExp(object: any): object is RegExp;
    /** @deprecated since v4.0.0 - use `util.types.isDate()` instead. */
    function isDate(object: any): object is Date;
    /** @deprecated since v4.0.0 - use `util.types.isNativeError()` instead. */
    function isError(object: any): object is Error;
    function inherits(constructor: any, superConstructor: any): void;
    function debuglog(key: string): (msg: string, ...param: any[]) => void;
    /** @deprecated since v4.0.0 - use `typeof value === 'boolean'` instead. */
    function isBoolean(object: any): object is boolean;
    /** @deprecated since v4.0.0 - use `Buffer.isBuffer()` instead. */
    function isBuffer(object: any): object is Buffer;
    /** @deprecated since v4.0.0 - use `typeof value === 'function'` instead. */
    function isFunction(object: any): boolean;
    /** @deprecated since v4.0.0 - use `value === null` instead. */
    function isNull(object: any): object is null;
    /** @deprecated since v4.0.0 - use `value === null || value === undefined` instead. */
    function isNullOrUndefined(object: any): object is null | undefined;
    /** @deprecated since v4.0.0 - use `typeof value === 'number'` instead. */
    function isNumber(object: any): object is number;
    /** @deprecated since v4.0.0 - use `value !== null && typeof value === 'object'` instead. */
    function isObject(object: any): boolean;
    /** @deprecated since v4.0.0 - use `(typeof value !== 'object' && typeof value !== 'function') || value === null` instead. */
    function isPrimitive(object: any): boolean;
    /** @deprecated since v4.0.0 - use `typeof value === 'string'` instead. */
    function isString(object: any): object is string;
    /** @deprecated since v4.0.0 - use `typeof value === 'symbol'` instead. */
    function isSymbol(object: any): object is symbol;
    /** @deprecated since v4.0.0 - use `value === undefined` instead. */
    function isUndefined(object: any): object is undefined;
    function deprecate<T extends Function>(fn: T, message: string, code?: string): T;
    function isDeepStrictEqual(val1: any, val2: any): boolean;

    interface CustomPromisify<TCustom extends Function> extends Function {
        __promisify__: TCustom;
    }

    function callbackify(fn: () => Promise<void>): (callback: (err: NodeJS.ErrnoException) => void) => void;
    function callbackify<TResult>(fn: () => Promise<TResult>): (callback: (err: NodeJS.ErrnoException | null, result: TResult) => void) => void;
    function callbackify<T1>(fn: (arg1: T1) => Promise<void>): (arg1: T1, callback: (err: NodeJS.ErrnoException) => void) => void;
    function callbackify<T1, TResult>(fn: (arg1: T1) => Promise<TResult>): (arg1: T1, callback: (err: NodeJS.ErrnoException | null, result: TResult) => void) => void;
    function callbackify<T1, T2>(fn: (arg1: T1, arg2: T2) => Promise<void>): (arg1: T1, arg2: T2, callback: (err: NodeJS.ErrnoException) => void) => void;
    function callbackify<T1, T2, TResult>(fn: (arg1: T1, arg2: T2) => Promise<TResult>): (arg1: T1, arg2: T2, callback: (err: NodeJS.ErrnoException | null, result: TResult) => void) => void;
    function callbackify<T1, T2, T3>(fn: (arg1: T1, arg2: T2, arg3: T3) => Promise<void>): (arg1: T1, arg2: T2, arg3: T3, callback: (err: NodeJS.ErrnoException) => void) => void;
    function callbackify<T1, T2, T3, TResult>(
        fn: (arg1: T1, arg2: T2, arg3: T3) => Promise<TResult>): (arg1: T1, arg2: T2, arg3: T3, callback: (err: NodeJS.ErrnoException | null, result: TResult) => void) => void;
    function callbackify<T1, T2, T3, T4>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4) => Promise<void>): (arg1: T1, arg2: T2, arg3: T3, arg4: T4, callback: (err: NodeJS.ErrnoException) => void) => void;
    function callbackify<T1, T2, T3, T4, TResult>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4) => Promise<TResult>): (arg1: T1, arg2: T2, arg3: T3, arg4: T4, callback: (err: NodeJS.ErrnoException | null, result: TResult) => void) => void;
    function callbackify<T1, T2, T3, T4, T5>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5) => Promise<void>): (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5, callback: (err: NodeJS.ErrnoException) => void) => void;
    function callbackify<T1, T2, T3, T4, T5, TResult>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5) => Promise<TResult>,
    ): (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5, callback: (err: NodeJS.ErrnoException | null, result: TResult) => void) => void;
    function callbackify<T1, T2, T3, T4, T5, T6>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5, arg6: T6) => Promise<void>,
    ): (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5, arg6: T6, callback: (err: NodeJS.ErrnoException) => void) => void;
    function callbackify<T1, T2, T3, T4, T5, T6, TResult>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5, arg6: T6) => Promise<TResult>
    ): (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5, arg6: T6, callback: (err: NodeJS.ErrnoException | null, result: TResult) => void) => void;

    function promisify<TCustom extends Function>(fn: CustomPromisify<TCustom>): TCustom;
    function promisify<TResult>(fn: (callback: (err: Error | null, result: TResult) => void) => void): () => Promise<TResult>;
    function promisify(fn: (callback: (err?: Error | null) => void) => void): () => Promise<void>;
    function promisify<T1, TResult>(fn: (arg1: T1, callback: (err: Error | null, result: TResult) => void) => void): (arg1: T1) => Promise<TResult>;
    function promisify<T1>(fn: (arg1: T1, callback: (err?: Error | null) => void) => void): (arg1: T1) => Promise<void>;
    function promisify<T1, T2, TResult>(fn: (arg1: T1, arg2: T2, callback: (err: Error | null, result: TResult) => void) => void): (arg1: T1, arg2: T2) => Promise<TResult>;
    function promisify<T1, T2>(fn: (arg1: T1, arg2: T2, callback: (err?: Error | null) => void) => void): (arg1: T1, arg2: T2) => Promise<void>;
    function promisify<T1, T2, T3, TResult>(fn: (arg1: T1, arg2: T2, arg3: T3, callback: (err: Error | null, result: TResult) => void) => void): (arg1: T1, arg2: T2, arg3: T3) => Promise<TResult>;
    function promisify<T1, T2, T3>(fn: (arg1: T1, arg2: T2, arg3: T3, callback: (err?: Error | null) => void) => void): (arg1: T1, arg2: T2, arg3: T3) => Promise<void>;
    function promisify<T1, T2, T3, T4, TResult>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4, callback: (err: Error | null, result: TResult) => void) => void,
    ): (arg1: T1, arg2: T2, arg3: T3, arg4: T4) => Promise<TResult>;
    function promisify<T1, T2, T3, T4>(fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4, callback: (err?: Error | null) => void) => void): (arg1: T1, arg2: T2, arg3: T3, arg4: T4) => Promise<void>;
    function promisify<T1, T2, T3, T4, T5, TResult>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5, callback: (err: Error | null, result: TResult) => void) => void,
    ): (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5) => Promise<TResult>;
    function promisify<T1, T2, T3, T4, T5>(
        fn: (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5, callback: (err?: Error | null) => void) => void,
    ): (arg1: T1, arg2: T2, arg3: T3, arg4: T4, arg5: T5) => Promise<void>;
    function promisify(fn: Function): Function;

    namespace types {
        function isAnyArrayBuffer(object: any): object is ArrayBufferLike;
        function isArgumentsObject(object: any): object is IArguments;
        function isArrayBuffer(object: any): object is ArrayBuffer;
        function isArrayBufferView(object: any): object is NodeJS.ArrayBufferView;
        function isAsyncFunction(object: any): boolean;
        function isBigInt64Array(value: any): value is BigInt64Array;
        function isBigUint64Array(value: any): value is BigUint64Array;
        function isBooleanObject(object: any): object is Boolean;
        function isBoxedPrimitive(object: any): object is String | Number | BigInt | Boolean | Symbol;
        function isDataView(object: any): object is DataView;
        function isDate(object: any): object is Date;
        function isExternal(object: any): boolean;
        function isFloat32Array(object: any): object is Float32Array;
        function isFloat64Array(object: any): object is Float64Array;
        function isGeneratorFunction(object: any): object is GeneratorFunction;
        function isGeneratorObject(object: any): object is Generator;
        function isInt8Array(object: any): object is Int8Array;
        function isInt16Array(object: any): object is Int16Array;
        function isInt32Array(object: any): object is Int32Array;
        function isMap<T>(
            object: T | {},
        ): object is T extends ReadonlyMap<any, any>
            ? unknown extends T
                ? never
                : ReadonlyMap<any, any>
            : Map<any, any>;
        function isMapIterator(object: any): boolean;
        function isModuleNamespaceObject(value: any): boolean;
        function isNativeError(object: any): object is Error;
        function isNumberObject(object: any): object is Number;
        function isPromise(object: any): object is Promise<any>;
        function isProxy(object: any): boolean;
        function isRegExp(object: any): object is RegExp;
        function isSet<T>(
            object: T | {},
        ): object is T extends ReadonlySet<any>
            ? unknown extends T
                ? never
                : ReadonlySet<any>
            : Set<any>;
        function isSetIterator(object: any): boolean;
        function isSharedArrayBuffer(object: any): object is SharedArrayBuffer;
        function isStringObject(object: any): object is String;
        function isSymbolObject(object: any): object is Symbol;
        function isTypedArray(object: any): object is NodeJS.TypedArray;
        function isUint8Array(object: any): object is Uint8Array;
        function isUint8ClampedArray(object: any): object is Uint8ClampedArray;
        function isUint16Array(object: any): object is Uint16Array;
        function isUint32Array(object: any): object is Uint32Array;
        function isWeakMap(object: any): object is WeakMap<any, any>;
        function isWeakSet(object: any): object is WeakSet<any>;
        /** @deprecated Removed in v14.0.0 */
        function isWebAssemblyCompiledModule(object: any): boolean;
    }

    class TextDecoder {
        readonly encoding: string;
        readonly fatal: boolean;
        readonly ignoreBOM: boolean;
        constructor(
          encoding?: string,
          options?: { fatal?: boolean; ignoreBOM?: boolean }
        );
        decode(
          input?: NodeJS.TypedArray | DataView | ArrayBuffer | null,
          options?: { stream?: boolean }
        ): string;
    }

    class TextEncoder {
        readonly encoding: string;
        constructor();
        encode(input?: string): Uint8Array;
    }
}
