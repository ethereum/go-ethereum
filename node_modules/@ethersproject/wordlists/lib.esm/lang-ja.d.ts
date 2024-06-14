import { Wordlist } from "./wordlist";
declare class LangJa extends Wordlist {
    constructor();
    getWord(index: number): string;
    getWordIndex(word: string): number;
    split(mnemonic: string): Array<string>;
    join(words: Array<string>): string;
}
declare const langJa: LangJa;
export { langJa };
//# sourceMappingURL=lang-ja.d.ts.map