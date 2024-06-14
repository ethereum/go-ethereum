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
exports.OrderedATNConfigSet = void 0;
const ATNConfigSet_1 = require("./ATNConfigSet");
const Decorators_1 = require("../Decorators");
/**
 *
 * @author Sam Harwell
 */
class OrderedATNConfigSet extends ATNConfigSet_1.ATNConfigSet {
    constructor(set, readonly) {
        if (set != null && readonly != null) {
            super(set, readonly);
        }
        else {
            super();
        }
    }
    clone(readonly) {
        let copy = new OrderedATNConfigSet(this, readonly);
        if (!readonly && this.isReadOnly) {
            copy.addAll(this);
        }
        return copy;
    }
    getKey(e) {
        // This is a specially crafted key to ensure configurations are only merged if they are equal
        return { state: 0, alt: e.hashCode() };
    }
    canMerge(left, leftKey, right) {
        return left.equals(right);
    }
}
__decorate([
    Decorators_1.Override
], OrderedATNConfigSet.prototype, "clone", null);
__decorate([
    Decorators_1.Override
], OrderedATNConfigSet.prototype, "getKey", null);
__decorate([
    Decorators_1.Override
], OrderedATNConfigSet.prototype, "canMerge", null);
exports.OrderedATNConfigSet = OrderedATNConfigSet;
//# sourceMappingURL=OrderedATNConfigSet.js.map