"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.SuppressWarnings = exports.Override = exports.Nullable = exports.NotNull = void 0;
function NotNull(target, propertyKey, propertyDescriptor) {
    // intentionally empty
}
exports.NotNull = NotNull;
function Nullable(target, propertyKey, propertyDescriptor) {
    // intentionally empty
}
exports.Nullable = Nullable;
function Override(target, propertyKey, propertyDescriptor) {
    // do something with 'target' ...
}
exports.Override = Override;
function SuppressWarnings(options) {
    return (target, propertyKey, descriptor) => {
        // intentionally empty
    };
}
exports.SuppressWarnings = SuppressWarnings;
//# sourceMappingURL=Decorators.js.map