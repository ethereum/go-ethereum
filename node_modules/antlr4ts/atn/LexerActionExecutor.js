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
exports.LexerActionExecutor = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:28.8810453-07:00
const ArrayEqualityComparator_1 = require("../misc/ArrayEqualityComparator");
const LexerIndexedCustomAction_1 = require("./LexerIndexedCustomAction");
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
/**
 * Represents an executor for a sequence of lexer actions which traversed during
 * the matching operation of a lexer rule (token).
 *
 * The executor tracks position information for position-dependent lexer actions
 * efficiently, ensuring that actions appearing only at the end of the rule do
 * not cause bloating of the {@link DFA} created for the lexer.
 *
 * @author Sam Harwell
 * @since 4.2
 */
let LexerActionExecutor = class LexerActionExecutor {
    /**
     * Constructs an executor for a sequence of {@link LexerAction} actions.
     * @param lexerActions The lexer actions to execute.
     */
    constructor(lexerActions) {
        this._lexerActions = lexerActions;
        let hash = MurmurHash_1.MurmurHash.initialize();
        for (let lexerAction of lexerActions) {
            hash = MurmurHash_1.MurmurHash.update(hash, lexerAction);
        }
        this.cachedHashCode = MurmurHash_1.MurmurHash.finish(hash, lexerActions.length);
    }
    /**
     * Creates a {@link LexerActionExecutor} which executes the actions for
     * the input `lexerActionExecutor` followed by a specified
     * `lexerAction`.
     *
     * @param lexerActionExecutor The executor for actions already traversed by
     * the lexer while matching a token within a particular
     * {@link ATNConfig}. If this is `undefined`, the method behaves as though
     * it were an empty executor.
     * @param lexerAction The lexer action to execute after the actions
     * specified in `lexerActionExecutor`.
     *
     * @returns A {@link LexerActionExecutor} for executing the combine actions
     * of `lexerActionExecutor` and `lexerAction`.
     */
    static append(lexerActionExecutor, lexerAction) {
        if (!lexerActionExecutor) {
            return new LexerActionExecutor([lexerAction]);
        }
        let lexerActions = lexerActionExecutor._lexerActions.slice(0);
        lexerActions.push(lexerAction);
        return new LexerActionExecutor(lexerActions);
    }
    /**
     * Creates a {@link LexerActionExecutor} which encodes the current offset
     * for position-dependent lexer actions.
     *
     * Normally, when the executor encounters lexer actions where
     * {@link LexerAction#isPositionDependent} returns `true`, it calls
     * {@link IntStream#seek} on the input {@link CharStream} to set the input
     * position to the *end* of the current token. This behavior provides
     * for efficient DFA representation of lexer actions which appear at the end
     * of a lexer rule, even when the lexer rule matches a variable number of
     * characters.
     *
     * Prior to traversing a match transition in the ATN, the current offset
     * from the token start index is assigned to all position-dependent lexer
     * actions which have not already been assigned a fixed offset. By storing
     * the offsets relative to the token start index, the DFA representation of
     * lexer actions which appear in the middle of tokens remains efficient due
     * to sharing among tokens of the same length, regardless of their absolute
     * position in the input stream.
     *
     * If the current executor already has offsets assigned to all
     * position-dependent lexer actions, the method returns `this`.
     *
     * @param offset The current offset to assign to all position-dependent
     * lexer actions which do not already have offsets assigned.
     *
     * @returns A {@link LexerActionExecutor} which stores input stream offsets
     * for all position-dependent lexer actions.
     */
    fixOffsetBeforeMatch(offset) {
        let updatedLexerActions;
        for (let i = 0; i < this._lexerActions.length; i++) {
            if (this._lexerActions[i].isPositionDependent && !(this._lexerActions[i] instanceof LexerIndexedCustomAction_1.LexerIndexedCustomAction)) {
                if (!updatedLexerActions) {
                    updatedLexerActions = this._lexerActions.slice(0);
                }
                updatedLexerActions[i] = new LexerIndexedCustomAction_1.LexerIndexedCustomAction(offset, this._lexerActions[i]);
            }
        }
        if (!updatedLexerActions) {
            return this;
        }
        return new LexerActionExecutor(updatedLexerActions);
    }
    /**
     * Gets the lexer actions to be executed by this executor.
     * @returns The lexer actions to be executed by this executor.
     */
    get lexerActions() {
        return this._lexerActions;
    }
    /**
     * Execute the actions encapsulated by this executor within the context of a
     * particular {@link Lexer}.
     *
     * This method calls {@link IntStream#seek} to set the position of the
     * `input` {@link CharStream} prior to calling
     * {@link LexerAction#execute} on a position-dependent action. Before the
     * method returns, the input position will be restored to the same position
     * it was in when the method was invoked.
     *
     * @param lexer The lexer instance.
     * @param input The input stream which is the source for the current token.
     * When this method is called, the current {@link IntStream#index} for
     * `input` should be the start of the following token, i.e. 1
     * character past the end of the current token.
     * @param startIndex The token start index. This value may be passed to
     * {@link IntStream#seek} to set the `input` position to the beginning
     * of the token.
     */
    execute(lexer, input, startIndex) {
        let requiresSeek = false;
        let stopIndex = input.index;
        try {
            for (let lexerAction of this._lexerActions) {
                if (lexerAction instanceof LexerIndexedCustomAction_1.LexerIndexedCustomAction) {
                    let offset = lexerAction.offset;
                    input.seek(startIndex + offset);
                    lexerAction = lexerAction.action;
                    requiresSeek = (startIndex + offset) !== stopIndex;
                }
                else if (lexerAction.isPositionDependent) {
                    input.seek(stopIndex);
                    requiresSeek = false;
                }
                lexerAction.execute(lexer);
            }
        }
        finally {
            if (requiresSeek) {
                input.seek(stopIndex);
            }
        }
    }
    hashCode() {
        return this.cachedHashCode;
    }
    equals(obj) {
        if (obj === this) {
            return true;
        }
        else if (!(obj instanceof LexerActionExecutor)) {
            return false;
        }
        return this.cachedHashCode === obj.cachedHashCode
            && ArrayEqualityComparator_1.ArrayEqualityComparator.INSTANCE.equals(this._lexerActions, obj._lexerActions);
    }
};
__decorate([
    Decorators_1.NotNull
], LexerActionExecutor.prototype, "_lexerActions", void 0);
__decorate([
    Decorators_1.NotNull
], LexerActionExecutor.prototype, "lexerActions", null);
__decorate([
    __param(0, Decorators_1.NotNull)
], LexerActionExecutor.prototype, "execute", null);
__decorate([
    Decorators_1.Override
], LexerActionExecutor.prototype, "hashCode", null);
__decorate([
    Decorators_1.Override
], LexerActionExecutor.prototype, "equals", null);
__decorate([
    Decorators_1.NotNull,
    __param(1, Decorators_1.NotNull)
], LexerActionExecutor, "append", null);
LexerActionExecutor = __decorate([
    __param(0, Decorators_1.NotNull)
], LexerActionExecutor);
exports.LexerActionExecutor = LexerActionExecutor;
//# sourceMappingURL=LexerActionExecutor.js.map