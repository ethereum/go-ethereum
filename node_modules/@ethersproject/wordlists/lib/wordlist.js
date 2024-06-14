"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Wordlist = exports.logger = void 0;
// This gets overridden by rollup
var exportWordlist = false;
var hash_1 = require("@ethersproject/hash");
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
exports.logger = new logger_1.Logger(_version_1.version);
var Wordlist = /** @class */ (function () {
    function Wordlist(locale) {
        var _newTarget = this.constructor;
        exports.logger.checkAbstract(_newTarget, Wordlist);
        (0, properties_1.defineReadOnly)(this, "locale", locale);
    }
    // Subclasses may override this
    Wordlist.prototype.split = function (mnemonic) {
        return mnemonic.toLowerCase().split(/ +/g);
    };
    // Subclasses may override this
    Wordlist.prototype.join = function (words) {
        return words.join(" ");
    };
    Wordlist.check = function (wordlist) {
        var words = [];
        for (var i = 0; i < 2048; i++) {
            var word = wordlist.getWord(i);
            /* istanbul ignore if */
            if (i !== wordlist.getWordIndex(word)) {
                return "0x";
            }
            words.push(word);
        }
        return (0, hash_1.id)(words.join("\n") + "\n");
    };
    Wordlist.register = function (lang, name) {
        if (!name) {
            name = lang.locale;
        }
        /* istanbul ignore if */
        if (exportWordlist) {
            try {
                var anyGlobal = window;
                if (anyGlobal._ethers && anyGlobal._ethers.wordlists) {
                    if (!anyGlobal._ethers.wordlists[name]) {
                        (0, properties_1.defineReadOnly)(anyGlobal._ethers.wordlists, name, lang);
                    }
                }
            }
            catch (error) { }
        }
    };
    return Wordlist;
}());
exports.Wordlist = Wordlist;
//# sourceMappingURL=wordlist.js.map