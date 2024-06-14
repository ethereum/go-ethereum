"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.PredictionContextCache = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:35.6390614-07:00
const Array2DHashMap_1 = require("../misc/Array2DHashMap");
const Decorators_1 = require("../Decorators");
const ObjectEqualityComparator_1 = require("../misc/ObjectEqualityComparator");
const PredictionContext_1 = require("./PredictionContext");
const assert = require("assert");
/** Used to cache {@link PredictionContext} objects. Its used for the shared
 *  context cash associated with contexts in DFA states. This cache
 *  can be used for both lexers and parsers.
 *
 * @author Sam Harwell
 */
class PredictionContextCache {
    constructor(enableCache = true) {
        this.contexts = new Array2DHashMap_1.Array2DHashMap(ObjectEqualityComparator_1.ObjectEqualityComparator.INSTANCE);
        this.childContexts = new Array2DHashMap_1.Array2DHashMap(ObjectEqualityComparator_1.ObjectEqualityComparator.INSTANCE);
        this.joinContexts = new Array2DHashMap_1.Array2DHashMap(ObjectEqualityComparator_1.ObjectEqualityComparator.INSTANCE);
        this.enableCache = enableCache;
    }
    getAsCached(context) {
        if (!this.enableCache) {
            return context;
        }
        let result = this.contexts.get(context);
        if (!result) {
            result = context;
            this.contexts.put(context, context);
        }
        return result;
    }
    getChild(context, invokingState) {
        if (!this.enableCache) {
            return context.getChild(invokingState);
        }
        let operands = new PredictionContextCache.PredictionContextAndInt(context, invokingState);
        let result = this.childContexts.get(operands);
        if (!result) {
            result = context.getChild(invokingState);
            result = this.getAsCached(result);
            this.childContexts.put(operands, result);
        }
        return result;
    }
    join(x, y) {
        if (!this.enableCache) {
            return PredictionContext_1.PredictionContext.join(x, y, this);
        }
        let operands = new PredictionContextCache.IdentityCommutativePredictionContextOperands(x, y);
        let result = this.joinContexts.get(operands);
        if (result) {
            return result;
        }
        result = PredictionContext_1.PredictionContext.join(x, y, this);
        result = this.getAsCached(result);
        this.joinContexts.put(operands, result);
        return result;
    }
}
exports.PredictionContextCache = PredictionContextCache;
PredictionContextCache.UNCACHED = new PredictionContextCache(false);
(function (PredictionContextCache) {
    class PredictionContextAndInt {
        constructor(obj, value) {
            this.obj = obj;
            this.value = value;
        }
        equals(obj) {
            if (!(obj instanceof PredictionContextAndInt)) {
                return false;
            }
            else if (obj === this) {
                return true;
            }
            let other = obj;
            return this.value === other.value
                && (this.obj === other.obj || (this.obj != null && this.obj.equals(other.obj)));
        }
        hashCode() {
            let hashCode = 5;
            hashCode = 7 * hashCode + (this.obj != null ? this.obj.hashCode() : 0);
            hashCode = 7 * hashCode + this.value;
            return hashCode;
        }
    }
    __decorate([
        Decorators_1.Override
    ], PredictionContextAndInt.prototype, "equals", null);
    __decorate([
        Decorators_1.Override
    ], PredictionContextAndInt.prototype, "hashCode", null);
    PredictionContextCache.PredictionContextAndInt = PredictionContextAndInt;
    class IdentityCommutativePredictionContextOperands {
        constructor(x, y) {
            assert(x != null);
            assert(y != null);
            this._x = x;
            this._y = y;
        }
        get x() {
            return this._x;
        }
        get y() {
            return this._y;
        }
        equals(o) {
            if (!(o instanceof IdentityCommutativePredictionContextOperands)) {
                return false;
            }
            else if (this === o) {
                return true;
            }
            let other = o;
            return (this._x === other._x && this._y === other._y) || (this._x === other._y && this._y === other._x);
        }
        hashCode() {
            return this._x.hashCode() ^ this._y.hashCode();
        }
    }
    __decorate([
        Decorators_1.Override
    ], IdentityCommutativePredictionContextOperands.prototype, "equals", null);
    __decorate([
        Decorators_1.Override
    ], IdentityCommutativePredictionContextOperands.prototype, "hashCode", null);
    PredictionContextCache.IdentityCommutativePredictionContextOperands = IdentityCommutativePredictionContextOperands;
})(PredictionContextCache = exports.PredictionContextCache || (exports.PredictionContextCache = {}));
//# sourceMappingURL=PredictionContextCache.js.map