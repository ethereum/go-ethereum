import { Logger } from "@ethersproject/logger";
export declare const logger: Logger;
export declare abstract class Wordlist {
    readonly locale: string;
    constructor(locale: string);
    abstract getWord(index: number): string;
    abstract getWordIndex(word: string): number;
    split(mnemonic: string): Array<string>;
    join(words: Array<string>): string;
    static check(wordlist: Wordlist): string;
    static register(lang: Wordlist, name?: string): void;
}
//# sourceMappingURL=wordlist.d.ts.map