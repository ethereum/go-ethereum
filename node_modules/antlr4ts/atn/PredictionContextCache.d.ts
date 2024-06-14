/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { PredictionContext } from "./PredictionContext";
/** Used to cache {@link PredictionContext} objects. Its used for the shared
 *  context cash associated with contexts in DFA states. This cache
 *  can be used for both lexers and parsers.
 *
 * @author Sam Harwell
 */
export declare class PredictionContextCache {
    static UNCACHED: PredictionContextCache;
    private contexts;
    private childContexts;
    private joinContexts;
    private enableCache;
    constructor(enableCache?: boolean);
    getAsCached(context: PredictionContext): PredictionContext;
    getChild(context: PredictionContext, invokingState: number): PredictionContext;
    join(x: PredictionContext, y: PredictionContext): PredictionContext;
}
export declare namespace PredictionContextCache {
    class PredictionContextAndInt {
        private obj;
        private value;
        constructor(obj: PredictionContext, value: number);
        equals(obj: any): boolean;
        hashCode(): number;
    }
    class IdentityCommutativePredictionContextOperands {
        private _x;
        private _y;
        constructor(x: PredictionContext, y: PredictionContext);
        get x(): PredictionContext;
        get y(): PredictionContext;
        equals(o: any): boolean;
        hashCode(): number;
    }
}
