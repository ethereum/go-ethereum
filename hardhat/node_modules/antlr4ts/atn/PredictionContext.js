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
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.SingletonPredictionContext = exports.PredictionContext = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:35.3812636-07:00
const Array2DHashMap_1 = require("../misc/Array2DHashMap");
const Array2DHashSet_1 = require("../misc/Array2DHashSet");
const Arrays_1 = require("../misc/Arrays");
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
const PredictionContextCache_1 = require("./PredictionContextCache");
const assert = require("assert");
const INITIAL_HASH = 1;
class PredictionContext {
    constructor(cachedHashCode) {
        this.cachedHashCode = cachedHashCode;
    }
    static calculateEmptyHashCode() {
        let hash = MurmurHash_1.MurmurHash.initialize(INITIAL_HASH);
        hash = MurmurHash_1.MurmurHash.finish(hash, 0);
        return hash;
    }
    static calculateSingleHashCode(parent, returnState) {
        let hash = MurmurHash_1.MurmurHash.initialize(INITIAL_HASH);
        hash = MurmurHash_1.MurmurHash.update(hash, parent);
        hash = MurmurHash_1.MurmurHash.update(hash, returnState);
        hash = MurmurHash_1.MurmurHash.finish(hash, 2);
        return hash;
    }
    static calculateHashCode(parents, returnStates) {
        let hash = MurmurHash_1.MurmurHash.initialize(INITIAL_HASH);
        for (let parent of parents) {
            hash = MurmurHash_1.MurmurHash.update(hash, parent);
        }
        for (let returnState of returnStates) {
            hash = MurmurHash_1.MurmurHash.update(hash, returnState);
        }
        hash = MurmurHash_1.MurmurHash.finish(hash, 2 * parents.length);
        return hash;
    }
    static fromRuleContext(atn, outerContext, fullContext = true) {
        if (outerContext.isEmpty) {
            return fullContext ? PredictionContext.EMPTY_FULL : PredictionContext.EMPTY_LOCAL;
        }
        let parent;
        if (outerContext._parent) {
            parent = PredictionContext.fromRuleContext(atn, outerContext._parent, fullContext);
        }
        else {
            parent = fullContext ? PredictionContext.EMPTY_FULL : PredictionContext.EMPTY_LOCAL;
        }
        let state = atn.states[outerContext.invokingState];
        let transition = state.transition(0);
        return parent.getChild(transition.followState.stateNumber);
    }
    static addEmptyContext(context) {
        return context.addEmptyContext();
    }
    static removeEmptyContext(context) {
        return context.removeEmptyContext();
    }
    static join(context0, context1, contextCache = PredictionContextCache_1.PredictionContextCache.UNCACHED) {
        if (context0 === context1) {
            return context0;
        }
        if (context0.isEmpty) {
            return PredictionContext.isEmptyLocal(context0) ? context0 : PredictionContext.addEmptyContext(context1);
        }
        else if (context1.isEmpty) {
            return PredictionContext.isEmptyLocal(context1) ? context1 : PredictionContext.addEmptyContext(context0);
        }
        let context0size = context0.size;
        let context1size = context1.size;
        if (context0size === 1 && context1size === 1 && context0.getReturnState(0) === context1.getReturnState(0)) {
            let merged = contextCache.join(context0.getParent(0), context1.getParent(0));
            if (merged === context0.getParent(0)) {
                return context0;
            }
            else if (merged === context1.getParent(0)) {
                return context1;
            }
            else {
                return merged.getChild(context0.getReturnState(0));
            }
        }
        let count = 0;
        let parentsList = new Array(context0size + context1size);
        let returnStatesList = new Array(parentsList.length);
        let leftIndex = 0;
        let rightIndex = 0;
        let canReturnLeft = true;
        let canReturnRight = true;
        while (leftIndex < context0size && rightIndex < context1size) {
            if (context0.getReturnState(leftIndex) === context1.getReturnState(rightIndex)) {
                parentsList[count] = contextCache.join(context0.getParent(leftIndex), context1.getParent(rightIndex));
                returnStatesList[count] = context0.getReturnState(leftIndex);
                canReturnLeft = canReturnLeft && parentsList[count] === context0.getParent(leftIndex);
                canReturnRight = canReturnRight && parentsList[count] === context1.getParent(rightIndex);
                leftIndex++;
                rightIndex++;
            }
            else if (context0.getReturnState(leftIndex) < context1.getReturnState(rightIndex)) {
                parentsList[count] = context0.getParent(leftIndex);
                returnStatesList[count] = context0.getReturnState(leftIndex);
                canReturnRight = false;
                leftIndex++;
            }
            else {
                assert(context1.getReturnState(rightIndex) < context0.getReturnState(leftIndex));
                parentsList[count] = context1.getParent(rightIndex);
                returnStatesList[count] = context1.getReturnState(rightIndex);
                canReturnLeft = false;
                rightIndex++;
            }
            count++;
        }
        while (leftIndex < context0size) {
            parentsList[count] = context0.getParent(leftIndex);
            returnStatesList[count] = context0.getReturnState(leftIndex);
            leftIndex++;
            canReturnRight = false;
            count++;
        }
        while (rightIndex < context1size) {
            parentsList[count] = context1.getParent(rightIndex);
            returnStatesList[count] = context1.getReturnState(rightIndex);
            rightIndex++;
            canReturnLeft = false;
            count++;
        }
        if (canReturnLeft) {
            return context0;
        }
        else if (canReturnRight) {
            return context1;
        }
        if (count < parentsList.length) {
            parentsList = parentsList.slice(0, count);
            returnStatesList = returnStatesList.slice(0, count);
        }
        if (parentsList.length === 0) {
            // if one of them was EMPTY_LOCAL, it would be empty and handled at the beginning of the method
            return PredictionContext.EMPTY_FULL;
        }
        else if (parentsList.length === 1) {
            return new SingletonPredictionContext(parentsList[0], returnStatesList[0]);
        }
        else {
            return new ArrayPredictionContext(parentsList, returnStatesList);
        }
    }
    static isEmptyLocal(context) {
        return context === PredictionContext.EMPTY_LOCAL;
    }
    static getCachedContext(context, contextCache, visited) {
        if (context.isEmpty) {
            return context;
        }
        let existing = visited.get(context);
        if (existing) {
            return existing;
        }
        existing = contextCache.get(context);
        if (existing) {
            visited.put(context, existing);
            return existing;
        }
        let changed = false;
        let parents = new Array(context.size);
        for (let i = 0; i < parents.length; i++) {
            let parent = PredictionContext.getCachedContext(context.getParent(i), contextCache, visited);
            if (changed || parent !== context.getParent(i)) {
                if (!changed) {
                    parents = new Array(context.size);
                    for (let j = 0; j < context.size; j++) {
                        parents[j] = context.getParent(j);
                    }
                    changed = true;
                }
                parents[i] = parent;
            }
        }
        if (!changed) {
            existing = contextCache.putIfAbsent(context, context);
            visited.put(context, existing != null ? existing : context);
            return context;
        }
        // We know parents.length>0 because context.isEmpty is checked at the beginning of the method.
        let updated;
        if (parents.length === 1) {
            updated = new SingletonPredictionContext(parents[0], context.getReturnState(0));
        }
        else {
            let returnStates = new Array(context.size);
            for (let i = 0; i < context.size; i++) {
                returnStates[i] = context.getReturnState(i);
            }
            updated = new ArrayPredictionContext(parents, returnStates, context.hashCode());
        }
        existing = contextCache.putIfAbsent(updated, updated);
        visited.put(updated, existing || updated);
        visited.put(context, existing || updated);
        return updated;
    }
    appendSingleContext(returnContext, contextCache) {
        return this.appendContext(PredictionContext.EMPTY_FULL.getChild(returnContext), contextCache);
    }
    getChild(returnState) {
        return new SingletonPredictionContext(this, returnState);
    }
    hashCode() {
        return this.cachedHashCode;
    }
    toStrings(recognizer, currentState, stop = PredictionContext.EMPTY_FULL) {
        let result = [];
        outer: for (let perm = 0;; perm++) {
            let offset = 0;
            let last = true;
            let p = this;
            let stateNumber = currentState;
            let localBuffer = "";
            localBuffer += "[";
            while (!p.isEmpty && p !== stop) {
                let index = 0;
                if (p.size > 0) {
                    let bits = 1;
                    while (((1 << bits) >>> 0) < p.size) {
                        bits++;
                    }
                    let mask = ((1 << bits) >>> 0) - 1;
                    index = (perm >> offset) & mask;
                    last = last && index >= p.size - 1;
                    if (index >= p.size) {
                        continue outer;
                    }
                    offset += bits;
                }
                if (recognizer) {
                    if (localBuffer.length > 1) {
                        // first char is '[', if more than that this isn't the first rule
                        localBuffer += " ";
                    }
                    let atn = recognizer.atn;
                    let s = atn.states[stateNumber];
                    let ruleName = recognizer.ruleNames[s.ruleIndex];
                    localBuffer += ruleName;
                }
                else if (p.getReturnState(index) !== PredictionContext.EMPTY_FULL_STATE_KEY) {
                    if (!p.isEmpty) {
                        if (localBuffer.length > 1) {
                            // first char is '[', if more than that this isn't the first rule
                            localBuffer += " ";
                        }
                        localBuffer += p.getReturnState(index);
                    }
                }
                stateNumber = p.getReturnState(index);
                p = p.getParent(index);
            }
            localBuffer += "]";
            result.push(localBuffer);
            if (last) {
                break;
            }
        }
        return result;
    }
}
__decorate([
    Decorators_1.Override
], PredictionContext.prototype, "hashCode", null);
__decorate([
    __param(0, Decorators_1.NotNull), __param(1, Decorators_1.NotNull), __param(2, Decorators_1.NotNull)
], PredictionContext, "join", null);
__decorate([
    __param(0, Decorators_1.NotNull),
    __param(1, Decorators_1.NotNull),
    __param(2, Decorators_1.NotNull)
], PredictionContext, "getCachedContext", null);
exports.PredictionContext = PredictionContext;
class EmptyPredictionContext extends PredictionContext {
    constructor(fullContext) {
        super(PredictionContext.calculateEmptyHashCode());
        this.fullContext = fullContext;
    }
    get isFullContext() {
        return this.fullContext;
    }
    addEmptyContext() {
        return this;
    }
    removeEmptyContext() {
        throw new Error("Cannot remove the empty context from itself.");
    }
    getParent(index) {
        throw new Error("index out of bounds");
    }
    getReturnState(index) {
        throw new Error("index out of bounds");
    }
    findReturnState(returnState) {
        return -1;
    }
    get size() {
        return 0;
    }
    appendSingleContext(returnContext, contextCache) {
        return contextCache.getChild(this, returnContext);
    }
    appendContext(suffix, contextCache) {
        return suffix;
    }
    get isEmpty() {
        return true;
    }
    get hasEmpty() {
        return true;
    }
    equals(o) {
        return this === o;
    }
    toStrings(recognizer, currentState, stop) {
        return ["[]"];
    }
}
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "addEmptyContext", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "removeEmptyContext", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "getParent", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "getReturnState", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "findReturnState", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "size", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "appendSingleContext", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "appendContext", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "isEmpty", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "hasEmpty", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "equals", null);
__decorate([
    Decorators_1.Override
], EmptyPredictionContext.prototype, "toStrings", null);
let ArrayPredictionContext = class ArrayPredictionContext extends PredictionContext {
    constructor(parents, returnStates, hashCode) {
        super(hashCode || PredictionContext.calculateHashCode(parents, returnStates));
        assert(parents.length === returnStates.length);
        assert(returnStates.length > 1 || returnStates[0] !== PredictionContext.EMPTY_FULL_STATE_KEY, "Should be using PredictionContext.EMPTY instead.");
        this.parents = parents;
        this.returnStates = returnStates;
    }
    getParent(index) {
        return this.parents[index];
    }
    getReturnState(index) {
        return this.returnStates[index];
    }
    findReturnState(returnState) {
        return Arrays_1.Arrays.binarySearch(this.returnStates, returnState);
    }
    get size() {
        return this.returnStates.length;
    }
    get isEmpty() {
        return false;
    }
    get hasEmpty() {
        return this.returnStates[this.returnStates.length - 1] === PredictionContext.EMPTY_FULL_STATE_KEY;
    }
    addEmptyContext() {
        if (this.hasEmpty) {
            return this;
        }
        let parents2 = this.parents.slice(0);
        let returnStates2 = this.returnStates.slice(0);
        parents2.push(PredictionContext.EMPTY_FULL);
        returnStates2.push(PredictionContext.EMPTY_FULL_STATE_KEY);
        return new ArrayPredictionContext(parents2, returnStates2);
    }
    removeEmptyContext() {
        if (!this.hasEmpty) {
            return this;
        }
        if (this.returnStates.length === 2) {
            return new SingletonPredictionContext(this.parents[0], this.returnStates[0]);
        }
        else {
            let parents2 = this.parents.slice(0, this.parents.length - 1);
            let returnStates2 = this.returnStates.slice(0, this.returnStates.length - 1);
            return new ArrayPredictionContext(parents2, returnStates2);
        }
    }
    appendContext(suffix, contextCache) {
        return ArrayPredictionContext.appendContextImpl(this, suffix, new PredictionContext.IdentityHashMap());
    }
    static appendContextImpl(context, suffix, visited) {
        if (suffix.isEmpty) {
            if (PredictionContext.isEmptyLocal(suffix)) {
                if (context.hasEmpty) {
                    return PredictionContext.EMPTY_LOCAL;
                }
                throw new Error("what to do here?");
            }
            return context;
        }
        if (suffix.size !== 1) {
            throw new Error("Appending a tree suffix is not yet supported.");
        }
        let result = visited.get(context);
        if (!result) {
            if (context.isEmpty) {
                result = suffix;
            }
            else {
                let parentCount = context.size;
                if (context.hasEmpty) {
                    parentCount--;
                }
                let updatedParents = new Array(parentCount);
                let updatedReturnStates = new Array(parentCount);
                for (let i = 0; i < parentCount; i++) {
                    updatedReturnStates[i] = context.getReturnState(i);
                }
                for (let i = 0; i < parentCount; i++) {
                    updatedParents[i] = ArrayPredictionContext.appendContextImpl(context.getParent(i), suffix, visited);
                }
                if (updatedParents.length === 1) {
                    result = new SingletonPredictionContext(updatedParents[0], updatedReturnStates[0]);
                }
                else {
                    assert(updatedParents.length > 1);
                    result = new ArrayPredictionContext(updatedParents, updatedReturnStates);
                }
                if (context.hasEmpty) {
                    result = PredictionContext.join(result, suffix);
                }
            }
            visited.put(context, result);
        }
        return result;
    }
    equals(o) {
        if (this === o) {
            return true;
        }
        else if (!(o instanceof ArrayPredictionContext)) {
            return false;
        }
        if (this.hashCode() !== o.hashCode()) {
            // can't be same if hash is different
            return false;
        }
        let other = o;
        return this.equalsImpl(other, new Array2DHashSet_1.Array2DHashSet());
    }
    equalsImpl(other, visited) {
        let selfWorkList = [];
        let otherWorkList = [];
        selfWorkList.push(this);
        otherWorkList.push(other);
        while (true) {
            let currentSelf = selfWorkList.pop();
            let currentOther = otherWorkList.pop();
            if (!currentSelf || !currentOther) {
                break;
            }
            let operands = new PredictionContextCache_1.PredictionContextCache.IdentityCommutativePredictionContextOperands(currentSelf, currentOther);
            if (!visited.add(operands)) {
                continue;
            }
            let selfSize = operands.x.size;
            if (selfSize === 0) {
                if (!operands.x.equals(operands.y)) {
                    return false;
                }
                continue;
            }
            let otherSize = operands.y.size;
            if (selfSize !== otherSize) {
                return false;
            }
            for (let i = 0; i < selfSize; i++) {
                if (operands.x.getReturnState(i) !== operands.y.getReturnState(i)) {
                    return false;
                }
                let selfParent = operands.x.getParent(i);
                let otherParent = operands.y.getParent(i);
                if (selfParent.hashCode() !== otherParent.hashCode()) {
                    return false;
                }
                if (selfParent !== otherParent) {
                    selfWorkList.push(selfParent);
                    otherWorkList.push(otherParent);
                }
            }
        }
        return true;
    }
};
__decorate([
    Decorators_1.NotNull
], ArrayPredictionContext.prototype, "parents", void 0);
__decorate([
    Decorators_1.NotNull
], ArrayPredictionContext.prototype, "returnStates", void 0);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "getParent", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "getReturnState", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "findReturnState", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "size", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "isEmpty", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "hasEmpty", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "addEmptyContext", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "removeEmptyContext", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "appendContext", null);
__decorate([
    Decorators_1.Override
], ArrayPredictionContext.prototype, "equals", null);
ArrayPredictionContext = __decorate([
    __param(0, Decorators_1.NotNull)
], ArrayPredictionContext);
let SingletonPredictionContext = class SingletonPredictionContext extends PredictionContext {
    constructor(parent, returnState) {
        super(PredictionContext.calculateSingleHashCode(parent, returnState));
        // assert(returnState != PredictionContext.EMPTY_FULL_STATE_KEY && returnState != PredictionContext.EMPTY_LOCAL_STATE_KEY);
        this.parent = parent;
        this.returnState = returnState;
    }
    getParent(index) {
        // assert(index == 0);
        return this.parent;
    }
    getReturnState(index) {
        // assert(index == 0);
        return this.returnState;
    }
    findReturnState(returnState) {
        return this.returnState === returnState ? 0 : -1;
    }
    get size() {
        return 1;
    }
    get isEmpty() {
        return false;
    }
    get hasEmpty() {
        return false;
    }
    appendContext(suffix, contextCache) {
        return contextCache.getChild(this.parent.appendContext(suffix, contextCache), this.returnState);
    }
    addEmptyContext() {
        let parents = [this.parent, PredictionContext.EMPTY_FULL];
        let returnStates = [this.returnState, PredictionContext.EMPTY_FULL_STATE_KEY];
        return new ArrayPredictionContext(parents, returnStates);
    }
    removeEmptyContext() {
        return this;
    }
    equals(o) {
        if (o === this) {
            return true;
        }
        else if (!(o instanceof SingletonPredictionContext)) {
            return false;
        }
        let other = o;
        if (this.hashCode() !== other.hashCode()) {
            return false;
        }
        return this.returnState === other.returnState
            && this.parent.equals(other.parent);
    }
};
__decorate([
    Decorators_1.NotNull
], SingletonPredictionContext.prototype, "parent", void 0);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "getParent", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "getReturnState", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "findReturnState", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "size", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "isEmpty", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "hasEmpty", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "appendContext", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "addEmptyContext", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "removeEmptyContext", null);
__decorate([
    Decorators_1.Override
], SingletonPredictionContext.prototype, "equals", null);
SingletonPredictionContext = __decorate([
    __param(0, Decorators_1.NotNull)
], SingletonPredictionContext);
exports.SingletonPredictionContext = SingletonPredictionContext;
(function (PredictionContext) {
    PredictionContext.EMPTY_LOCAL = new EmptyPredictionContext(false);
    PredictionContext.EMPTY_FULL = new EmptyPredictionContext(true);
    PredictionContext.EMPTY_LOCAL_STATE_KEY = -((1 << 31) >>> 0);
    PredictionContext.EMPTY_FULL_STATE_KEY = ((1 << 31) >>> 0) - 1;
    class IdentityHashMap extends Array2DHashMap_1.Array2DHashMap {
        constructor() {
            super(IdentityEqualityComparator.INSTANCE);
        }
    }
    PredictionContext.IdentityHashMap = IdentityHashMap;
    class IdentityEqualityComparator {
        IdentityEqualityComparator() {
            // intentionally empty
        }
        hashCode(obj) {
            return obj.hashCode();
        }
        equals(a, b) {
            return a === b;
        }
    }
    IdentityEqualityComparator.INSTANCE = new IdentityEqualityComparator();
    __decorate([
        Decorators_1.Override
    ], IdentityEqualityComparator.prototype, "hashCode", null);
    __decorate([
        Decorators_1.Override
    ], IdentityEqualityComparator.prototype, "equals", null);
    PredictionContext.IdentityEqualityComparator = IdentityEqualityComparator;
})(PredictionContext = exports.PredictionContext || (exports.PredictionContext = {}));
//# sourceMappingURL=PredictionContext.js.map