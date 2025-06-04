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
exports.ArrayEqualityComparator = void 0;
const Decorators_1 = require("../Decorators");
const MurmurHash_1 = require("./MurmurHash");
const ObjectEqualityComparator_1 = require("./ObjectEqualityComparator");
/**
 * This default implementation of {@link EqualityComparator} uses object equality
 * for comparisons by calling {@link Object#hashCode} and {@link Object#equals}.
 *
 * @author Sam Harwell
 */
class ArrayEqualityComparator {
    /**
     * {@inheritDoc}
     *
     * This implementation returns
     * `obj.`{@link Object#hashCode hashCode()}.
     */
    hashCode(obj) {
        if (obj == null) {
            return 0;
        }
        return MurmurHash_1.MurmurHash.hashCode(obj, 0);
    }
    /**
     * {@inheritDoc}
     *
     * This implementation relies on object equality. If both objects are
     * `undefined`, this method returns `true`. Otherwise if only
     * `a` is `undefined`, this method returns `false`. Otherwise,
     * this method returns the result of
     * `a.`{@link Object#equals equals}`(b)`.
     */
    equals(a, b) {
        if (a == null) {
            return b == null;
        }
        else if (b == null) {
            return false;
        }
        if (a.length !== b.length) {
            return false;
        }
        for (let i = 0; i < a.length; i++) {
            if (!ObjectEqualityComparator_1.ObjectEqualityComparator.INSTANCE.equals(a[i], b[i])) {
                return false;
            }
        }
        return true;
    }
}
ArrayEqualityComparator.INSTANCE = new ArrayEqualityComparator();
__decorate([
    Decorators_1.Override
], ArrayEqualityComparator.prototype, "hashCode", null);
__decorate([
    Decorators_1.Override
], ArrayEqualityComparator.prototype, "equals", null);
exports.ArrayEqualityComparator = ArrayEqualityComparator;
//# sourceMappingURL=ArrayEqualityComparator.js.map