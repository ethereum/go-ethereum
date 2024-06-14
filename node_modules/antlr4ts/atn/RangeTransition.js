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
exports.RangeTransition = void 0;
const IntervalSet_1 = require("../misc/IntervalSet");
const Decorators_1 = require("../Decorators");
const Transition_1 = require("./Transition");
let RangeTransition = class RangeTransition extends Transition_1.Transition {
    constructor(target, from, to) {
        super(target);
        this.from = from;
        this.to = to;
    }
    get serializationType() {
        return 2 /* RANGE */;
    }
    get label() {
        return IntervalSet_1.IntervalSet.of(this.from, this.to);
    }
    matches(symbol, minVocabSymbol, maxVocabSymbol) {
        return symbol >= this.from && symbol <= this.to;
    }
    toString() {
        return "'" + String.fromCodePoint(this.from) + "'..'" + String.fromCodePoint(this.to) + "'";
    }
};
__decorate([
    Decorators_1.Override
], RangeTransition.prototype, "serializationType", null);
__decorate([
    Decorators_1.Override,
    Decorators_1.NotNull
], RangeTransition.prototype, "label", null);
__decorate([
    Decorators_1.Override
], RangeTransition.prototype, "matches", null);
__decorate([
    Decorators_1.Override,
    Decorators_1.NotNull
], RangeTransition.prototype, "toString", null);
RangeTransition = __decorate([
    __param(0, Decorators_1.NotNull)
], RangeTransition);
exports.RangeTransition = RangeTransition;
//# sourceMappingURL=RangeTransition.js.map