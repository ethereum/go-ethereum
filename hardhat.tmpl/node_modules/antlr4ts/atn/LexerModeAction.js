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
exports.LexerModeAction = void 0;
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
/**
 * Implements the `mode` lexer action by calling {@link Lexer#mode} with
 * the assigned mode.
 *
 * @author Sam Harwell
 * @since 4.2
 */
class LexerModeAction {
    /**
     * Constructs a new `mode` action with the specified mode value.
     * @param mode The mode value to pass to {@link Lexer#mode}.
     */
    constructor(mode) {
        this._mode = mode;
    }
    /**
     * Get the lexer mode this action should transition the lexer to.
     *
     * @returns The lexer mode for this `mode` command.
     */
    get mode() {
        return this._mode;
    }
    /**
     * {@inheritDoc}
     * @returns This method returns {@link LexerActionType#MODE}.
     */
    get actionType() {
        return 2 /* MODE */;
    }
    /**
     * {@inheritDoc}
     * @returns This method returns `false`.
     */
    get isPositionDependent() {
        return false;
    }
    /**
     * {@inheritDoc}
     *
     * This action is implemented by calling {@link Lexer#mode} with the
     * value provided by {@link #getMode}.
     */
    execute(lexer) {
        lexer.mode(this._mode);
    }
    hashCode() {
        let hash = MurmurHash_1.MurmurHash.initialize();
        hash = MurmurHash_1.MurmurHash.update(hash, this.actionType);
        hash = MurmurHash_1.MurmurHash.update(hash, this._mode);
        return MurmurHash_1.MurmurHash.finish(hash, 2);
    }
    equals(obj) {
        if (obj === this) {
            return true;
        }
        else if (!(obj instanceof LexerModeAction)) {
            return false;
        }
        return this._mode === obj._mode;
    }
    toString() {
        return `mode(${this._mode})`;
    }
}
__decorate([
    Decorators_1.Override
], LexerModeAction.prototype, "actionType", null);
__decorate([
    Decorators_1.Override
], LexerModeAction.prototype, "isPositionDependent", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull)
], LexerModeAction.prototype, "execute", null);
__decorate([
    Decorators_1.Override
], LexerModeAction.prototype, "hashCode", null);
__decorate([
    Decorators_1.Override
], LexerModeAction.prototype, "equals", null);
__decorate([
    Decorators_1.Override
], LexerModeAction.prototype, "toString", null);
exports.LexerModeAction = LexerModeAction;
//# sourceMappingURL=LexerModeAction.js.map