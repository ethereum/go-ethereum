import { HexString } from 'web3-types';
import { IbanOptions } from './types.js';
/**
 * Converts Ethereum addresses to IBAN or BBAN addresses and vice versa.
 */
export declare class Iban {
    private readonly _iban;
    /**
     * Prepare an IBAN for mod 97 computation by moving the first 4 chars to the end and transforming the letters to
     * numbers (A = 10, B = 11, ..., Z = 35), as specified in ISO13616.
     */
    private static readonly _iso13616Prepare;
    /**
     * return the bigint of the given string with the specified base
     */
    private static readonly _parseInt;
    /**
     * Calculates the MOD 97 10 of the passed IBAN as specified in ISO7064.
     */
    private static readonly _mod9710;
    /**
     * A static method that checks if an IBAN is Direct.
     * It actually check the length of the provided variable and, only if it is 34 or 35, it returns true.
     * Note: this is also available as a method at an Iban instance.
     * @param iban - an IBAN to be checked
     * @returns - `true` if the provided `iban` is a Direct IBAN, and `false` otherwise.
     *
     * @example
     * ```ts
     * web3.eth.Iban.isDirect("XE81ETHXREGGAVOFYORK");
     * > false
     * ```
     */
    static isDirect(iban: string): boolean;
    /**
     * An instance method that checks if iban number is Direct.
     * It actually check the length of the provided variable and, only if it is 34 or 35, it returns true.
     * Note: this is also available as a static method.
     * @param iban - an IBAN to be checked
     * @returns - `true` if the provided `iban` is a Direct IBAN, and `false` otherwise.
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban("XE81ETHXREGGAVOFYORK");
     * iban.isDirect();
     * > false
     * ```
     */
    isDirect(): boolean;
    /**
     * A static method that checks if an IBAN is Indirect.
     * It actually check the length of the provided variable and, only if it is 20, it returns true.
     * Note: this is also available as a method at an Iban instance.
     * @param iban - an IBAN to be checked
     * @returns - `true` if the provided `iban` is an Indirect IBAN, and `false` otherwise.
     *
     * @example
     * ```ts
     * web3.eth.Iban.isIndirect("XE81ETHXREGGAVOFYORK");
     * > true
     * ```
     */
    static isIndirect(iban: string): boolean;
    /**
     * check if iban number if indirect
     * It actually check the length of the provided variable and, only if it is 20, it returns true.
     * Note: this is also available as a static method.
     * @param iban - an IBAN to be checked
     * @returns - `true` if the provided `iban` is an Indirect IBAN, and `false` otherwise.
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban("XE81ETHXREGGAVOFYORK");
     * iban.isIndirect();
     * > true
     * ```
     */
    isIndirect(): boolean;
    /**
     * This method could be used to check if a given string is valid IBAN object.
     * Note: this is also available as a method at an Iban instance.
     *
     * @param iban - a string to be checked if it is in IBAN
     * @returns - true if it is valid IBAN
     *
     * @example
     * ```ts
     * web3.eth.Iban.isValid("XE81ETHXREGGAVOFYORK");
     * > true
     *
     * web3.eth.Iban.isValid("XE82ETHXREGGAVOFYORK");
     * > false // because the checksum is incorrect
     * ```
     */
    static isValid(iban: string): boolean;
    /**
     * Should be called to check if the early provided IBAN is correct.
     * Note: this is also available as a static method.
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban("XE81ETHXREGGAVOFYORK");
     * iban.isValid();
     * > true
     *
     * const iban = new web3.eth.Iban("XE82ETHXREGGAVOFYORK");
     * iban.isValid();
     * > false // because the checksum is incorrect
     * ```
     */
    isValid(): boolean;
    /**
     * Construct a direct or indirect IBAN that has conversion methods and validity checks.
     * If the provided string was not of either the length of a direct IBAN (34 or 35),
     * nor the length of an indirect IBAN (20), an Error will be thrown ('Invalid IBAN was provided').
     *
     * @param iban - a Direct or an Indirect IBAN
     * @returns - Iban instance
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban("XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS");
     * > Iban { _iban: 'XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS' }
     * ```
     */
    constructor(iban: string);
    /**
     * Convert the passed BBAN to an IBAN for this country specification.
     * Please note that <i>"generation of the IBAN shall be the exclusive responsibility of the bank/branch servicing the account"</i>.
     * This method implements the preferred algorithm described in http://en.wikipedia.org/wiki/International_Bank_Account_Number#Generating_IBAN_check_digits
     *
     * @param bban - the BBAN to convert to IBAN
     * @returns an Iban class instance that holds the equivalent IBAN
     *
     * @example
     * ```ts
     * web3.eth.Iban.fromBban('ETHXREGGAVOFYORK');
     * > Iban {_iban: "XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS"}
     * ```
     */
    static fromBban(bban: string): Iban;
    /**
     * Should be used to create IBAN object for given institution and identifier
     *
     * @param options - an object holds the `institution` and the `identifier` which will be composed to create an `Iban` object from.
     * @returns an Iban class instance that holds the equivalent IBAN
     *
     * @example
     * ```ts
     * web3.eth.Iban.createIndirect({
     *     institution: "XREG",
     *     identifier: "GAVOFYORK"
     * });
     * > Iban {_iban: "XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS"}
     * ```
     */
    static createIndirect(options: IbanOptions): Iban;
    /**
     * This method should be used to create iban object from an Ethereum address.
     *
     * @param address - an Ethereum address
     * @returns an Iban class instance that holds the equivalent IBAN
     *
     * @example
     * ```ts
     * web3.eth.Iban.fromAddress("0x00c5496aEe77C1bA1f0854206A26DdA82a81D6D8");
     * > Iban {_iban: "XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS"}
     * ```
     */
    static fromAddress(address: HexString): Iban;
    /**
     * This method should be used to create an ethereum address from a Direct IBAN address.
     * If the provided string was not a direct IBAN (has the length of 34 or 35), an Error will be thrown:
     * ('Iban is indirect and cannot be converted. Must be length of 34 or 35').
     * Note: this is also available as a method at an Iban instance.
     *
     * @param iban - a Direct IBAN address
     * @return the equivalent ethereum address
     *
     * @example
     * ```ts
     * web3.eth.Iban.toAddress("XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS");
     * > "0x00c5496aEe77C1bA1f0854206A26DdA82a81D6D8"
     * ```
     */
    static toAddress: (iban: string) => HexString;
    /**
     * This method should be used to create the equivalent ethereum address for the early provided Direct IBAN address.
     * If the provided string was not a direct IBAN (has the length of 34 or 35), an Error will be thrown:
     * ('Iban is indirect and cannot be converted. Must be length of 34 or 35').
     * Note: this is also available as a static method.
     *
     * @return the equivalent ethereum address
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban("XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS");
     * iban.toAddress();
     * > "0x00c5496aEe77C1bA1f0854206A26DdA82a81D6D8"
     * ```
     */
    toAddress: () => HexString;
    /**
     * This method should be used to create IBAN address from an Ethereum address
     *
     * @param address - an Ethereum address
     * @return the equivalent IBAN address
     *
     * @example
     * ```ts
     * web3.eth.Iban.toIban("0x00c5496aEe77C1bA1f0854206A26DdA82a81D6D8");
     * > "XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS"
     * ```
     */
    static toIban(address: HexString): string;
    /**
     * Should be called to get client identifier within institution
     *
     * @return the client of the IBAN instance.
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban("XE81ETHXREGGAVOFYORK");
     * iban.client();
     * > 'GAVOFYORK'
     * ```
     */
    client(): string;
    /**
     * Returns the IBAN checksum of the early provided IBAN
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban("XE81ETHXREGGAVOFYORK");
     * iban.checksum();
     * > "81"
     * ```
     *
     */
    checksum(): string;
    /**
     * Returns institution identifier from the early provided  IBAN
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban("XE81ETHXREGGAVOFYORK");
     * iban.institution();
     * > 'XREG'
     * ```
     */
    institution(): string;
    /**
     * Simply returns the early provided IBAN
     *
     * @example
     * ```ts
     * const iban = new web3.eth.Iban('XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS');
     * iban.toString();
     * > 'XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS'
     * ```
     */
    toString(): string;
}
//# sourceMappingURL=iban.d.ts.map