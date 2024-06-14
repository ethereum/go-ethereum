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
exports.ActionTransition = void 0;
const Decorators_1 = require("../Decorators");
const Transition_1 = require("./Transition");
let ActionTransition = class ActionTransition extends Transition_1.Transition {
    constructor(target, ruleIndex, actionIndex = -1, isCtxDependent = false) {
        super(target);
        this.ruleIndex = ruleIndex;
        this.actionIndex = actionIndex;
        this.isCtxDependent = isCtxDependent;
    }
    get serializationType() {
        return 6 /* ACTION */;
    }
    get isEpsilon() {
        return true; // we are to be ignored by analysis 'cept for predicates
    }
    matches(symbol, minVocabSymbol, maxVocabSymbol) {
        return false;
    }
    toString() {
        return "action_" + this.ruleIndex + ":" + this.actionIndex;
    }
};
__decorate([
    Decorators_1.Override
], ActionTransition.prototype, "serializationType", null);
__decorate([
    Decorators_1.Override
], ActionTransition.prototype, "isEpsilon", null);
__decorate([
    Decorators_1.Override
], ActionTransition.prototype, "matches", null);
__decorate([
    Decorators_1.Override
], ActionTransition.prototype, "toString", null);
ActionTransition = __decorate([
    __param(0, Decorators_1.NotNull)
], ActionTransition);
exports.ActionTransition = ActionTransition;
//# sourceMappingURL=ActionTransition.js.map