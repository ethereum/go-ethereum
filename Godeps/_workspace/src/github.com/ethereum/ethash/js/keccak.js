// keccak.js
// Tim Hughes <tim@twistedfury.com>
// derived from Markku-Juhani O. Saarinen's C code (http://keccak.noekeon.org/readable_code.html)

/*jslint node: true, shadow:true */
"use strict";

var Keccak_f1600_RC = new Uint32Array([
	0x00000001, 0x00000000,
	0x00008082, 0x00000000,
	0x0000808a, 0x80000000,
	0x80008000, 0x80000000,
	0x0000808b, 0x00000000,
	0x80000001, 0x00000000,
	0x80008081, 0x80000000,
	0x00008009, 0x80000000,
	0x0000008a, 0x00000000,
	0x00000088, 0x00000000,
	0x80008009, 0x00000000,
	0x8000000a, 0x00000000,
	0x8000808b, 0x00000000,
	0x0000008b, 0x80000000,
	0x00008089, 0x80000000,
	0x00008003, 0x80000000,
	0x00008002, 0x80000000,
	0x00000080, 0x80000000,
	0x0000800a, 0x00000000,
	0x8000000a, 0x80000000,
	0x80008081, 0x80000000,
	0x00008080, 0x80000000,
	0x80000001, 0x00000000,
	0x80008008, 0x80000000
]);

