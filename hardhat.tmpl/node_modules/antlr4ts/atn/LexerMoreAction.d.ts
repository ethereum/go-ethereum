/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Lexer } from "../Lexer";
import { LexerAction } from "./LexerAction";
import { LexerActionType } from "./LexerActionType";
/**
 * Implements the `more` lexer action by calling {@link Lexer#more}.
 *
 * The `more` command does not have any parameters, so this action is
 * implemented as a singleton instance exposed by {@link #INSTANCE}.
 *
 * @author Sam Harwell
 * @since 4.2
 */
export declare class LexerMoreAction implements LexerAction {
    /**
     * Constructs the singleton instance of the lexer `more` command.
     */
    constructor();
    /**
     * {@inheritDoc}
     * @returns This method returns {@link LexerActionType#MORE}.
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
     * This action is implemented by calling {@link Lexer#more}.
     */
    execute(lexer: Lexer): void;
    hashCode(): number;
    equals(obj: any): boolean;
    toString(): string;
}
export declare namespace LexerMoreAction {
    /**
     * Provides a singleton instance of this parameterless lexer action.
     */
    const INSTANCE: LexerMoreAction;
}
