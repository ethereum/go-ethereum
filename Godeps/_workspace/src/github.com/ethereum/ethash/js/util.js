// util.js
// Tim Hughes <tim@twistedfury.com>

/*jslint node: true, shadow:true */
"use strict";

function nibbleToChar(nibble)
{
	return String.fromCharCode((nibble < 10 ? 48 : 87) + nibble);
}

function charToNibble(chr)
{
	if (chr >= 48 && chr <= 57)
	{
		return chr - 48;
	}
	if (chr >= 65 && chr <= 70)
	{
		return chr - 65 + 10;
	}
	if (chr >= 97 && chr <= 102)
	{
		return chr - 97 + 10;
	}
	return 0;
}

function stringToBytes(str)
{
	var bytes = new Uint8Array(str.length);
	for (var i = 0; i != str.length; ++i)
	{
		bytes[i] = str.charCodeAt(i);
	}
	return bytes;
}

function hexStringToBytes(str)
{
	var bytes = new Uint8Array(str.length>>>1);
	for (var i = 0; i != bytes.length; ++i)
	{
		bytes[i] = charToNibble(str.charCodeAt(i<<1 | 0)) << 4;
		bytes[i] |= charToNibble(str.charCodeAt(i<<1 | 1));
	}
	return bytes;
}

function bytesToHexString(bytes)
{
	var str = "";
	for (var i = 0; i != bytes.length; ++i)
	{
		str += nibbleToChar(bytes[i] >>> 4);
		str += nibbleToChar(bytes[i] & 0xf);
	}
	return str;
}

function wordsToHexString(words)
{
	return bytesToHexString(new Uint8Array(words.buffer));
}

function uint32ToHexString(num)
{
	var buf = new Uint8Array(4);
	buf[0] = (num >> 24) & 0xff;
	buf[1] = (num >> 16) & 0xff;
	buf[2] = (num >> 8) & 0xff;
	buf[3] = (num >> 0) & 0xff;
	return bytesToHexString(buf);
}

function toWords(input)
{
	if (input instanceof Uint32Array)
	{
		return input;
	}
	else if (input instanceof Uint8Array)
	{
		var tmp = new Uint8Array((input.length + 3) & ~3);
		tmp.set(input);
		return new Uint32Array(tmp.buffer);
	}
	else if (typeof input === typeof "")
	{
		return toWords(stringToBytes(input));
	}
	return null;
}

exports.stringToBytes = stringToBytes;
exports.hexStringToBytes = hexStringToBytes;
exports.bytesToHexString = bytesToHexString;
exports.wordsToHexString = wordsToHexString;
exports.uint32ToHexString = uint32ToHexString;
exports.toWords = toWords;