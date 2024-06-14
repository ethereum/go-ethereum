/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Lexer } from "../Lexer";
import { LexerAction } from "./LexerAction";
import { LexerActionType } from "./LexerActionType";
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
export declare class LexerCustomAction implements LexerAction {
    private readonly _ruleIndex;
    private readonly _actionIndex;
    /**
     * Constructs a custom lexer action with the specified rule and action
     * indexes.
     *
     * @param ruleIndex The rule index to use for calls to
     * {@link Recognizer#action}.
     * @param actionIndex The action index to use for calls to
     * {@link Recognizer#action}.
     */
    constructor(ruleIndex: number, actionIndex: number);
    /**
     * Gets the rule index to use for calls to {@link Recognizer#action}.
     *
     * @returns The rule index for the custom action.
     */
    get ruleIndex(): number;
    /**
     * Gets the action index to use for calls to {@link Recognizer#action}.
     *
     * @returns The action index for the custom action.
     */
    get actionIndex(): number;
    /**
     * {@inheritDoc}
     *
     * @returns This method returns {@link LexerActionType#CUSTOM}.
     */
    get actionType(): LexerActionType;
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
    get isPositionDependent(): boolean;
    /**
     * {@inheritDoc}
     *
     * Custom actions are implemented by calling {@link Lexer#action} with the
     * appropriate rule and action indexes.
     */
    execute(lexer: Lexer): void;
    hashCode(): number;
    equals(obj: any): boolean;
}
