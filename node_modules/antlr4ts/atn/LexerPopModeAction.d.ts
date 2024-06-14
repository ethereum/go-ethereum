/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Lexer } from "../Lexer";
import { LexerAction } from "./LexerAction";
import { LexerActionType } from "./LexerActionType";
/**
 * Implements the `popMode` lexer action by calling {@link Lexer#popMode}.
 *
 * The `popMode` command does not have any parameters, so this action is
 * implemented as a singleton instance exposed by {@link #INSTANCE}.
 *
 * @author Sam Harwell
 * @since 4.2
 */
export declare class LexerPopModeAction implements LexerAction {
    /**
     * Constructs the singleton instance of the lexer `popMode` command.
     */
    constructor();
    /**
     * {@inheritDoc}
     * @returns This method returns {@link LexerActionType#POP_MODE}.
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
     * This action is implemented by calling {@link Lexer#popMode}.
     */
    execute(lexer: Lexer): void;
    hashCode(): number;
    equals(obj: any): boolean;
    toString(): string;
}
export declare namespace LexerPopModeAction {
    /**
     * Provides a singleton instance of this parameterless lexer action.
     */
    const INSTANCE: LexerPopModeAction;
}
