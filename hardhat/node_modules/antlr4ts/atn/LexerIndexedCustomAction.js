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
exports.LexerIndexedCustomAction = void 0;
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
/**
 * This implementation of {@link LexerAction} is used for tracking input offsets
 * for position-dependent actions within a {@link LexerActionExecutor}.
 *
 * This action is not serialized as part of the ATN, and is only required for
 * position-dependent lexer actions which appear at a location other than the
 * end of a rule. For more information about DFA optimizations employed for
 * lexer actions, see {@link LexerActionExecutor#append} and
 * {@link LexerActionExecutor#fixOffsetBeforeMatch}.
 *
 * @author Sam Harwell
 * @since 4.2
 */
let LexerIndexedCustomAction = class LexerIndexedCustomAction {
    /**
     * Constructs a new indexed custom action by associating a character offset
     * with a {@link LexerAction}.
     *
     * Note: This class is only required for lexer actions for which
     * {@link LexerAction#isPositionDependent} returns `true`.
     *
     * @param offset The offset into the input {@link CharStream}, relative to
     * the token start index, at which the specified lexer action should be
     * executed.
     * @param action The lexer action to execute at a particular offset in the
     * input {@link CharStream}.
     */
    constructor(offset, action) {
        this._offset = offset;
        this._action = action;
    }
    /**
     * Gets the location in the input {@link CharStream} at which the lexer
     * action should be executed. The value is interpreted as an offset relative
     * to the token start index.
     *
     * @returns The location in the input {@link CharStream} at which the lexer
     * action should be executed.
     */
    get offset() {
        return this._offset;
    }
    /**
     * Gets the lexer action to execute.
     *
     * @returns A {@link LexerAction} object which executes the lexer action.
     */
    get action() {
        return this._action;
    }
    /**
     * {@inheritDoc}
     *
     * @returns This method returns the result of calling {@link #getActionType}
     * on the {@link LexerAction} returned by {@link #getAction}.
     */
    get actionType() {
        return this._action.actionType;
    }
    /**
     * {@inheritDoc}
     * @returns This method returns `true`.
     */
    get isPositionDependent() {
        return true;
    }
    /**
     * {@inheritDoc}
     *
     * This method calls {@link #execute} on the result of {@link #getAction}
     * using the provided `lexer`.
     */
    execute(lexer) {
        // assume the input stream position was properly set by the calling code
        this._action.execute(lexer);
    }
    hashCode() {
        let hash = MurmurHash_1.MurmurHash.initialize();
        hash = MurmurHash_1.MurmurHash.update(hash, this._offset);
        hash = MurmurHash_1.MurmurHash.update(hash, this._action);
        return MurmurHash_1.MurmurHash.finish(hash, 2);
    }
    equals(obj) {
        if (obj === this) {
            return true;
        }
        else if (!(obj instanceof LexerIndexedCustomAction)) {
            return false;
        }
        return this._offset === obj._offset
            && this._action.equals(obj._action);
    }
};
__decorate([
    Decorators_1.NotNull
], LexerIndexedCustomAction.prototype, "action", null);
__decorate([
    Decorators_1.Override
], LexerIndexedCustomAction.prototype, "actionType", null);
__decorate([
    Decorators_1.Override
], LexerIndexedCustomAction.prototype, "isPositionDependent", null);
__decorate([
    Decorators_1.Override
], LexerIndexedCustomAction.prototype, "execute", null);
__decorate([
    Decorators_1.Override
], LexerIndexedCustomAction.prototype, "hashCode", null);
__decorate([
    Decorators_1.Override
], LexerIndexedCustomAction.prototype, "equals", null);
LexerIndexedCustomAction = __decorate([
    __param(1, Decorators_1.NotNull)
], LexerIndexedCustomAction);
exports.LexerIndexedCustomAction = LexerIndexedCustomAction;
//# sourceMappingURL=LexerIndexedCustomAction.js.map