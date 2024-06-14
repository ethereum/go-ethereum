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
exports.DFAState = void 0;
const ATN_1 = require("../atn/ATN");
const BitSet_1 = require("../misc/BitSet");
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
const PredictionContext_1 = require("../atn/PredictionContext");
const assert = require("assert");
/** A DFA state represents a set of possible ATN configurations.
 *  As Aho, Sethi, Ullman p. 117 says "The DFA uses its state
 *  to keep track of all possible states the ATN can be in after
 *  reading each input symbol.  That is to say, after reading
 *  input a1a2..an, the DFA is in a state that represents the
 *  subset T of the states of the ATN that are reachable from the
 *  ATN's start state along some path labeled a1a2..an."
 *  In conventional NFA&rarr;DFA conversion, therefore, the subset T
 *  would be a bitset representing the set of states the
 *  ATN could be in.  We need to track the alt predicted by each
 *  state as well, however.  More importantly, we need to maintain
 *  a stack of states, tracking the closure operations as they
 *  jump from rule to rule, emulating rule invocations (method calls).
 *  I have to add a stack to simulate the proper lookahead sequences for
 *  the underlying LL grammar from which the ATN was derived.
 *
 *  I use a set of ATNConfig objects not simple states.  An ATNConfig
 *  is both a state (ala normal conversion) and a RuleContext describing
 *  the chain of rules (if any) followed to arrive at that state.
 *
 *  A DFA state may have multiple references to a particular state,
 *  but with different ATN contexts (with same or different alts)
 *  meaning that state was reached via a different set of rule invocations.
 */
class DFAState {
    /**
     * Constructs a new `DFAState`.
     *
     * @param configs The set of ATN configurations defining this state.
     */
    constructor(configs) {
        this.stateNumber = -1;
        this.configs = configs;
        this.edges = new Map();
        this.contextEdges = new Map();
    }
    get isContextSensitive() {
        return !!this.contextSymbols;
    }
    isContextSymbol(symbol) {
        if (!this.isContextSensitive) {
            return false;
        }
        return this.contextSymbols.get(symbol);
    }
    setContextSymbol(symbol) {
        assert(this.isContextSensitive);
        this.contextSymbols.set(symbol);
    }
    setContextSensitive(atn) {
        assert(!this.configs.isOutermostConfigSet);
        if (this.isContextSensitive) {
            return;
        }
        if (!this.contextSymbols) {
            this.contextSymbols = new BitSet_1.BitSet();
        }
    }
    get acceptStateInfo() {
        return this._acceptStateInfo;
    }
    set acceptStateInfo(acceptStateInfo) {
        this._acceptStateInfo = acceptStateInfo;
    }
    get isAcceptState() {
        return !!this._acceptStateInfo;
    }
    get prediction() {
        if (!this._acceptStateInfo) {
            return ATN_1.ATN.INVALID_ALT_NUMBER;
        }
        return this._acceptStateInfo.prediction;
    }
    get lexerActionExecutor() {
        if (!this._acceptStateInfo) {
            return undefined;
        }
        return this._acceptStateInfo.lexerActionExecutor;
    }
    getTarget(symbol) {
        return this.edges.get(symbol);
    }
    setTarget(symbol, target) {
        this.edges.set(symbol, target);
    }
    getEdgeMap() {
        return this.edges;
    }
    getContextTarget(invokingState) {
        if (invokingState === PredictionContext_1.PredictionContext.EMPTY_FULL_STATE_KEY) {
            invokingState = -1;
        }
        return this.contextEdges.get(invokingState);
    }
    setContextTarget(invokingState, target) {
        if (!this.isContextSensitive) {
            throw new Error("The state is not context sensitive.");
        }
        if (invokingState === PredictionContext_1.PredictionContext.EMPTY_FULL_STATE_KEY) {
            invokingState = -1;
        }
        this.contextEdges.set(invokingState, target);
    }
    getContextEdgeMap() {
        let map = new Map(this.contextEdges);
        let existing = map.get(-1);
        if (existing !== undefined) {
            if (map.size === 1) {
                let result = new Map();
                result.set(PredictionContext_1.PredictionContext.EMPTY_FULL_STATE_KEY, existing);
                return result;
            }
            else {
                map.delete(-1);
                map.set(PredictionContext_1.PredictionContext.EMPTY_FULL_STATE_KEY, existing);
            }
        }
        return map;
    }
    hashCode() {
        let hash = MurmurHash_1.MurmurHash.initialize(7);
        hash = MurmurHash_1.MurmurHash.update(hash, this.configs.hashCode());
        hash = MurmurHash_1.MurmurHash.finish(hash, 1);
        return hash;
    }
    /**
     * Two {@link DFAState} instances are equal if their ATN configuration sets
     * are the same. This method is used to see if a state already exists.
     *
     * Because the number of alternatives and number of ATN configurations are
     * finite, there is a finite number of DFA states that can be processed.
     * This is necessary to show that the algorithm terminates.
     *
     * Cannot test the DFA state numbers here because in
     * {@link ParserATNSimulator#addDFAState} we need to know if any other state
     * exists that has this exact set of ATN configurations. The
     * {@link #stateNumber} is irrelevant.
     */
    equals(o) {
        // compare set of ATN configurations in this set with other
        if (this === o) {
            return true;
        }
        if (!(o instanceof DFAState)) {
            return false;
        }
        let other = o;
        let sameSet = this.configs.equals(other.configs);
        //		System.out.println("DFAState.equals: "+configs+(sameSet?"==":"!=")+other.configs);
        return sameSet;
    }
    toString() {
        let buf = "";
        buf += (this.stateNumber) + (":") + (this.configs);
        if (this.isAcceptState) {
            buf += ("=>");
            if (this.predicates) {
                buf += this.predicates;
            }
            else {
                buf += (this.prediction);
            }
        }
        return buf.toString();
    }
}
__decorate([
    Decorators_1.NotNull
], DFAState.prototype, "configs", void 0);
__decorate([
    Decorators_1.NotNull
], DFAState.prototype, "edges", void 0);
__decorate([
    Decorators_1.NotNull
], DFAState.prototype, "contextEdges", void 0);
__decorate([
    Decorators_1.Override
], DFAState.prototype, "hashCode", null);
__decorate([
    Decorators_1.Override
], DFAState.prototype, "equals", null);
__decorate([
    Decorators_1.Override
], DFAState.prototype, "toString", null);
exports.DFAState = DFAState;
(function (DFAState) {
    /** Map a predicate to a predicted alternative. */
    let PredPrediction = class PredPrediction {
        constructor(pred, alt) {
            this.alt = alt;
            this.pred = pred;
        }
        toString() {
            return "(" + this.pred + ", " + this.alt + ")";
        }
    };
    __decorate([
        Decorators_1.NotNull
    ], PredPrediction.prototype, "pred", void 0);
    __decorate([
        Decorators_1.Override
    ], PredPrediction.prototype, "toString", null);
    PredPrediction = __decorate([
        __param(0, Decorators_1.NotNull)
    ], PredPrediction);
    DFAState.PredPrediction = PredPrediction;
})(DFAState = exports.DFAState || (exports.DFAState = {}));
//# sourceMappingURL=DFAState.js.map