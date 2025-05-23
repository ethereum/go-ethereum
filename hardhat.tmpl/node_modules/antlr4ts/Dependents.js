"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.Dependents = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:51.1349829-07:00
/**
 *
 * @author Sam Harwell
 */
var Dependents;
(function (Dependents) {
    /**
     * The element is dependent upon the specified rule.
     */
    Dependents[Dependents["SELF"] = 0] = "SELF";
    /**
     * The element is dependent upon the set of the specified rule's parents
     * (rules which directly reference it).
     */
    Dependents[Dependents["PARENTS"] = 1] = "PARENTS";
    /**
     * The element is dependent upon the set of the specified rule's children
     * (rules which it directly references).
     */
    Dependents[Dependents["CHILDREN"] = 2] = "CHILDREN";
    /**
     * The element is dependent upon the set of the specified rule's ancestors
     * (the transitive closure of `PARENTS` rules).
     */
    Dependents[Dependents["ANCESTORS"] = 3] = "ANCESTORS";
    /**
     * The element is dependent upon the set of the specified rule's descendants
     * (the transitive closure of `CHILDREN` rules).
     */
    Dependents[Dependents["DESCENDANTS"] = 4] = "DESCENDANTS";
    /**
     * The element is dependent upon the set of the specified rule's siblings
     * (the union of `CHILDREN` of its `PARENTS`).
     */
    Dependents[Dependents["SIBLINGS"] = 5] = "SIBLINGS";
    /**
     * The element is dependent upon the set of the specified rule's preceeding
     * siblings (the union of `CHILDREN` of its `PARENTS` which
     * appear before a reference to the rule).
     */
    Dependents[Dependents["PRECEEDING_SIBLINGS"] = 6] = "PRECEEDING_SIBLINGS";
    /**
     * The element is dependent upon the set of the specified rule's following
     * siblings (the union of `CHILDREN` of its `PARENTS` which
     * appear after a reference to the rule).
     */
    Dependents[Dependents["FOLLOWING_SIBLINGS"] = 7] = "FOLLOWING_SIBLINGS";
    /**
     * The element is dependent upon the set of the specified rule's preceeding
     * elements (rules which might end before the start of the specified rule
     * while parsing). This is calculated by taking the
     * `PRECEEDING_SIBLINGS` of the rule and each of its
     * `ANCESTORS`, along with the `DESCENDANTS` of those
     * elements.
     */
    Dependents[Dependents["PRECEEDING"] = 8] = "PRECEEDING";
    /**
     * The element is dependent upon the set of the specified rule's following
     * elements (rules which might start after the end of the specified rule
     * while parsing). This is calculated by taking the
     * `FOLLOWING_SIBLINGS` of the rule and each of its
     * `ANCESTORS`, along with the `DESCENDANTS` of those
     * elements.
     */
    Dependents[Dependents["FOLLOWING"] = 9] = "FOLLOWING";
})(Dependents = exports.Dependents || (exports.Dependents = {}));
//# sourceMappingURL=Dependents.js.map