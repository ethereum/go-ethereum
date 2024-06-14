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
exports.ConflictInfo = void 0;
const Decorators_1 = require("../Decorators");
const Utils = require("../misc/Utils");
/**
 * This class stores information about a configuration conflict.
 *
 * @author Sam Harwell
 */
class ConflictInfo {
    constructor(conflictedAlts, exact) {
        this._conflictedAlts = conflictedAlts;
        this.exact = exact;
    }
    /**
     * Gets the set of conflicting alternatives for the configuration set.
     */
    get conflictedAlts() {
        return this._conflictedAlts;
    }
    /**
     * Gets whether or not the configuration conflict is an exact conflict.
     * An exact conflict occurs when the prediction algorithm determines that
     * the represented alternatives for a particular configuration set cannot be
     * further reduced by consuming additional input. After reaching an exact
     * conflict during an SLL prediction, only switch to full-context prediction
     * could reduce the set of viable alternatives. In LL prediction, an exact
     * conflict indicates a true ambiguity in the input.
     *
     * For the {@link PredictionMode#LL_EXACT_AMBIG_DETECTION} prediction mode,
     * accept states are conflicting but not exact are treated as non-accept
     * states.
     */
    get isExact() {
        return this.exact;
    }
    equals(obj) {
        if (obj === this) {
            return true;
        }
        else if (!(obj instanceof ConflictInfo)) {
            return false;
        }
        return this.isExact === obj.isExact
            && Utils.equals(this.conflictedAlts, obj.conflictedAlts);
    }
    hashCode() {
        return this.conflictedAlts.hashCode();
    }
}
__decorate([
    Decorators_1.Override
], ConflictInfo.prototype, "equals", null);
__decorate([
    Decorators_1.Override
], ConflictInfo.prototype, "hashCode", null);
exports.ConflictInfo = ConflictInfo;
//# sourceMappingURL=ConflictInfo.js.map