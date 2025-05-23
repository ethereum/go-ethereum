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
exports.PrecedencePredicateTransition = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:35.0994191-07:00
const AbstractPredicateTransition_1 = require("./AbstractPredicateTransition");
const Decorators_1 = require("../Decorators");
const SemanticContext_1 = require("./SemanticContext");
/**
 *
 * @author Sam Harwell
 */
let PrecedencePredicateTransition = class PrecedencePredicateTransition extends AbstractPredicateTransition_1.AbstractPredicateTransition {
    constructor(target, precedence) {
        super(target);
        this.precedence = precedence;
    }
    get serializationType() {
        return 10 /* PRECEDENCE */;
    }
    get isEpsilon() {
        return true;
    }
    matches(symbol, minVocabSymbol, maxVocabSymbol) {
        return false;
    }
    get predicate() {
        return new SemanticContext_1.SemanticContext.PrecedencePredicate(this.precedence);
    }
    toString() {
        return this.precedence + " >= _p";
    }
};
__decorate([
    Decorators_1.Override
], PrecedencePredicateTransition.prototype, "serializationType", null);
__decorate([
    Decorators_1.Override
], PrecedencePredicateTransition.prototype, "isEpsilon", null);
__decorate([
    Decorators_1.Override
], PrecedencePredicateTransition.prototype, "matches", null);
__decorate([
    Decorators_1.Override
], PrecedencePredicateTransition.prototype, "toString", null);
PrecedencePredicateTransition = __decorate([
    __param(0, Decorators_1.NotNull)
], PrecedencePredicateTransition);
exports.PrecedencePredicateTransition = PrecedencePredicateTransition;
//# sourceMappingURL=PrecedencePredicateTransition.js.map