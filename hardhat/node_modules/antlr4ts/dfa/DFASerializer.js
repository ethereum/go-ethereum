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
exports.DFASerializer = void 0;
const ATNSimulator_1 = require("../atn/ATNSimulator");
const Decorators_1 = require("../Decorators");
const PredictionContext_1 = require("../atn/PredictionContext");
const Recognizer_1 = require("../Recognizer");
const VocabularyImpl_1 = require("../VocabularyImpl");
/** A DFA walker that knows how to dump them to serialized strings. */
class DFASerializer {
    constructor(dfa, vocabulary, ruleNames, atn) {
        if (vocabulary instanceof Recognizer_1.Recognizer) {
            ruleNames = vocabulary.ruleNames;
            atn = vocabulary.atn;
            vocabulary = vocabulary.vocabulary;
        }
        else if (!vocabulary) {
            vocabulary = VocabularyImpl_1.VocabularyImpl.EMPTY_VOCABULARY;
        }
        this.dfa = dfa;
        this.vocabulary = vocabulary;
        this.ruleNames = ruleNames;
        this.atn = atn;
    }
    toString() {
        if (!this.dfa.s0) {
            return "";
        }
        let buf = "";
        if (this.dfa.states) {
            let states = new Array(...this.dfa.states.toArray());
            states.sort((o1, o2) => o1.stateNumber - o2.stateNumber);
            for (let s of states) {
                let edges = s.getEdgeMap();
                let edgeKeys = [...edges.keys()].sort((a, b) => a - b);
                let contextEdges = s.getContextEdgeMap();
                let contextEdgeKeys = [...contextEdges.keys()].sort((a, b) => a - b);
                for (let entry of edgeKeys) {
                    let value = edges.get(entry);
                    if ((value == null || value === ATNSimulator_1.ATNSimulator.ERROR) && !s.isContextSymbol(entry)) {
                        continue;
                    }
                    let contextSymbol = false;
                    buf += (this.getStateString(s)) + ("-") + (this.getEdgeLabel(entry)) + ("->");
                    if (s.isContextSymbol(entry)) {
                        buf += ("!");
                        contextSymbol = true;
                    }
                    let t = value;
                    if (t && t.stateNumber !== ATNSimulator_1.ATNSimulator.ERROR.stateNumber) {
                        buf += (this.getStateString(t)) + ("\n");
                    }
                    else if (contextSymbol) {
                        buf += ("ctx\n");
                    }
                }
                if (s.isContextSensitive) {
                    for (let entry of contextEdgeKeys) {
                        buf += (this.getStateString(s))
                            + ("-")
                            + (this.getContextLabel(entry))
                            + ("->")
                            + (this.getStateString(contextEdges.get(entry)))
                            + ("\n");
                    }
                }
            }
        }
        let output = buf;
        if (output.length === 0) {
            return "";
        }
        //return Utils.sortLinesInString(output);
        return output;
    }
    getContextLabel(i) {
        if (i === PredictionContext_1.PredictionContext.EMPTY_FULL_STATE_KEY) {
            return "ctx:EMPTY_FULL";
        }
        else if (i === PredictionContext_1.PredictionContext.EMPTY_LOCAL_STATE_KEY) {
            return "ctx:EMPTY_LOCAL";
        }
        if (this.atn && i > 0 && i <= this.atn.states.length) {
            let state = this.atn.states[i];
            let ruleIndex = state.ruleIndex;
            if (this.ruleNames && ruleIndex >= 0 && ruleIndex < this.ruleNames.length) {
                return "ctx:" + String(i) + "(" + this.ruleNames[ruleIndex] + ")";
            }
        }
        return "ctx:" + String(i);
    }
    getEdgeLabel(i) {
        return this.vocabulary.getDisplayName(i);
    }
    getStateString(s) {
        if (s === ATNSimulator_1.ATNSimulator.ERROR) {
            return "ERROR";
        }
        let n = s.stateNumber;
        let stateStr = "s" + n;
        if (s.isAcceptState) {
            if (s.predicates) {
                stateStr = ":s" + n + "=>" + s.predicates;
            }
            else {
                stateStr = ":s" + n + "=>" + s.prediction;
            }
        }
        if (s.isContextSensitive) {
            stateStr += "*";
            for (let config of s.configs) {
                if (config.reachesIntoOuterContext) {
                    stateStr += "*";
                    break;
                }
            }
        }
        return stateStr;
    }
}
__decorate([
    Decorators_1.NotNull
], DFASerializer.prototype, "dfa", void 0);
__decorate([
    Decorators_1.NotNull
], DFASerializer.prototype, "vocabulary", void 0);
__decorate([
    Decorators_1.Override
], DFASerializer.prototype, "toString", null);
exports.DFASerializer = DFASerializer;
//# sourceMappingURL=DFASerializer.js.map