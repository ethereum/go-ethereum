import * as core from "zod/v4/core";
import { util } from "zod/v4/core";
export * as coerce from "./coerce.js";
export * as iso from "./iso.js";
type SomeType = core.$ZodType;
export interface ZodMiniType<out Output = unknown, out Input = unknown> extends core.$ZodType<Output, Input> {
    check(...checks: (core.CheckFn<this["_zod"]["output"]> | core.$ZodCheck<this["_zod"]["output"]>)[]): this;
    clone(def?: this["_zod"]["def"], params?: {
        parent: boolean;
    }): this;
    register<R extends core.$ZodRegistry>(registry: R, ...meta: this extends R["_schema"] ? undefined extends R["_meta"] ? [core.$ZodRegistry<R["_meta"], this>["_meta"]?] : [core.$ZodRegistry<R["_meta"], this>["_meta"]] : ["Incompatible schema"]): this;
    brand<T extends PropertyKey = PropertyKey>(value?: T): PropertyKey extends T ? this : this & Record<"_zod", Record<"~output", core.output<this> & core.$brand<T>>>;
    def: this["_zod"]["def"];
    parse(data: unknown, params?: core.ParseContext<core.$ZodIssue>): core.output<this>;
    safeParse(data: unknown, params?: core.ParseContext<core.$ZodIssue>): util.SafeParseResult<core.output<this>>;
    parseAsync(data: unknown, params?: core.ParseContext<core.$ZodIssue>): Promise<core.output<this>>;
    safeParseAsync(data: unknown, params?: core.ParseContext<core.$ZodIssue>): Promise<util.SafeParseResult<core.output<this>>>;
}
export declare const ZodMiniType: core.$constructor<ZodMiniType>;
export interface ZodMiniString<Input = unknown> extends ZodMiniType {
    _zod: core.$ZodStringInternals<Input>;
}
export declare const ZodMiniString: core.$constructor<ZodMiniString>;
export declare function string(params?: string | core.$ZodStringParams): ZodMiniString<string>;
export interface ZodMiniStringFormat<Format extends core.$ZodStringFormats = core.$ZodStringFormats> extends ZodMiniString {
    _zod: core.$ZodStringFormatInternals<Format>;
}
export declare const ZodMiniStringFormat: core.$constructor<ZodMiniStringFormat>;
export interface ZodMiniEmail extends ZodMiniStringFormat<"email"> {
    _zod: core.$ZodEmailInternals;
}
export declare const ZodMiniEmail: core.$constructor<ZodMiniEmail>;
export declare function email(params?: string | core.$ZodEmailParams): ZodMiniEmail;
export interface ZodMiniGUID extends ZodMiniStringFormat<"guid"> {
    _zod: core.$ZodGUIDInternals;
}
export declare const ZodMiniGUID: core.$constructor<ZodMiniGUID>;
export declare function guid(params?: string | core.$ZodGUIDParams): ZodMiniGUID;
export interface ZodMiniUUID extends ZodMiniStringFormat<"uuid"> {
    _zod: core.$ZodUUIDInternals;
}
export declare const ZodMiniUUID: core.$constructor<ZodMiniUUID>;
export declare function uuid(params?: string | core.$ZodUUIDParams): ZodMiniUUID;
export declare function uuidv4(params?: string | core.$ZodUUIDv4Params): ZodMiniUUID;
export declare function uuidv6(params?: string | core.$ZodUUIDv6Params): ZodMiniUUID;
export declare function uuidv7(params?: string | core.$ZodUUIDv7Params): ZodMiniUUID;
export interface ZodMiniURL extends ZodMiniStringFormat<"url"> {
    _zod: core.$ZodURLInternals;
}
export declare const ZodMiniURL: core.$constructor<ZodMiniURL>;
export declare function url(params?: string | core.$ZodURLParams): ZodMiniURL;
export interface ZodMiniEmoji extends ZodMiniStringFormat<"emoji"> {
    _zod: core.$ZodEmojiInternals;
}
export declare const ZodMiniEmoji: core.$constructor<ZodMiniEmoji>;
export declare function emoji(params?: string | core.$ZodEmojiParams): ZodMiniEmoji;
export interface ZodMiniNanoID extends ZodMiniStringFormat<"nanoid"> {
    _zod: core.$ZodNanoIDInternals;
}
export declare const ZodMiniNanoID: core.$constructor<ZodMiniNanoID>;
export declare function nanoid(params?: string | core.$ZodNanoIDParams): ZodMiniNanoID;
export interface ZodMiniCUID extends ZodMiniStringFormat<"cuid"> {
    _zod: core.$ZodCUIDInternals;
}
export declare const ZodMiniCUID: core.$constructor<ZodMiniCUID>;
export declare function cuid(params?: string | core.$ZodCUIDParams): ZodMiniCUID;
export interface ZodMiniCUID2 extends ZodMiniStringFormat<"cuid2"> {
    _zod: core.$ZodCUID2Internals;
}
export declare const ZodMiniCUID2: core.$constructor<ZodMiniCUID2>;
export declare function cuid2(params?: string | core.$ZodCUID2Params): ZodMiniCUID2;
export interface ZodMiniULID extends ZodMiniStringFormat<"ulid"> {
    _zod: core.$ZodULIDInternals;
}
export declare const ZodMiniULID: core.$constructor<ZodMiniULID>;
export declare function ulid(params?: string | core.$ZodULIDParams): ZodMiniULID;
export interface ZodMiniXID extends ZodMiniStringFormat<"xid"> {
    _zod: core.$ZodXIDInternals;
}
export declare const ZodMiniXID: core.$constructor<ZodMiniXID>;
export declare function xid(params?: string | core.$ZodXIDParams): ZodMiniXID;
export interface ZodMiniKSUID extends ZodMiniStringFormat<"ksuid"> {
    _zod: core.$ZodKSUIDInternals;
}
export declare const ZodMiniKSUID: core.$constructor<ZodMiniKSUID>;
export declare function ksuid(params?: string | core.$ZodKSUIDParams): ZodMiniKSUID;
export interface ZodMiniIPv4 extends ZodMiniStringFormat<"ipv4"> {
    _zod: core.$ZodIPv4Internals;
}
export declare const ZodMiniIPv4: core.$constructor<ZodMiniIPv4>;
export declare function ipv4(params?: string | core.$ZodIPv4Params): ZodMiniIPv4;
export interface ZodMiniIPv6 extends ZodMiniStringFormat<"ipv6"> {
    _zod: core.$ZodIPv6Internals;
}
export declare const ZodMiniIPv6: core.$constructor<ZodMiniIPv6>;
export declare function ipv6(params?: string | core.$ZodIPv6Params): ZodMiniIPv6;
export interface ZodMiniCIDRv4 extends ZodMiniStringFormat<"cidrv4"> {
    _zod: core.$ZodCIDRv4Internals;
}
export declare const ZodMiniCIDRv4: core.$constructor<ZodMiniCIDRv4>;
export declare function cidrv4(params?: string | core.$ZodCIDRv4Params): ZodMiniCIDRv4;
export interface ZodMiniCIDRv6 extends ZodMiniStringFormat<"cidrv6"> {
    _zod: core.$ZodCIDRv6Internals;
}
export declare const ZodMiniCIDRv6: core.$constructor<ZodMiniCIDRv6>;
export declare function cidrv6(params?: string | core.$ZodCIDRv6Params): ZodMiniCIDRv6;
export interface ZodMiniBase64 extends ZodMiniStringFormat<"base64"> {
    _zod: core.$ZodBase64Internals;
}
export declare const ZodMiniBase64: core.$constructor<ZodMiniBase64>;
export declare function base64(params?: string | core.$ZodBase64Params): ZodMiniBase64;
export interface ZodMiniBase64URL extends ZodMiniStringFormat<"base64url"> {
    _zod: core.$ZodBase64URLInternals;
}
export declare const ZodMiniBase64URL: core.$constructor<ZodMiniBase64URL>;
export declare function base64url(params?: string | core.$ZodBase64URLParams): ZodMiniBase64URL;
export interface ZodMiniE164 extends ZodMiniStringFormat<"e164"> {
    _zod: core.$ZodE164Internals;
}
export declare const ZodMiniE164: core.$constructor<ZodMiniE164>;
export declare function e164(params?: string | core.$ZodE164Params): ZodMiniE164;
export interface ZodMiniJWT extends ZodMiniStringFormat<"jwt"> {
    _zod: core.$ZodJWTInternals;
}
export declare const ZodMiniJWT: core.$constructor<ZodMiniJWT>;
export declare function jwt(params?: string | core.$ZodJWTParams): ZodMiniJWT;
export interface ZodMiniNumber<Input = unknown> extends ZodMiniType {
    _zod: core.$ZodNumberInternals<Input>;
}
export declare const ZodMiniNumber: core.$constructor<ZodMiniNumber>;
export declare function number(params?: string | core.$ZodNumberParams): ZodMiniNumber<number>;
export interface ZodMiniNumberFormat extends ZodMiniNumber {
    _zod: core.$ZodNumberFormatInternals;
}
export declare const ZodMiniNumberFormat: core.$constructor<ZodMiniNumberFormat>;
export declare function int(params?: string | core.$ZodCheckNumberFormatParams): ZodMiniNumberFormat;
export declare function float32(params?: string | core.$ZodCheckNumberFormatParams): ZodMiniNumberFormat;
export declare function float64(params?: string | core.$ZodCheckNumberFormatParams): ZodMiniNumberFormat;
export declare function int32(params?: string | core.$ZodCheckNumberFormatParams): ZodMiniNumberFormat;
export declare function uint32(params?: string | core.$ZodCheckNumberFormatParams): ZodMiniNumberFormat;
export interface ZodMiniBoolean<T = unknown> extends ZodMiniType {
    _zod: core.$ZodBooleanInternals<T>;
}
export declare const ZodMiniBoolean: core.$constructor<ZodMiniBoolean>;
export declare function boolean(params?: string | core.$ZodBooleanParams): ZodMiniBoolean<boolean>;
export interface ZodMiniBigInt<T = unknown> extends ZodMiniType {
    _zod: core.$ZodBigIntInternals<T>;
}
export declare const ZodMiniBigInt: core.$constructor<ZodMiniBigInt>;
export declare function bigint(params?: string | core.$ZodBigIntParams): ZodMiniBigInt<bigint>;
export interface ZodMiniBigIntFormat extends ZodMiniType {
    _zod: core.$ZodBigIntFormatInternals;
}
export declare const ZodMiniBigIntFormat: core.$constructor<ZodMiniBigIntFormat>;
export declare function int64(params?: string | core.$ZodBigIntFormatParams): ZodMiniBigIntFormat;
export declare function uint64(params?: string | core.$ZodBigIntFormatParams): ZodMiniBigIntFormat;
export interface ZodMiniSymbol extends ZodMiniType {
    _zod: core.$ZodSymbolInternals;
}
export declare const ZodMiniSymbol: core.$constructor<ZodMiniSymbol>;
export declare function symbol(params?: string | core.$ZodSymbolParams): ZodMiniSymbol;
export interface ZodMiniUndefined extends ZodMiniType {
    _zod: core.$ZodUndefinedInternals;
}
export declare const ZodMiniUndefined: core.$constructor<ZodMiniUndefined>;
declare function _undefined(params?: string | core.$ZodUndefinedParams): ZodMiniUndefined;
export { _undefined as undefined };
export interface ZodMiniNull extends ZodMiniType {
    _zod: core.$ZodNullInternals;
}
export declare const ZodMiniNull: core.$constructor<ZodMiniNull>;
declare function _null(params?: string | core.$ZodNullParams): ZodMiniNull;
export { _null as null };
export interface ZodMiniAny extends ZodMiniType {
    _zod: core.$ZodAnyInternals;
}
export declare const ZodMiniAny: core.$constructor<ZodMiniAny>;
export declare function any(): ZodMiniAny;
export interface ZodMiniUnknown extends ZodMiniType {
    _zod: core.$ZodUnknownInternals;
}
export declare const ZodMiniUnknown: core.$constructor<ZodMiniUnknown>;
export declare function unknown(): ZodMiniUnknown;
export interface ZodMiniNever extends ZodMiniType {
    _zod: core.$ZodNeverInternals;
}
export declare const ZodMiniNever: core.$constructor<ZodMiniNever>;
export declare function never(params?: string | core.$ZodNeverParams): ZodMiniNever;
export interface ZodMiniVoid extends ZodMiniType {
    _zod: core.$ZodVoidInternals;
}
export declare const ZodMiniVoid: core.$constructor<ZodMiniVoid>;
declare function _void(params?: string | core.$ZodVoidParams): ZodMiniVoid;
export { _void as void };
export interface ZodMiniDate<T = unknown> extends ZodMiniType {
    _zod: core.$ZodDateInternals<T>;
}
export declare const ZodMiniDate: core.$constructor<ZodMiniDate>;
export declare function date(params?: string | core.$ZodDateParams): ZodMiniDate<Date>;
export interface ZodMiniArray<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodArrayInternals<T>;
}
export declare const ZodMiniArray: core.$constructor<ZodMiniArray>;
export declare function array<T extends SomeType>(element: T, params?: string | core.$ZodArrayParams): ZodMiniArray<T>;
export declare function keyof<T extends ZodMiniObject>(schema: T): ZodMiniLiteral<keyof T["shape"]>;
export interface ZodMiniObject<
/** @ts-ignore Cast variance */
out Shape extends core.$ZodShape = core.$ZodShape, out Config extends core.$ZodObjectConfig = core.$ZodObjectConfig> extends ZodMiniType {
    _zod: core.$ZodObjectInternals<Shape, Config>;
    shape: Shape;
}
export declare const ZodMiniObject: core.$constructor<ZodMiniObject>;
export declare function object<T extends core.$ZodLooseShape = Record<never, SomeType>>(shape?: T, params?: string | core.$ZodObjectParams): ZodMiniObject<T, core.$strip>;
export declare function strictObject<T extends core.$ZodLooseShape>(shape: T, params?: string | core.$ZodObjectParams): ZodMiniObject<T, core.$strict>;
export declare function looseObject<T extends core.$ZodLooseShape>(shape: T, params?: string | core.$ZodObjectParams): ZodMiniObject<T, core.$loose>;
export declare function extend<T extends ZodMiniObject, U extends core.$ZodLooseShape>(schema: T, shape: U): ZodMiniObject<util.Extend<T["shape"], U>, T["_zod"]["config"]>;
/** @deprecated Identical to `z.extend(A, B)` */
export declare function merge<T extends ZodMiniObject, U extends ZodMiniObject>(a: T, b: U): ZodMiniObject<util.Extend<T["shape"], U["shape"]>, T["_zod"]["config"]>;
export declare function pick<T extends ZodMiniObject, M extends util.Mask<keyof T["shape"]>>(schema: T, mask: M): ZodMiniObject<util.Flatten<Pick<T["shape"], keyof T["shape"] & keyof M>>, T["_zod"]["config"]>;
export declare function omit<T extends ZodMiniObject, const M extends util.Mask<keyof T["shape"]>>(schema: T, mask: M): ZodMiniObject<util.Flatten<Omit<T["shape"], keyof M>>, T["_zod"]["config"]>;
export declare function partial<T extends ZodMiniObject>(schema: T): ZodMiniObject<{
    [k in keyof T["shape"]]: ZodMiniOptional<T["shape"][k]>;
}, T["_zod"]["config"]>;
export declare function partial<T extends ZodMiniObject, M extends util.Mask<keyof T["shape"]>>(schema: T, mask: M): ZodMiniObject<{
    [k in keyof T["shape"]]: k extends keyof M ? ZodMiniOptional<T["shape"][k]> : T["shape"][k];
}, T["_zod"]["config"]>;
export type RequiredInterfaceShape<Shape extends core.$ZodLooseShape, Keys extends PropertyKey = keyof Shape> = util.Identity<{
    [k in keyof Shape as k extends Keys ? k : never]: ZodMiniNonOptional<Shape[k]>;
} & {
    [k in keyof Shape as k extends Keys ? never : k]: Shape[k];
}>;
export declare function required<T extends ZodMiniObject>(schema: T): ZodMiniObject<{
    [k in keyof T["shape"]]: ZodMiniNonOptional<T["shape"][k]>;
}, T["_zod"]["config"]>;
export declare function required<T extends ZodMiniObject, M extends util.Mask<keyof T["shape"]>>(schema: T, mask: M): ZodMiniObject<util.Extend<T["shape"], {
    [k in keyof M & keyof T["shape"]]: ZodMiniNonOptional<T["shape"][k]>;
}>, T["_zod"]["config"]>;
export interface ZodMiniUnion<T extends readonly SomeType[] = readonly SomeType[]> extends ZodMiniType {
    _zod: core.$ZodUnionInternals<T>;
}
export declare const ZodMiniUnion: core.$constructor<ZodMiniUnion>;
export declare function union<const T extends readonly SomeType[]>(options: T, params?: string | core.$ZodUnionParams): ZodMiniUnion<T>;
export interface ZodMiniDiscriminatedUnion<Options extends readonly SomeType[] = readonly SomeType[]> extends ZodMiniUnion<Options> {
    _zod: core.$ZodDiscriminatedUnionInternals<Options>;
}
export declare const ZodMiniDiscriminatedUnion: core.$constructor<ZodMiniDiscriminatedUnion>;
export interface $ZodTypeDiscriminableInternals extends core.$ZodTypeInternals {
    disc: util.DiscriminatorMap;
}
export interface $ZodTypeDiscriminable extends ZodMiniType {
    _zod: $ZodTypeDiscriminableInternals;
}
export declare function discriminatedUnion<Types extends readonly [$ZodTypeDiscriminable, ...$ZodTypeDiscriminable[]]>(discriminator: string, options: Types, params?: string | core.$ZodDiscriminatedUnionParams): ZodMiniDiscriminatedUnion<Types>;
export interface ZodMiniIntersection<A extends SomeType = SomeType, B extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodIntersectionInternals<A, B>;
}
export declare const ZodMiniIntersection: core.$constructor<ZodMiniIntersection>;
export declare function intersection<T extends SomeType, U extends SomeType>(left: T, right: U): ZodMiniIntersection<T, U>;
export interface ZodMiniTuple<T extends util.TupleItems = util.TupleItems, Rest extends SomeType | null = SomeType | null> extends ZodMiniType {
    _zod: core.$ZodTupleInternals<T, Rest>;
}
export declare const ZodMiniTuple: core.$constructor<ZodMiniTuple>;
export declare function tuple<T extends readonly [SomeType, ...SomeType[]]>(items: T, params?: string | core.$ZodTupleParams): ZodMiniTuple<T, null>;
export declare function tuple<T extends readonly [SomeType, ...SomeType[]], Rest extends SomeType>(items: T, rest: Rest, params?: string | core.$ZodTupleParams): ZodMiniTuple<T, Rest>;
export declare function tuple(items: [], params?: string | core.$ZodTupleParams): ZodMiniTuple<[], null>;
export interface ZodMiniRecord<Key extends core.$ZodRecordKey = core.$ZodRecordKey, Value extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodRecordInternals<Key, Value>;
}
export declare const ZodMiniRecord: core.$constructor<ZodMiniRecord>;
export declare function record<Key extends core.$ZodRecordKey, Value extends SomeType>(keyType: Key, valueType: Value, params?: string | core.$ZodRecordParams): ZodMiniRecord<Key, Value>;
export declare function partialRecord<Key extends core.$ZodRecordKey, Value extends SomeType>(keyType: Key, valueType: Value, params?: string | core.$ZodRecordParams): ZodMiniRecord<ZodMiniUnion<[Key, ZodMiniNever]>, Value>;
export interface ZodMiniMap<Key extends SomeType = SomeType, Value extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodMapInternals<Key, Value>;
}
export declare const ZodMiniMap: core.$constructor<ZodMiniMap>;
export declare function map<Key extends SomeType, Value extends SomeType>(keyType: Key, valueType: Value, params?: string | core.$ZodMapParams): ZodMiniMap<Key, Value>;
export interface ZodMiniSet<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodSetInternals<T>;
}
export declare const ZodMiniSet: core.$constructor<ZodMiniSet>;
export declare function set<Value extends SomeType>(valueType: Value, params?: string | core.$ZodSetParams): ZodMiniSet<Value>;
export interface ZodMiniEnum<T extends util.EnumLike = util.EnumLike> extends ZodMiniType {
    _zod: core.$ZodEnumInternals<T>;
}
export declare const ZodMiniEnum: core.$constructor<ZodMiniEnum>;
declare function _enum<const T extends readonly string[]>(values: T, params?: string | core.$ZodEnumParams): ZodMiniEnum<util.ToEnum<T[number]>>;
declare function _enum<T extends util.EnumLike>(entries: T, params?: string | core.$ZodEnumParams): ZodMiniEnum<T>;
export { _enum as enum };
/** @deprecated This API has been merged into `z.enum()`. Use `z.enum()` instead.
 *
 * ```ts
 * enum Colors { red, green, blue }
 * z.enum(Colors);
 * ```
 */
