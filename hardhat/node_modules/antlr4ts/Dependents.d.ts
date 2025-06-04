/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
/**
 *
 * @author Sam Harwell
 */
export declare enum Dependents {
    /**
     * The element is dependent upon the specified rule.
     */
    SELF = 0,
    /**
     * The element is dependent upon the set of the specified rule's parents
     * (rules which directly reference it).
     */
    PARENTS = 1,
    /**
     * The element is dependent upon the set of the specified rule's children
     * (rules which it directly references).
     */
    CHILDREN = 2,
    /**
     * The element is dependent upon the set of the specified rule's ancestors
     * (the transitive closure of `PARENTS` rules).
     */
    ANCESTORS = 3,
    /**
     * The element is dependent upon the set of the specified rule's descendants
     * (the transitive closure of `CHILDREN` rules).
     */
    DESCENDANTS = 4,
    /**
     * The element is dependent upon the set of the specified rule's siblings
     * (the union of `CHILDREN` of its `PARENTS`).
     */
    SIBLINGS = 5,
    /**
     * The element is dependent upon the set of the specified rule's preceeding
     * siblings (the union of `CHILDREN` of its `PARENTS` which
     * appear before a reference to the rule).
     */
    PRECEEDING_SIBLINGS = 6,
    /**
     * The element is dependent upon the set of the specified rule's following
     * siblings (the union of `CHILDREN` of its `PARENTS` which
     * appear after a reference to the rule).
     */
    FOLLOWING_SIBLINGS = 7,
    /**
     * The element is dependent upon the set of the specified rule's preceeding
     * elements (rules which might end before the start of the specified rule
     * while parsing). This is calculated by taking the
     * `PRECEEDING_SIBLINGS` of the rule and each of its
     * `ANCESTORS`, along with the `DESCENDANTS` of those
     * elements.
     */
    PRECEEDING = 8,
    /**
     * The element is dependent upon the set of the specified rule's following
     * elements (rules which might start after the end of the specified rule
     * while parsing). This is calculated by taking the
     * `FOLLOWING_SIBLINGS` of the rule and each of its
     * `ANCESTORS`, along with the `DESCENDANTS` of those
     * elements.
     */
    FOLLOWING = 9
}
