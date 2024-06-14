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
exports.SetTransition = void 0;
const IntervalSet_1 = require("../misc/IntervalSet");
const Decorators_1 = require("../Decorators");
const Token_1 = require("../Token");
const Transition_1 = require("./Transition");
/** A transition containing a set of values. */
let SetTransition = class SetTransition extends Transition_1.Transition {
    // TODO (sam): should we really allow undefined here?
    constructor(target, set) {
        super(target);
        if (set == null) {
            set = IntervalSet_1.IntervalSet.of(Token_1.Token.INVALID_TYPE);
        }
        this.set = set;
    }
    get serializationType() {
        return 7 /* SET */;
    }
    get label() {
        return this.set;
    }
    matches(symbol, minVocabSymbol, maxVocabSymbol) {
        return this.set.contains(symbol);
    }
    toString() {
        return this.set.toString();
    }
};
__decorate([
    Decorators_1.NotNull
], SetTransition.prototype, "set", void 0);
__decorate([
    Decorators_1.Override
], SetTransition.prototype, "serializationType", null);
__decorate([
    Decorators_1.Override,
    Decorators_1.NotNull
], SetTransition.prototype, "label", null);
__decorate([
    Decorators_1.Override
], SetTransition.prototype, "matches", null);
__decorate([
    Decorators_1.Override,
    Decorators_1.NotNull
], SetTransition.prototype, "toString", null);
SetTransition = __decorate([
    __param(0, Decorators_1.NotNull), __param(1, Decorators_1.Nullable)
], SetTransition);
exports.SetTransition = SetTransition;
//# sourceMappingURL=SetTransition.js.map