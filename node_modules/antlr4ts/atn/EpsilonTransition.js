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
exports.EpsilonTransition = void 0;
const Decorators_1 = require("../Decorators");
const Transition_1 = require("./Transition");
let EpsilonTransition = class EpsilonTransition extends Transition_1.Transition {
    constructor(target, outermostPrecedenceReturn = -1) {
        super(target);
        this._outermostPrecedenceReturn = outermostPrecedenceReturn;
    }
    /**
     * @returns the rule index of a precedence rule for which this transition is
     * returning from, where the precedence value is 0; otherwise, -1.
     *
     * @see ATNConfig.isPrecedenceFilterSuppressed
     * @see ParserATNSimulator#applyPrecedenceFilter(ATNConfigSet, ParserRuleContext, PredictionContextCache)
     * @since 4.4.1
     */
    get outermostPrecedenceReturn() {
        return this._outermostPrecedenceReturn;
    }
    get serializationType() {
        return 1 /* EPSILON */;
    }
    get isEpsilon() {
        return true;
    }
    matches(symbol, minVocabSymbol, maxVocabSymbol) {
        return false;
    }
    toString() {
        return "epsilon";
    }
};
__decorate([
    Decorators_1.Override
], EpsilonTransition.prototype, "serializationType", null);
__decorate([
    Decorators_1.Override
], EpsilonTransition.prototype, "isEpsilon", null);
__decorate([
    Decorators_1.Override
], EpsilonTransition.prototype, "matches", null);
__decorate([
    Decorators_1.Override,
    Decorators_1.NotNull
], EpsilonTransition.prototype, "toString", null);
EpsilonTransition = __decorate([
    __param(0, Decorators_1.NotNull)
], EpsilonTransition);
exports.EpsilonTransition = EpsilonTransition;
//# sourceMappingURL=EpsilonTransition.js.map