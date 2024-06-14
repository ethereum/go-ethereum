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
/**
 * @file index.d.ts
 * @author Josh Stevens <joshstevens19@hotmail.co.uk>
 * @date 2018
 */

import BN = require('bn.js');

export type Unit =
    | 'noether'
    | 'wei'
    | 'kwei'
    | 'Kwei'
    | 'babbage'
    | 'femtoether'
    | 'mwei'
    | 'Mwei'
    | 'lovelace'
    | 'picoether'
    | 'gwei'
    | 'Gwei'
    | 'shannon'
    | 'nanoether'
    | 'nano'
    | 'szabo'
    | 'microether'
    | 'micro'
    | 'finney'
    | 'milliether'
    | 'milli'
    | 'ether'
    | 'kether'
    | 'grand'
    | 'mether'
    | 'gether'
    | 'tether';

export type Mixed =
    | string
    | number
    | BN
    | {
          type: string;
          value: string;
      }
    | {
          t: string;
          v: string | BN | number;
      }
    | boolean;

export type Hex = string | number;

// utils types
export function isBN(value: string | number): boolean;
export function isBigNumber(value: BN): boolean;
export function toBN(value: number | string): BN;
export function toTwosComplement(value: number | string | BN): string;
export function isAddress(address: string, chainId?: number): boolean;
export function isHex(hex: Hex): boolean;
export function isHexStrict(hex: Hex): boolean;
export function asciiToHex(string: string, length?: number): string;
export function hexToAscii(string: string): string;
export function toAscii(string: string): string;
export function bytesToHex(bytes: number[]): string;
export function numberToHex(value: number | string | BN): string;
export function checkAddressChecksum(address: string): boolean;
export function fromAscii(string: string): string;
export function fromDecimal(value: string | number): string;
export function fromUtf8(string: string): string;
export function fromWei(value: string | BN, unit?: Unit): string;
export function hexToBytes(hex: Hex): number[];
export function hexToNumber(hex: Hex,bigIntOnOverflow?: boolean): number | string;
export function hexToNumberString(hex: Hex): string;
export function hexToString(hex: Hex): string;
export function hexToUtf8(string: string): string;
export function keccak256(value: string | BN): string;
export function padLeft(value: string | number, characterAmount: number, sign?: string): string;
export function leftPad(string: string | number, characterAmount: number, sign?: string): string;
export function rightPad(string: string | number, characterAmount: number, sign?: string): string;
export function padRight(string: string | number, characterAmount: number, sign?: string): string;
export function sha3(value: string | BN | Buffer): string | null;
export function sha3Raw(value: string | BN | Buffer): string;
export function randomHex(bytesSize: number): string;
export function utf8ToHex(string: string): string;
export function stringToHex(string: string): string;
export function toChecksumAddress(address: string): string;
export function toDecimal(hex: Hex): number;
export function toHex(value: number | string | BN): string;
export function toUtf8(string: string): string;
export function toWei(val: BN, unit?: Unit): BN;
export function toWei(val: string, unit?: Unit): string;
export function isBloom(bloom: string): boolean;
export function isInBloom(bloom: string, value: string | Uint8Array): boolean;
export function isUserEthereumAddressInBloom(bloom: string, ethereumAddress: string): boolean;
export function isContractAddressInBloom(bloom: string, contractAddress: string): boolean;
export function isTopicInBloom(bloom: string, topic: string): boolean;
export function isTopic(topic: string): boolean;
export function jsonInterfaceMethodToString(abiItem: AbiItem): string;
export function soliditySha3(...val: Mixed[]): string | null;
export function soliditySha3Raw(...val: Mixed[]): string;
export function encodePacked(...val: Mixed[]): string | null;
export function getUnitValue(unit: Unit): string;
export function unitMap(): Units;
export function testAddress(bloom: string, address: string): boolean;
export function testTopic(bloom: string, topic: string): boolean;
export function getSignatureParameters(signature: string): {r: string; s: string; v: number};
export function stripHexPrefix(str: string): string;
export function toNumber(value: number | string | BN, bigIntOnOverflow?: boolean): number | string;

