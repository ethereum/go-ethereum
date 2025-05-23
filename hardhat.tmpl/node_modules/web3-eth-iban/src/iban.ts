/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/

import { HexString } from 'web3-types';
import { toChecksumAddress, leftPad, hexToNumber } from 'web3-utils';
import { isAddress } from 'web3-validator';
import { InvalidAddressError } from 'web3-errors';
import { IbanOptions } from './types.js';

/**
 * Converts Ethereum addresses to IBAN or BBAN addresses and vice versa.
 */
export class Iban {
	private readonly _iban: string;

	/**
	 * Prepare an IBAN for mod 97 computation by moving the first 4 chars to the end and transforming the letters to
	 * numbers (A = 10, B = 11, ..., Z = 35), as specified in ISO13616.
	 */
	private static readonly _iso13616Prepare = (iban: string): string => {
		const A = 'A'.charCodeAt(0);
		const Z = 'Z'.charCodeAt(0);

		const upperIban = iban.toUpperCase();
		const modifiedIban = `${upperIban.slice(4)}${upperIban.slice(0, 4)}`;

		return modifiedIban
			.split('')
			.map(n => {
				const code = n.charCodeAt(0);
				if (code >= A && code <= Z) {
					// A = 10, B = 11, ... Z = 35
					return code - A + 10;
				}
				return n;
			})
			.join('');
	};

	/**
	 * return the bigint of the given string with the specified base
	 */
	private static readonly _parseInt = (str: string, base: number): bigint =>
		[...str].reduce(
			(acc, curr) => BigInt(parseInt(curr, base)) + BigInt(base) * acc,
			BigInt(0),
		);

	/**
	 * Calculates the MOD 97 10 of the passed IBAN as specified in ISO7064.
	 */
	private static readonly _mod9710 = (iban: string): number => {
		let remainder = iban;
		let block;

		while (remainder.length > 2) {
			block = remainder.slice(0, 9);
			remainder = `${(parseInt(block, 10) % 97).toString()}${remainder.slice(block.length)}`;
		}

		return parseInt(remainder, 10) % 97;
	};

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
	public static isDirect(iban: string): boolean {
		return iban.length === 34 || iban.length === 35;
	}

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
	public isDirect(): boolean {
		return Iban.isDirect(this._iban);
	}

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
	public static isIndirect(iban: string): boolean {
		return iban.length === 20;
	}

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
	public isIndirect(): boolean {
		return Iban.isIndirect(this._iban);
	}

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
	public static isValid(iban: string) {
		return (
			/^XE[0-9]{2}(ETH[0-9A-Z]{13}|[0-9A-Z]{30,31})$/.test(iban) &&
			Iban._mod9710(Iban._iso13616Prepare(iban)) === 1
		);
	}

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
	public isValid(): boolean {
		return Iban.isValid(this._iban);
	}

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
	public constructor(iban: string) {
		if (Iban.isIndirect(iban) || Iban.isDirect(iban)) {
			this._iban = iban;
		} else {
			throw new Error('Invalid IBAN was provided');
		}
	}

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
	public static fromBban(bban: string): Iban {
		const countryCode = 'XE';

		const remainder = this._mod9710(this._iso13616Prepare(`${countryCode}00${bban}`));
		const checkDigit = `0${(98 - remainder).toString()}`.slice(-2);

		return new Iban(`${countryCode}${checkDigit}${bban}`);
	}

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
	public static createIndirect(options: IbanOptions): Iban {
		return Iban.fromBban(`ETH${options.institution}${options.identifier}`);
	}

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
	public static fromAddress(address: HexString): Iban {
		if (!isAddress(address)) {
			throw new InvalidAddressError(address);
		}

		const num = BigInt(hexToNumber(address));
		const base36 = num.toString(36);
		const padded = leftPad(base36, 15);
		return Iban.fromBban(padded.toUpperCase());
	}

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
	public static toAddress = (iban: string): HexString => {
		const ibanObject = new Iban(iban);
		return ibanObject.toAddress();
	};

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
	public toAddress = (): HexString => {
		if (this.isDirect()) {
			// check if Iban can be converted to an address
			const base36 = this._iban.slice(4);
			const parsedBigInt = Iban._parseInt(base36, 36); // convert the base36 string to a bigint
			const paddedBigInt = leftPad(parsedBigInt, 40);
			return toChecksumAddress(paddedBigInt);
		}
		throw new Error('Iban is indirect and cannot be converted. Must be length of 34 or 35');
	};

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
	public static toIban(address: HexString): string {
		return Iban.fromAddress(address).toString();
	}

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
	public client(): string {
		return this.isIndirect() ? this._iban.slice(11) : '';
	}

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
	public checksum(): string {
		return this._iban.slice(2, 4);
	}

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
	public institution(): string {
		return this.isIndirect() ? this._iban.slice(7, 11) : '';
	}

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
	public toString(): string {
		return this._iban;
	}
}
