#!/usr/bin/env node
// makekeccak.js
// Tim Hughes <tim@twistedfury.com>

/*jslint node: true, shadow:true */
"use strict";

var Keccak_f1600_Rho = [
	1,  3,  6,  10, 15, 21, 28, 36, 45, 55, 2,  14, 
	27, 41, 56, 8,  25, 43, 62, 18, 39, 61, 20, 44
];

var Keccak_f1600_Pi= [
	10, 7,  11, 17, 18, 3, 5,  16, 8,  21, 24, 4, 
	15, 23, 19, 13, 12, 2, 20, 14, 22, 9,  6,  1 
];

var Keccak_f1600_RC = [ 
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
	0x80008008, 0x80000000,
];

function makeRotLow(lo, hi, n)
{
	if (n === 0 || n === 32) throw Error("unsupported");
	if ((n & 0x20) !== 0)
	{
		n &= ~0x20;
		var t = hi;
		hi = lo;
		lo = t;
	}
	var hir = hi + " >>> " + (32 - n);
	var los = lo + " << " + n;
	return los + " | " + hir;
}

function makeRotHigh(lo, hi, n)
{
	if (n === 0 || n === 32) throw Error("unsupported");
	if ((n & 0x20) !== 0)
	{
		n &= ~0x20;
		var t = hi;
		hi = lo;
		lo = t;
	}
	var his = hi + " << " + n;
	var lor = lo + " >>> " + (32 - n);
	return his + " | " + lor;
}

function makeKeccak_f1600()
{
	var format = function(n)
	{
		return n < 10 ? "0"+n : ""+n;
	};
	
	var a = function(n, w)
	{
		return "a" + format(n) + (w !== 0?'h':'l');
	};
	
	var b = function(n, w)
	{
		return "b" + format(n) + (w !== 0?'h':'l');
	};
	
	var str = "";	
	str += "function keccak_f1600(outState, outOffset, outSize, inState)\n";
	str += "{\n";
	
	for (var i = 0; i < 25; ++i)
	{
		for (var w = 0; w <= 1; ++w)
		{
			str += "\tvar " + a(i,w) + " = inState["+(i<<1|w)+"]|0;\n";
		}
	}
	
	for (var j = 0; j < 5; ++j)
	{
		str += "\tvar ";
		for (var i = 0; i < 5; ++i)
		{
			if (i !== 0)
				str += ", ";
			str += b(j*5+i,0) + ", " + b(j*5+i,1);
		}
		str += ";\n";
	}
	
	str += "\tvar tl, th;\n";
	str += "\n";
	str += "\tfor (var r = 0; r < 48; r = (r+2)|0)\n";
	str += "\t{\n";
	
	
	// Theta
	str += "\t\t// Theta\n";
	for (var i = 0; i < 5; ++i)
	{
		for (var w = 0; w <= 1; ++w)
		{
			str += "\t\t" + b(i,w) + " = " + a(i,w) + " ^ " + a(i+5,w) + " ^ " + a(i+10,w) + " ^ " + a(i+15,w) + " ^ " + a(i+20,w) + ";\n";
		}
	}
	
	for (var i = 0; i < 5; ++i)
	{
		var i4 = (i + 4) % 5;
		var i1 = (i + 1) % 5;
		str += "\t\ttl = " + b(i4,0) + " ^ (" + b(i1,0) + " << 1 | " + b(i1,1) + " >>> 31);\n";
		str += "\t\tth = " + b(i4,1) + " ^ (" + b(i1,1) + " << 1 | " + b(i1,0) + " >>> 31);\n";

		for (var j = 0; j < 25; j = (j+5)|0)
		{
			str += "\t\t" + a((j+i),0) + " ^= tl;\n";
			str += "\t\t" + a((j+i),1) + " ^= th;\n";
		}
	}
	

	// Rho Pi
	str += "\n\t\t// Rho Pi\n";
	for (var w = 0; w <= 1; ++w)
	{
		str += "\t\t" + b(0,w) + " = " + a(0,w) + ";\n";
	}
	var opi = 1;
	for (var i = 0; i < 24; ++i)
	{
		var pi = Keccak_f1600_Pi[i];
		str += "\t\t" + b(pi,0) + " = " + makeRotLow(a(opi,0), a(opi,1), Keccak_f1600_Rho[i]) + ";\n";
		str += "\t\t" + b(pi,1) + " = " + makeRotHigh(a(opi,0), a(opi,1), Keccak_f1600_Rho[i]) + ";\n";
		opi = pi;
	}
	
	//  Chi
	str += "\n\t\t// Chi\n";
	for (var j = 0; j < 25; j += 5)
	{
		for (var i = 0; i < 5; ++i)
		{
			for (var w = 0; w <= 1; ++w)
			{
				str += "\t\t" + a(j+i,w) + " = " + b(j+i,w) + " ^ ~" + b(j+(i+1)%5,w) + " & " + b(j+(i+2)%5,w) + ";\n";
			}
		}
	}

	//  Iota
	str += "\n\t\t// Iota\n";
	for (var w = 0; w <= 1; ++w)
	{
		str += "\t\t" + a(0,w) + " ^= Keccak_f1600_RC[r|" + w + "];\n";
	}
	
	
	str += "\t}\n";
	
	for (var i = 0; i < 25; ++i)
	{
		if (i == 4 || i == 8)
		{
			str += "\tif (outSize == " + i*2 + ")\n\t\treturn;\n";
		}
		for (var w = 0; w <= 1; ++w)
		{
			str += "\toutState[outOffset|"+(i<<1|w)+"] = " + a(i,w) + ";\n";
		}
	}
	str += "}\n";
	
	return str;
}

console.log(makeKeccak_f1600());
