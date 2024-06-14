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
exports.AtomTransition = void 0;
const IntervalSet_1 = require("../misc/IntervalSet");
const Decorators_1 = require("../Decorators");
const Transition_1 = require("./Transition");
/** TODO: make all transitions sets? no, should remove set edges */
let AtomTransition = class AtomTransition extends Transition_1.Transition {
    constructor(target, label) {
        super(target);
        this._label = label;
    }
    get serializationType() {
        return 5 /* ATOM */;
    }
    get label() {
        return IntervalSet_1.IntervalSet.of(this._label);
    }
    matches(symbol, minVocabSymbol, maxVocabSymbol) {
        return this._label === symbol;
    }
    toString() {
        return String(this.label);
    }
};
__decorate([
    Decorators_1.Override
], AtomTransition.prototype, "serializationType", null);
__decorate([
    Decorators_1.Override,
    Decorators_1.NotNull
], AtomTransition.prototype, "label", null);
__decorate([
    Decorators_1.Override
], AtomTransition.prototype, "matches", null);
__decorate([
    Decorators_1.Override,
    Decorators_1.NotNull
], AtomTransition.prototype, "toString", null);
AtomTransition = __decorate([
    __param(0, Decorators_1.NotNull)
], AtomTransition);
exports.AtomTransition = AtomTransition;
//# sourceMappingURL=AtomTransition.js.map