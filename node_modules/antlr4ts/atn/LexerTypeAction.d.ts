/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Lexer } from "../Lexer";
import { LexerAction } from "./LexerAction";
import { LexerActionType } from "./LexerActionType";
/**
 * Implements the `type` lexer action by setting `Lexer.type`
 * with the assigned type.
 *
 * @author Sam Harwell
 * @since 4.2
 */
export declare class LexerTypeAction implements LexerAction {
    private readonly _type;
    /**
     * Constructs a new `type` action with the specified token type value.
     * @param type The type to assign to the token using `Lexer.type`.
     */
    constructor(type: number);
    /**
     * Gets the type to assign to a token created by the lexer.
     * @returns The type to assign to a token created by the lexer.
     */
    get type(): number;
    /**
     * {@inheritDoc}
     * @returns This method returns {@link LexerActionType#TYPE}.
     */
    get actionType(): LexerActionType;
    /**
     * {@inheritDoc}
     * @returns This method returns `false`.
     */
    get isPositionDependent(): boolean;
    /**
     * {@inheritDoc}
     *
     * This action is implemented by setting `Lexer.type` with the
     * value provided by `type`.
     */
    execute(lexer: Lexer): void;
    hashCode(): number;
    equals(obj: any): boolean;
    toString(): string;
}
