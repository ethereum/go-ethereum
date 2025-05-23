"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.MultiMap = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:42.1346951-07:00
class MultiMap extends Map {
    constructor() {
        super();
    }
    map(key, value) {
        let elementsForKey = super.get(key);
        if (!elementsForKey) {
            elementsForKey = [];
            super.set(key, elementsForKey);
        }
        elementsForKey.push(value);
    }
    getPairs() {
        let pairs = [];
        this.forEach((values, key) => {
            values.forEach((v) => {
                pairs.push([key, v]);
            });
        });
        return pairs;
    }
}
exports.MultiMap = MultiMap;
//# sourceMappingURL=MultiMap.js.map