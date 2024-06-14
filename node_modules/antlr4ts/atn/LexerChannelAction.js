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
exports.LexerChannelAction = void 0;
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
/**
 * Implements the `channel` lexer action by calling
 * {@link Lexer#setChannel} with the assigned channel.
 *
 * @author Sam Harwell
 * @since 4.2
 */
class LexerChannelAction {
    /**
     * Constructs a new `channel` action with the specified channel value.
     * @param channel The channel value to pass to {@link Lexer#setChannel}.
     */
    constructor(channel) {
        this._channel = channel;
    }
    /**
     * Gets the channel to use for the {@link Token} created by the lexer.
     *
     * @returns The channel to use for the {@link Token} created by the lexer.
     */
    get channel() {
        return this._channel;
    }
    /**
     * {@inheritDoc}
     * @returns This method returns {@link LexerActionType#CHANNEL}.
     */
    get actionType() {
        return 0 /* CHANNEL */;
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
     * This action is implemented by calling {@link Lexer#setChannel} with the
     * value provided by {@link #getChannel}.
     */
    execute(lexer) {
        lexer.channel = this._channel;
    }
    hashCode() {
        let hash = MurmurHash_1.MurmurHash.initialize();
        hash = MurmurHash_1.MurmurHash.update(hash, this.actionType);
        hash = MurmurHash_1.MurmurHash.update(hash, this._channel);
        return MurmurHash_1.MurmurHash.finish(hash, 2);
    }
    equals(obj) {
        if (obj === this) {
            return true;
        }
        else if (!(obj instanceof LexerChannelAction)) {
            return false;
        }
        return this._channel === obj._channel;
    }
    toString() {
        return `channel(${this._channel})`;
    }
}
__decorate([
    Decorators_1.Override
], LexerChannelAction.prototype, "actionType", null);
__decorate([
    Decorators_1.Override
], LexerChannelAction.prototype, "isPositionDependent", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull)
], LexerChannelAction.prototype, "execute", null);
__decorate([
    Decorators_1.Override
], LexerChannelAction.prototype, "hashCode", null);
__decorate([
    Decorators_1.Override
], LexerChannelAction.prototype, "equals", null);
__decorate([
    Decorators_1.Override
], LexerChannelAction.prototype, "toString", null);
exports.LexerChannelAction = LexerChannelAction;
//# sourceMappingURL=LexerChannelAction.js.map