export declare function nativeEnum<T extends util.EnumLike>(entries: T, params?: string | core.$ZodEnumParams): ZodMiniEnum<T>;
export interface ZodMiniLiteral<T extends util.Primitive = util.Primitive> extends ZodMiniType {
    _zod: core.$ZodLiteralInternals<T>;
}
export declare const ZodMiniLiteral: core.$constructor<ZodMiniLiteral>;
export declare function literal<const T extends Array<util.Literal>>(value: T, params?: string | core.$ZodLiteralParams): ZodMiniLiteral<T[number]>;
export declare function literal<const T extends util.Literal>(value: T, params?: string | core.$ZodLiteralParams): ZodMiniLiteral<T>;
export interface ZodMiniFile extends ZodMiniType {
    _zod: core.$ZodFileInternals;
}
export declare const ZodMiniFile: core.$constructor<ZodMiniFile>;
export declare function file(params?: string | core.$ZodFileParams): ZodMiniFile;
export interface ZodMiniTransform<O = unknown, I = unknown> extends ZodMiniType {
    _zod: core.$ZodTransformInternals<O, I>;
}
export declare const ZodMiniTransform: core.$constructor<ZodMiniTransform>;
export declare function transform<I = unknown, O = I>(fn: (input: I, ctx: core.ParsePayload) => O): ZodMiniTransform<Awaited<O>, I>;
export interface ZodMiniOptional<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodOptionalInternals<T>;
}
export declare const ZodMiniOptional: core.$constructor<ZodMiniOptional>;
export declare function optional<T extends SomeType>(innerType: T): ZodMiniOptional<T>;
export interface ZodMiniNullable<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodNullableInternals<T>;
}
export declare const ZodMiniNullable: core.$constructor<ZodMiniNullable>;
export declare function nullable<T extends SomeType>(innerType: T): ZodMiniNullable<T>;
export declare function nullish<T extends SomeType>(innerType: T): ZodMiniOptional<ZodMiniNullable<T>>;
export interface ZodMiniDefault<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodDefaultInternals<T>;
}
export declare const ZodMiniDefault: core.$constructor<ZodMiniDefault>;
export declare function _default<T extends SomeType>(innerType: T, defaultValue: util.NoUndefined<core.output<T>> | (() => util.NoUndefined<core.output<T>>)): ZodMiniDefault<T>;
export interface ZodMiniPrefault<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodPrefaultInternals<T>;
}
export declare const ZodMiniPrefault: core.$constructor<ZodMiniPrefault>;
export declare function prefault<T extends SomeType>(innerType: T, defaultValue: util.NoUndefined<core.input<T>> | (() => util.NoUndefined<core.input<T>>)): ZodMiniPrefault<T>;
export interface ZodMiniNonOptional<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodNonOptionalInternals<T>;
}
export declare const ZodMiniNonOptional: core.$constructor<ZodMiniNonOptional>;
export declare function nonoptional<T extends SomeType>(innerType: T, params?: string | core.$ZodNonOptionalParams): ZodMiniNonOptional<T>;
export interface ZodMiniSuccess<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodSuccessInternals<T>;
}
export declare const ZodMiniSuccess: core.$constructor<ZodMiniSuccess>;
export declare function success<T extends SomeType>(innerType: T): ZodMiniSuccess<T>;
export interface ZodMiniCatch<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodCatchInternals<T>;
}
export declare const ZodMiniCatch: core.$constructor<ZodMiniCatch>;
declare function _catch<T extends SomeType>(innerType: T, catchValue: core.output<T> | ((ctx: core.$ZodCatchCtx) => core.output<T>)): ZodMiniCatch<T>;
export { _catch as catch };
export interface ZodMiniNaN extends ZodMiniType {
    _zod: core.$ZodNaNInternals;
}
export declare const ZodMiniNaN: core.$constructor<ZodMiniNaN>;
export declare function nan(params?: string | core.$ZodNaNParams): ZodMiniNaN;
export interface ZodMiniPipe<A extends SomeType = SomeType, B extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodPipeInternals<A, B>;
}
export declare const ZodMiniPipe: core.$constructor<ZodMiniPipe>;
export declare function pipe<const A extends core.$ZodType, B extends core.$ZodType<unknown, core.output<A>> = core.$ZodType<unknown, core.output<A>>>(in_: A, out: B | core.$ZodType<unknown, core.output<A>>, params?: string | core.$ZodPipeParams): ZodMiniPipe<A, B>;
export interface ZodMiniReadonly<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodReadonlyInternals<T>;
}
export declare const ZodMiniReadonly: core.$constructor<ZodMiniReadonly>;
export declare function readonly<T extends SomeType>(innerType: T): ZodMiniReadonly<T>;
export interface ZodMiniTemplateLiteral<Template extends string = string> extends ZodMiniType {
    _zod: core.$ZodTemplateLiteralInternals<Template>;
}
export declare const ZodMiniTemplateLiteral: core.$constructor<ZodMiniTemplateLiteral>;
export declare function templateLiteral<const Parts extends core.$ZodTemplateLiteralPart[]>(parts: Parts, params?: string | core.$ZodTemplateLiteralParams): ZodMiniTemplateLiteral<core.$PartsToTemplateLiteral<Parts>>;
export interface ZodMiniLazy<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodLazyInternals<T>;
}
export declare const ZodMiniLazy: core.$constructor<ZodMiniLazy>;
declare function _lazy<T extends SomeType>(getter: () => T): ZodMiniLazy<T>;
export { _lazy as lazy };
export interface ZodMiniPromise<T extends SomeType = SomeType> extends ZodMiniType {
    _zod: core.$ZodPromiseInternals<T>;
}
export declare const ZodMiniPromise: core.$constructor<ZodMiniPromise>;
export declare function promise<T extends SomeType>(innerType: T): ZodMiniPromise<T>;
export interface ZodMiniCustom<O = unknown, I = unknown> extends ZodMiniType {
    _zod: core.$ZodCustomInternals<O, I>;
}
export declare const ZodMiniCustom: core.$constructor<ZodMiniCustom>;
export declare function check<O = unknown>(fn: core.CheckFn<O>, params?: string | core.$ZodCustomParams): core.$ZodCheck<O>;
export declare function refine<T>(fn: (arg: NoInfer<T>) => util.MaybeAsync<unknown>, _params?: string | core.$ZodCustomParams): core.$ZodCheck<T>;
export declare function custom<O = unknown, I = O>(fn?: (data: O) => unknown, _params?: string | core.$ZodCustomParams | undefined): ZodMiniCustom<O, I>;
declare abstract class Class {
    constructor(..._args: any[]);
}
declare function _instanceof<T extends typeof Class>(cls: T, params?: core.$ZodCustomParams): ZodMiniCustom<InstanceType<T>>;
export { _instanceof as instanceof };
export declare const stringbool: (_params?: string | core.$ZodStringBoolParams) => ZodMiniPipe<ZodMiniUnknown, ZodMiniBoolean<boolean>>;
export type ZodMiniJSONSchema = ZodMiniLazy<ZodMiniUnion<[
    ZodMiniString<string>,
    ZodMiniNumber<number>,
    ZodMiniBoolean<boolean>,
    ZodMiniNull,
    ZodMiniArray<ZodMiniJSONSchema>,
    ZodMiniRecord<ZodMiniString<string>, ZodMiniJSONSchema>
]>> & {
    _zod: {
        input: util.JSONType;
        output: util.JSONType;
    };
};
export declare function json(): ZodMiniJSONSchema;