function keccak_f1600(outState, outOffset, outSize, inState)
{
	// todo, handle big endian loads
	var a00l = inState[0]|0;
	var a00h = inState[1]|0;
	var a01l = inState[2]|0;
	var a01h = inState[3]|0;
	var a02l = inState[4]|0;
	var a02h = inState[5]|0;
	var a03l = inState[6]|0;
	var a03h = inState[7]|0;
	var a04l = inState[8]|0;
	var a04h = inState[9]|0;
	var a05l = inState[10]|0;
	var a05h = inState[11]|0;
	var a06l = inState[12]|0;
	var a06h = inState[13]|0;
	var a07l = inState[14]|0;
	var a07h = inState[15]|0;
	var a08l = inState[16]|0;
	var a08h = inState[17]|0;
	var a09l = inState[18]|0;
	var a09h = inState[19]|0;
	var a10l = inState[20]|0;
	var a10h = inState[21]|0;
	var a11l = inState[22]|0;
	var a11h = inState[23]|0;
	var a12l = inState[24]|0;
	var a12h = inState[25]|0;
	var a13l = inState[26]|0;
	var a13h = inState[27]|0;
	var a14l = inState[28]|0;
	var a14h = inState[29]|0;
	var a15l = inState[30]|0;
	var a15h = inState[31]|0;
	var a16l = inState[32]|0;
	var a16h = inState[33]|0;
	var a17l = inState[34]|0;
	var a17h = inState[35]|0;
	var a18l = inState[36]|0;
	var a18h = inState[37]|0;
	var a19l = inState[38]|0;
	var a19h = inState[39]|0;
	var a20l = inState[40]|0;
	var a20h = inState[41]|0;
	var a21l = inState[42]|0;
	var a21h = inState[43]|0;
	var a22l = inState[44]|0;
	var a22h = inState[45]|0;
	var a23l = inState[46]|0;
	var a23h = inState[47]|0;
	var a24l = inState[48]|0;
	var a24h = inState[49]|0;
	var b00l, b00h, b01l, b01h, b02l, b02h, b03l, b03h, b04l, b04h;
	var b05l, b05h, b06l, b06h, b07l, b07h, b08l, b08h, b09l, b09h;
	var b10l, b10h, b11l, b11h, b12l, b12h, b13l, b13h, b14l, b14h;
	var b15l, b15h, b16l, b16h, b17l, b17h, b18l, b18h, b19l, b19h;
	var b20l, b20h, b21l, b21h, b22l, b22h, b23l, b23h, b24l, b24h;
	var tl, nl;
	var th, nh;

	for (var r = 0; r < 48; r = (r+2)|0)
	{
		// Theta
		b00l = a00l ^ a05l ^ a10l ^ a15l ^ a20l;
		b00h = a00h ^ a05h ^ a10h ^ a15h ^ a20h;
		b01l = a01l ^ a06l ^ a11l ^ a16l ^ a21l;
		b01h = a01h ^ a06h ^ a11h ^ a16h ^ a21h;
		b02l = a02l ^ a07l ^ a12l ^ a17l ^ a22l;
		b02h = a02h ^ a07h ^ a12h ^ a17h ^ a22h;
		b03l = a03l ^ a08l ^ a13l ^ a18l ^ a23l;
		b03h = a03h ^ a08h ^ a13h ^ a18h ^ a23h;
		b04l = a04l ^ a09l ^ a14l ^ a19l ^ a24l;
		b04h = a04h ^ a09h ^ a14h ^ a19h ^ a24h;
		tl = b04l ^ (b01l << 1 | b01h >>> 31);
		th = b04h ^ (b01h << 1 | b01l >>> 31);
		a00l ^= tl;
		a00h ^= th;
		a05l ^= tl;
		a05h ^= th;
		a10l ^= tl;
		a10h ^= th;
		a15l ^= tl;
		a15h ^= th;
		a20l ^= tl;
		a20h ^= th;
		tl = b00l ^ (b02l << 1 | b02h >>> 31);
		th = b00h ^ (b02h << 1 | b02l >>> 31);
		a01l ^= tl;
		a01h ^= th;
		a06l ^= tl;
		a06h ^= th;
		a11l ^= tl;
		a11h ^= th;
		a16l ^= tl;
		a16h ^= th;
		a21l ^= tl;
		a21h ^= th;
		tl = b01l ^ (b03l << 1 | b03h >>> 31);
		th = b01h ^ (b03h << 1 | b03l >>> 31);
		a02l ^= tl;
		a02h ^= th;
		a07l ^= tl;
		a07h ^= th;
		a12l ^= tl;
		a12h ^= th;
		a17l ^= tl;
		a17h ^= th;
		a22l ^= tl;
		a22h ^= th;
		tl = b02l ^ (b04l << 1 | b04h >>> 31);
		th = b02h ^ (b04h << 1 | b04l >>> 31);
		a03l ^= tl;
		a03h ^= th;
		a08l ^= tl;
		a08h ^= th;
		a13l ^= tl;
		a13h ^= th;
		a18l ^= tl;
		a18h ^= th;
		a23l ^= tl;
		a23h ^= th;
		tl = b03l ^ (b00l << 1 | b00h >>> 31);
		th = b03h ^ (b00h << 1 | b00l >>> 31);
		a04l ^= tl;
		a04h ^= th;
		a09l ^= tl;
		a09h ^= th;
		a14l ^= tl;
		a14h ^= th;
		a19l ^= tl;
		a19h ^= th;
		a24l ^= tl;
		a24h ^= th;

		// Rho Pi
		b00l = a00l;
		b00h = a00h;
		b10l = a01l << 1 | a01h >>> 31;
		b10h = a01h << 1 | a01l >>> 31;
		b07l = a10l << 3 | a10h >>> 29;
		b07h = a10h << 3 | a10l >>> 29;
		b11l = a07l << 6 | a07h >>> 26;
		b11h = a07h << 6 | a07l >>> 26;
		b17l = a11l << 10 | a11h >>> 22;
		b17h = a11h << 10 | a11l >>> 22;
		b18l = a17l << 15 | a17h >>> 17;
		b18h = a17h << 15 | a17l >>> 17;
		b03l = a18l << 21 | a18h >>> 11;
		b03h = a18h << 21 | a18l >>> 11;
		b05l = a03l << 28 | a03h >>> 4;
		b05h = a03h << 28 | a03l >>> 4;
		b16l = a05h << 4 | a05l >>> 28;
		b16h = a05l << 4 | a05h >>> 28;
		b08l = a16h << 13 | a16l >>> 19;
		b08h = a16l << 13 | a16h >>> 19;
		b21l = a08h << 23 | a08l >>> 9;
		b21h = a08l << 23 | a08h >>> 9;
		b24l = a21l << 2 | a21h >>> 30;
		b24h = a21h << 2 | a21l >>> 30;
		b04l = a24l << 14 | a24h >>> 18;
		b04h = a24h << 14 | a24l >>> 18;
		b15l = a04l << 27 | a04h >>> 5;
		b15h = a04h << 27 | a04l >>> 5;
		b23l = a15h << 9 | a15l >>> 23;
		b23h = a15l << 9 | a15h >>> 23;
		b19l = a23h << 24 | a23l >>> 8;
		b19h = a23l << 24 | a23h >>> 8;
		b13l = a19l << 8 | a19h >>> 24;
		b13h = a19h << 8 | a19l >>> 24;
		b12l = a13l << 25 | a13h >>> 7;
		b12h = a13h << 25 | a13l >>> 7;
		b02l = a12h << 11 | a12l >>> 21;
		b02h = a12l << 11 | a12h >>> 21;
		b20l = a02h << 30 | a02l >>> 2;
		b20h = a02l << 30 | a02h >>> 2;
		b14l = a20l << 18 | a20h >>> 14;
		b14h = a20h << 18 | a20l >>> 14;
		b22l = a14h << 7 | a14l >>> 25;
		b22h = a14l << 7 | a14h >>> 25;
		b09l = a22h << 29 | a22l >>> 3;
		b09h = a22l << 29 | a22h >>> 3;
		b06l = a09l << 20 | a09h >>> 12;
		b06h = a09h << 20 | a09l >>> 12;
		b01l = a06h << 12 | a06l >>> 20;
		b01h = a06l << 12 | a06h >>> 20;

		// Chi
		a00l = b00l ^ ~b01l & b02l;
		a00h = b00h ^ ~b01h & b02h;
		a01l = b01l ^ ~b02l & b03l;
		a01h = b01h ^ ~b02h & b03h;
		a02l = b02l ^ ~b03l & b04l;
		a02h = b02h ^ ~b03h & b04h;
		a03l = b03l ^ ~b04l & b00l;
		a03h = b03h ^ ~b04h & b00h;
		a04l = b04l ^ ~b00l & b01l;
		a04h = b04h ^ ~b00h & b01h;
		a05l = b05l ^ ~b06l & b07l;
		a05h = b05h ^ ~b06h & b07h;
		a06l = b06l ^ ~b07l & b08l;
		a06h = b06h ^ ~b07h & b08h;
		a07l = b07l ^ ~b08l & b09l;
		a07h = b07h ^ ~b08h & b09h;
		a08l = b08l ^ ~b09l & b05l;
		a08h = b08h ^ ~b09h & b05h;
		a09l = b09l ^ ~b05l & b06l;
		a09h = b09h ^ ~b05h & b06h;
		a10l = b10l ^ ~b11l & b12l;
		a10h = b10h ^ ~b11h & b12h;
		a11l = b11l ^ ~b12l & b13l;
		a11h = b11h ^ ~b12h & b13h;
		a12l = b12l ^ ~b13l & b14l;
		a12h = b12h ^ ~b13h & b14h;
		a13l = b13l ^ ~b14l & b10l;
		a13h = b13h ^ ~b14h & b10h;
		a14l = b14l ^ ~b10l & b11l;
		a14h = b14h ^ ~b10h & b11h;
		a15l = b15l ^ ~b16l & b17l;
		a15h = b15h ^ ~b16h & b17h;
		a16l = b16l ^ ~b17l & b18l;
		a16h = b16h ^ ~b17h & b18h;
		a17l = b17l ^ ~b18l & b19l;
		a17h = b17h ^ ~b18h & b19h;
		a18l = b18l ^ ~b19l & b15l;
		a18h = b18h ^ ~b19h & b15h;
		a19l = b19l ^ ~b15l & b16l;
		a19h = b19h ^ ~b15h & b16h;
		a20l = b20l ^ ~b21l & b22l;
		a20h = b20h ^ ~b21h & b22h;
		a21l = b21l ^ ~b22l & b23l;
		a21h = b21h ^ ~b22h & b23h;
		a22l = b22l ^ ~b23l & b24l;
		a22h = b22h ^ ~b23h & b24h;
		a23l = b23l ^ ~b24l & b20l;
		a23h = b23h ^ ~b24h & b20h;
		a24l = b24l ^ ~b20l & b21l;
		a24h = b24h ^ ~b20h & b21h;

		// Iota
		a00l ^= Keccak_f1600_RC[r|0];
		a00h ^= Keccak_f1600_RC[r|1];
	}
	
	// todo, handle big-endian stores
	outState[outOffset|0] = a00l;
	outState[outOffset|1] = a00h;
	outState[outOffset|2] = a01l;
	outState[outOffset|3] = a01h;
	outState[outOffset|4] = a02l;
	outState[outOffset|5] = a02h;
	outState[outOffset|6] = a03l;
	outState[outOffset|7] = a03h;
	if (outSize == 8)
		return;
	outState[outOffset|8] = a04l;
	outState[outOffset|9] = a04h;
	outState[outOffset|10] = a05l;
	outState[outOffset|11] = a05h;
	outState[outOffset|12] = a06l;
	outState[outOffset|13] = a06h;
	outState[outOffset|14] = a07l;
	outState[outOffset|15] = a07h;
	if (outSize == 16)
		return;
	outState[outOffset|16] = a08l;
	outState[outOffset|17] = a08h;
	outState[outOffset|18] = a09l;
	outState[outOffset|19] = a09h;
	outState[outOffset|20] = a10l;
	outState[outOffset|21] = a10h;
	outState[outOffset|22] = a11l;
	outState[outOffset|23] = a11h;
	outState[outOffset|24] = a12l;
	outState[outOffset|25] = a12h;
	outState[outOffset|26] = a13l;
	outState[outOffset|27] = a13h;
	outState[outOffset|28] = a14l;
	outState[outOffset|29] = a14h;
	outState[outOffset|30] = a15l;
	outState[outOffset|31] = a15h;
	outState[outOffset|32] = a16l;
	outState[outOffset|33] = a16h;
	outState[outOffset|34] = a17l;
	outState[outOffset|35] = a17h;
	outState[outOffset|36] = a18l;
	outState[outOffset|37] = a18h;
	outState[outOffset|38] = a19l;
	outState[outOffset|39] = a19h;
	outState[outOffset|40] = a20l;
	outState[outOffset|41] = a20h;
	outState[outOffset|42] = a21l;
	outState[outOffset|43] = a21h;
	outState[outOffset|44] = a22l;
	outState[outOffset|45] = a22h;
	outState[outOffset|46] = a23l;
	outState[outOffset|47] = a23h;
	outState[outOffset|48] = a24l;
	outState[outOffset|49] = a24h;
}

