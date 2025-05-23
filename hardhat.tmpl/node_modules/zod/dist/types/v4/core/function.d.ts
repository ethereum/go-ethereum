import * as schemas from "./schemas.js";
import { $ZodTuple } from "./schemas.js";
import type * as util from "./util.js";
export interface $ZodFunctionDef {
    type: "function";
    input: $ZodFunctionArgs | null;
    output: schemas.$ZodType | null;
}
export type $ZodFunctionArgs = schemas.$ZodType<unknown[], unknown[]>;
export type $InferInnerFunctionType<Args extends $ZodFunctionArgs, Returns extends schemas.$ZodType> = (...args: Args["_zod"]["output"]) => Returns["_zod"]["input"];
export type $InferInnerFunctionTypeAsync<Args extends $ZodFunctionArgs, Returns extends schemas.$ZodType> = (...args: Args["_zod"]["output"]) => util.MaybeAsync<Returns["_zod"]["input"]>;
export type $InferOuterFunctionType<Args extends $ZodFunctionArgs, Returns extends schemas.$ZodType> = (...args: Args["_zod"]["input"]) => Returns["_zod"]["output"];
export type $InferOuterFunctionTypeAsync<Args extends $ZodFunctionArgs, Returns extends schemas.$ZodType> = (...args: Args["_zod"]["input"]) => util.MaybeAsync<Returns["_zod"]["output"]>;
export declare class $ZodFunction<Args extends $ZodFunctionArgs = $ZodFunctionArgs, Returns extends schemas.$ZodType = schemas.$ZodType> {
    _def: $ZodFunctionDef;
    _input: $InferInnerFunctionType<Args, Returns>;
    _output: $InferOuterFunctionType<Args, Returns>;
    constructor(def: $ZodFunctionDef);
    implement<F extends $InferInnerFunctionType<Args, Returns>>(func: F): F extends this["_output"] ? F : this["_output"];
    implementAsync<F extends $InferInnerFunctionTypeAsync<Args, Returns>>(func: F): F extends $InferOuterFunctionTypeAsync<Args, Returns> ? F : $InferOuterFunctionTypeAsync<Args, Returns>;
    input<const Items extends util.TupleItems, const Rest extends schemas.$ZodType | null = null>(args: Items, rest?: Rest): $ZodFunction<schemas.$ZodTuple<Items, Rest>, Returns>;
    input<NewArgs extends $ZodFunctionArgs>(args: NewArgs): $ZodFunction<NewArgs, Returns>;
    output<NewReturns extends schemas.$ZodType>(output: NewReturns): $ZodFunction<Args, NewReturns>;
}
export interface $ZodFunctionParams<I extends $ZodFunctionArgs, O extends schemas.$ZodType> {
    input?: I;
    output?: O;
}
declare function _function(): $ZodFunction;
declare function _function<const In extends Array<schemas.$ZodType> = Array<schemas.$ZodType>, Out extends schemas.$ZodType = schemas.$ZodType>(params?: {
    input?: In;
    output?: Out;
}): $ZodFunction<$ZodTuple<In, null>, Out>;
declare function _function<In extends $ZodFunctionArgs = $ZodFunctionArgs, Out extends schemas.$ZodType = schemas.$ZodType>(params?: $ZodFunctionParams<In, Out>): $ZodFunction<In, Out>;
export { _function as function };
