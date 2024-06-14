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
exports.LexerCustomAction = void 0;
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
/**
 * Executes a custom lexer action by calling {@link Recognizer#action} with the
 * rule and action indexes assigned to the custom action. The implementation of
 * a custom action is added to the generated code for the lexer in an override
 * of {@link Recognizer#action} when the grammar is compiled.
 *
 * This class may represent embedded actions created with the `{...}`
 * syntax in ANTLR 4, as well as actions created for lexer commands where the
 * command argument could not be evaluated when the grammar was compiled.
 *
 * @author Sam Harwell
 * @since 4.2
 */
class LexerCustomAction {
    /**
     * Constructs a custom lexer action with the specified rule and action
     * indexes.
     *
     * @param ruleIndex The rule index to use for calls to
     * {@link Recognizer#action}.
     * @param actionIndex The action index to use for calls to
     * {@link Recognizer#action}.
     */
    constructor(ruleIndex, actionIndex) {
        this._ruleIndex = ruleIndex;
        this._actionIndex = actionIndex;
    }
    /**
     * Gets the rule index to use for calls to {@link Recognizer#action}.
     *
     * @returns The rule index for the custom action.
     */
    get ruleIndex() {
        return this._ruleIndex;
    }
    /**
     * Gets the action index to use for calls to {@link Recognizer#action}.
     *
     * @returns The action index for the custom action.
     */
    get actionIndex() {
        return this._actionIndex;
    }
    /**
     * {@inheritDoc}
     *
     * @returns This method returns {@link LexerActionType#CUSTOM}.
     */
    get actionType() {
        return 1 /* CUSTOM */;
    }
    /**
     * Gets whether the lexer action is position-dependent. Position-dependent
     * actions may have different semantics depending on the {@link CharStream}
     * index at the time the action is executed.
     *
     * Custom actions are position-dependent since they may represent a
     * user-defined embedded action which makes calls to methods like
     * {@link Lexer#getText}.
     *
     * @returns This method returns `true`.
     */
    get isPositionDependent() {
        return true;
    }
    /**
     * {@inheritDoc}
     *
     * Custom actions are implemented by calling {@link Lexer#action} with the
     * appropriate rule and action indexes.
     */
    execute(lexer) {
        lexer.action(undefined, this._ruleIndex, this._actionIndex);
    }
    hashCode() {
        let hash = MurmurHash_1.MurmurHash.initialize();
        hash = MurmurHash_1.MurmurHash.update(hash, this.actionType);
        hash = MurmurHash_1.MurmurHash.update(hash, this._ruleIndex);
        hash = MurmurHash_1.MurmurHash.update(hash, this._actionIndex);
        return MurmurHash_1.MurmurHash.finish(hash, 3);
    }
    equals(obj) {
        if (obj === this) {
            return true;
        }
        else if (!(obj instanceof LexerCustomAction)) {
            return false;
        }
        return this._ruleIndex === obj._ruleIndex
            && this._actionIndex === obj._actionIndex;
    }
}
__decorate([
    Decorators_1.Override
], LexerCustomAction.prototype, "actionType", null);
__decorate([
    Decorators_1.Override
], LexerCustomAction.prototype, "isPositionDependent", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull)
], LexerCustomAction.prototype, "execute", null);
__decorate([
    Decorators_1.Override
], LexerCustomAction.prototype, "hashCode", null);
__decorate([
    Decorators_1.Override
], LexerCustomAction.prototype, "equals", null);
exports.LexerCustomAction = LexerCustomAction;
//# sourceMappingURL=LexerCustomAction.js.map