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
exports.TokenTagToken = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:46.3281988-07:00
const CommonToken_1 = require("../../CommonToken");
const Decorators_1 = require("../../Decorators");
/**
 * A {@link Token} object representing a token of a particular type; e.g.,
 * `<ID>`. These tokens are created for {@link TagChunk} chunks where the
 * tag corresponds to a lexer rule or token type.
 */
let TokenTagToken = class TokenTagToken extends CommonToken_1.CommonToken {
    /**
     * Constructs a new instance of {@link TokenTagToken} with the specified
     * token name, type, and label.
     *
     * @param tokenName The token name.
     * @param type The token type.
     * @param label The label associated with the token tag, or `undefined` if
     * the token tag is unlabeled.
     */
    constructor(tokenName, type, label) {
        super(type);
        this._tokenName = tokenName;
        this._label = label;
    }
    /**
     * Gets the token name.
     * @returns The token name.
     */
    get tokenName() {
        return this._tokenName;
    }
    /**
     * Gets the label associated with the rule tag.
     *
     * @returns The name of the label associated with the rule tag, or
     * `undefined` if this is an unlabeled rule tag.
     */
    get label() {
        return this._label;
    }
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link TokenTagToken} returns the token tag
     * formatted with `<` and `>` delimiters.
     */
    get text() {
        if (this._label != null) {
            return "<" + this._label + ":" + this._tokenName + ">";
        }
        return "<" + this._tokenName + ">";
    }
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link TokenTagToken} returns a string of the form
     * `tokenName:type`.
     */
    toString() {
        return this._tokenName + ":" + this.type;
    }
};
__decorate([
    Decorators_1.NotNull
], TokenTagToken.prototype, "_tokenName", void 0);
__decorate([
    Decorators_1.NotNull
], TokenTagToken.prototype, "tokenName", null);
__decorate([
    Decorators_1.Override
], TokenTagToken.prototype, "text", null);
__decorate([
    Decorators_1.Override
], TokenTagToken.prototype, "toString", null);
TokenTagToken = __decorate([
    __param(0, Decorators_1.NotNull)
], TokenTagToken);
exports.TokenTagToken = TokenTagToken;
//# sourceMappingURL=TokenTagToken.js.map