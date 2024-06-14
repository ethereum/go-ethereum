"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ATNDeserializationOptions = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:25.8187912-07:00
const Decorators_1 = require("../Decorators");
/**
 *
 * @author Sam Harwell
 */
class ATNDeserializationOptions {
    constructor(options) {
        this.readOnly = false;
        if (options) {
            this.verifyATN = options.verifyATN;
            this.generateRuleBypassTransitions = options.generateRuleBypassTransitions;
            this.optimize = options.optimize;
        }
        else {
            this.verifyATN = true;
            this.generateRuleBypassTransitions = false;
            this.optimize = true;
        }
    }
    static get defaultOptions() {
        if (ATNDeserializationOptions._defaultOptions == null) {
            ATNDeserializationOptions._defaultOptions = new ATNDeserializationOptions();
            ATNDeserializationOptions._defaultOptions.makeReadOnly();
        }
        return ATNDeserializationOptions._defaultOptions;
    }
    get isReadOnly() {
        return this.readOnly;
    }
    makeReadOnly() {
        this.readOnly = true;
    }
    get isVerifyATN() {
        return this.verifyATN;
    }
    set isVerifyATN(verifyATN) {
        this.throwIfReadOnly();
        this.verifyATN = verifyATN;
    }
    get isGenerateRuleBypassTransitions() {
        return this.generateRuleBypassTransitions;
    }
    set isGenerateRuleBypassTransitions(generateRuleBypassTransitions) {
        this.throwIfReadOnly();
        this.generateRuleBypassTransitions = generateRuleBypassTransitions;
    }
    get isOptimize() {
        return this.optimize;
    }
    set isOptimize(optimize) {
        this.throwIfReadOnly();
        this.optimize = optimize;
    }
    throwIfReadOnly() {
        if (this.isReadOnly) {
            throw new Error("The object is read only.");
        }
    }
}
__decorate([
    Decorators_1.NotNull
], ATNDeserializationOptions, "defaultOptions", null);
exports.ATNDeserializationOptions = ATNDeserializationOptions;
//# sourceMappingURL=ATNDeserializationOptions.js.map