var Keccak = function()
{
	var stateBuf = new ArrayBuffer(200);
	var stateBytes = new Uint8Array(stateBuf);
	var stateWords = new Uint32Array(stateBuf);
	
	this.digest = function(oSize, iBytes)
	{
		for (var i = 0; i < 50; ++i)
		{
			stateWords[i] = 0;
		}
		
		var r = 200 - oSize*2;
		var iLength = iBytes.length;
		var iOffset = 0;	
		for ( ; ;)
		{
			var len = iLength < r ? iLength : r;
			for (i = 0; i < len; ++i, ++iOffset)
			{
				stateBytes[i] ^= iBytes[iOffset];
			}
			
			if (iLength < r)
				break;
			iLength -= len;
			
			keccak_f1600(stateWords, 0, 50, stateWords);
		}
		
		stateBytes[iLength] ^= 1;
		stateBytes[r-1] ^= 0x80;
		keccak_f1600(stateWords, 0, 50, stateWords);
		return stateBytes.subarray(0, oSize);
	};
	
	this.digestWords = function(oWords, oOffset, oLength, iWords, iOffset, iLength)
	{
		for (var i = 0; i < 50; ++i)
		{
			stateWords[i] = 0;
		}
		
		var r = 50 - oLength*2;
		for (; ; )
		{
			var len = iLength < r ? iLength : r;
			for (i = 0; i < len; ++i, ++iOffset)
			{
				stateWords[i] ^= iWords[iOffset];
			}
			
			if (iLength < r)
				break;
			iLength -= len;
			
			keccak_f1600(stateWords, 0, 50, stateWords);
		}
		
		stateBytes[iLength<<2] ^= 1;
		stateBytes[(r<<2) - 1] ^= 0x80;
		keccak_f1600(oWords, oOffset, oLength, stateWords);
	};
};

module.exports = Keccak;


