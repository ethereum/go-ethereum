/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { BitSet } from "../misc/BitSet";
/**
 * This class stores information about a configuration conflict.
 *
 * @author Sam Harwell
 */
export declare class ConflictInfo {
    private _conflictedAlts;
    private exact;
    constructor(conflictedAlts: BitSet, exact: boolean);
    /**
     * Gets the set of conflicting alternatives for the configuration set.
     */
    get conflictedAlts(): BitSet;
    /**
     * Gets whether or not the configuration conflict is an exact conflict.
     * An exact conflict occurs when the prediction algorithm determines that
     * the represented alternatives for a particular configuration set cannot be
     * further reduced by consuming additional input. After reaching an exact
     * conflict during an SLL prediction, only switch to full-context prediction
     * could reduce the set of viable alternatives. In LL prediction, an exact
     * conflict indicates a true ambiguity in the input.
     *
     * For the {@link PredictionMode#LL_EXACT_AMBIG_DETECTION} prediction mode,
     * accept states are conflicting but not exact are treated as non-accept
     * states.
     */
    get isExact(): boolean;
    equals(obj: any): boolean;
    hashCode(): number;
}
