/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
/**
 *
 * @author Sam Harwell
 */
export declare class ATNDeserializationOptions {
    private static _defaultOptions?;
    private readOnly;
    private verifyATN;
    private generateRuleBypassTransitions;
    private optimize;
    constructor(options?: ATNDeserializationOptions);
    static get defaultOptions(): ATNDeserializationOptions;
    get isReadOnly(): boolean;
    makeReadOnly(): void;
    get isVerifyATN(): boolean;
    set isVerifyATN(verifyATN: boolean);
    get isGenerateRuleBypassTransitions(): boolean;
    set isGenerateRuleBypassTransitions(generateRuleBypassTransitions: boolean);
    get isOptimize(): boolean;
    set isOptimize(optimize: boolean);
    protected throwIfReadOnly(): void;
}