// interfaces
export interface Utils {
    isBN(value: string | number): boolean;
    isBigNumber(value: BN): boolean;
    toBN(value: number | string): BN;
    toTwosComplement(value: number | string | BN): string;
    isAddress(address: string, chainId?: number): boolean;
    isHex(hex: Hex): boolean;
    isHexStrict(hex: Hex): boolean;
    asciiToHex(string: string, length?: number): string;
    hexToAscii(string: string): string;
    toAscii(string: string): string;
    bytesToHex(bytes: number[]): string;
    numberToHex(value: number | string | BN): string;
    checkAddressChecksum(address: string, chainId?: number): boolean;
    fromAscii(string: string): string;
    fromDecimal(value: string | number): string;
    fromUtf8(string: string): string;
    fromWei(value: string | BN, unit?: Unit): string;
    hexToBytes(hex: Hex): number[];
    hexToNumber(hex: Hex, bigIntOnOverflow?: boolean): number | string;
    hexToNumberString(hex: Hex): string;
    hexToString(hex: Hex): string;
    hexToUtf8(string: string): string;
    keccak256(value: string | BN): string;
    padLeft(value: string | number, characterAmount: number, sign?: string): string;
    leftPad(string: string | number, characterAmount: number, sign?: string): string;
    rightPad(string: string | number, characterAmount: number, sign?: string): string;
    padRight(string: string | number, characterAmount: number, sign?: string): string;
    sha3(value: string | BN): string | null;
    randomHex(bytesSize: number): string;
    utf8ToHex(string: string): string;
    stringToHex(string: string): string;
    toChecksumAddress(address: string): string;
    toDecimal(hex: Hex): number;
    toHex(value: number | string | BN): string;
    toUtf8(string: string): string;
    toWei(val: BN, unit?: Unit): BN;
    toWei(val: string, unit?: Unit): string;
    isBloom(bloom: string): boolean;
    isInBloom(bloom: string, value: string | Uint8Array): boolean;
    isUserEthereumAddressInBloom(bloom: string, ethereumAddress: string): boolean;
    isContractAddressInBloom(bloom: string, contractAddress: string): boolean;
    isTopicInBloom(bloom: string, topic: string): boolean;
    isTopic(topic: string): boolean;
    _jsonInterfaceMethodToString(abiItem: AbiItem): string;
    soliditySha3(...val: Mixed[]): string | null;
    soliditySha3Raw(...val: Mixed[]): string;
    encodePacked(...val: Mixed[]): string | null;
    getUnitValue(unit: Unit): string;
    unitMap(): Units;
    testAddress(bloom: string, address: string): boolean;
    testTopic(bloom: string, topic: string): boolean;
    getSignatureParameters(signature: string): {r: string; s: string; v: number};
    stripHexPrefix(str: string): string;
    toNumber(value: number | string | BN, bigIntOnOverflow?: boolean): number | string;
}

export interface Units {
    noether: string;
    wei: string;
    kwei: string;
    Kwei: string;
    babbage: string;
    femtoether: string;
    mwei: string;
    Mwei: string;
    lovelace: string;
    picoether: string;
    gwei: string;
    Gwei: string;
    shannon: string;
    nanoether: string;
    nano: string;
    szabo: string;
    microether: string;
    micro: string;
    finney: string;
    milliether: string;
    milli: string;
    ether: string;
    kether: string;
    grand: string;
    mether: string;
    gether: string;
    tether: string;
}

export type AbiType = 'function' | 'constructor' | 'event' | 'fallback' | 'receive';
export type StateMutabilityType = 'pure' | 'view' | 'nonpayable' | 'payable';

export interface AbiItem {
    anonymous?: boolean;
    constant?: boolean;
    inputs?: AbiInput[];
    name?: string;
    outputs?: AbiOutput[];
    payable?: boolean;
    stateMutability?: StateMutabilityType;
    type: AbiType;
    gas?: number;
}

export interface AbiInput {
    name: string;
    type: string;
    indexed?: boolean;
	components?: AbiInput[];
    internalType?: string;
}

export interface AbiOutput {
    name: string;
    type: string;
	components?: AbiOutput[];
    internalType?: string;
}
