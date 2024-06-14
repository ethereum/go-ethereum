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
exports.RuleTransition = void 0;
const Decorators_1 = require("../Decorators");
const Transition_1 = require("./Transition");
/** */
let RuleTransition = class RuleTransition extends Transition_1.Transition {
    constructor(ruleStart, ruleIndex, precedence, followState) {
        super(ruleStart);
        this.tailCall = false;
        this.optimizedTailCall = false;
        this.ruleIndex = ruleIndex;
        this.precedence = precedence;
        this.followState = followState;
    }
    get serializationType() {
        return 3 /* RULE */;
    }
    get isEpsilon() {
        return true;
    }
    matches(symbol, minVocabSymbol, maxVocabSymbol) {
        return false;
    }
};
__decorate([
    Decorators_1.NotNull
], RuleTransition.prototype, "followState", void 0);
__decorate([
    Decorators_1.Override
], RuleTransition.prototype, "serializationType", null);
__decorate([
    Decorators_1.Override
], RuleTransition.prototype, "isEpsilon", null);
__decorate([
    Decorators_1.Override
], RuleTransition.prototype, "matches", null);
RuleTransition = __decorate([
    __param(0, Decorators_1.NotNull), __param(3, Decorators_1.NotNull)
], RuleTransition);
exports.RuleTransition = RuleTransition;
//# sourceMappingURL=RuleTransition.js.map