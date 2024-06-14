/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNConfig } from "./ATNConfig";
import { ATNConfigSet } from "./ATNConfigSet";
/**
 *
 * @author Sam Harwell
 */
export declare class OrderedATNConfigSet extends ATNConfigSet {
    constructor();
    constructor(set: ATNConfigSet, readonly: boolean);
    clone(readonly: boolean): ATNConfigSet;
    protected getKey(e: ATNConfig): {
        state: number;
        alt: number;
    };
    protected canMerge(left: ATNConfig, leftKey: {
        state: number;
        alt: number;
    }, right: ATNConfig): boolean;
}
