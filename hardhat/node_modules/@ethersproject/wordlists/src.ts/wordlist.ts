"use strict";

// This gets overridden by rollup
const exportWordlist = false;

import { id } from "@ethersproject/hash";
import { defineReadOnly } from "@ethersproject/properties";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
export const logger = new Logger(version);

export abstract class Wordlist {
    readonly locale: string;

    constructor(locale: string) {
        logger.checkAbstract(new.target, Wordlist);
        defineReadOnly(this, "locale", locale);
    }

    abstract getWord(index: number): string;
    abstract getWordIndex(word: string): number;

    // Subclasses may override this
    split(mnemonic: string): Array<string> {
        return mnemonic.toLowerCase().split(/ +/g)
    }

    // Subclasses may override this
    join(words: Array<string>): string {
        return words.join(" ");
    }

    static check(wordlist: Wordlist): string {
        const words = [];
        for (let i = 0; i < 2048; i++) {
            const word = wordlist.getWord(i);
            /* istanbul ignore if */
            if (i !== wordlist.getWordIndex(word)) { return "0x"; }
            words.push(word);
        }
        return id(words.join("\n") + "\n");
    }

    static register(lang: Wordlist, name?: string): void {
        if (!name) { name = lang.locale; }

        /* istanbul ignore if */
        if (exportWordlist) {
            try {
                const anyGlobal = (window as any)
                if (anyGlobal._ethers && anyGlobal._ethers.wordlists) {
                    if (!anyGlobal._ethers.wordlists[name]) {
                         defineReadOnly(anyGlobal._ethers.wordlists, name, lang);
                    }
                }
            } catch (error) { }
        }
    }

}

