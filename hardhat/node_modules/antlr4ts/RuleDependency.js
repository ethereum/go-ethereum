"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.RuleDependency = void 0;
/**
 * Declares a dependency upon a grammar rule, along with a set of zero or more dependent rules.
 *
 * Version numbers within a grammar should be assigned on a monotonically increasing basis to allow for accurate
 * tracking of dependent rules.
 *
 * @author Sam Harwell
 */
function RuleDependency(dependency) {
    return (target, propertyKey, propertyDescriptor) => {
        // intentionally empty
    };
}
exports.RuleDependency = RuleDependency;
//# sourceMappingURL=RuleDependency.js.map