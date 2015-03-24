// ethash.js
// Tim Hughes <tim@twistedfury.com>
// Revision 19

/*jslint node: true, shadow:true */
"use strict";

var Keccak = require('./keccak');
var util = require('./util');

// 32-bit unsigned modulo
function mod32(x, n)
{
	return (x>>>0) % (n>>>0);
}

function fnv(x, y)
{
	// js integer multiply by 0x01000193 will lose precision
	return ((x*0x01000000 | 0) + (x*0x193 | 0)) ^ y;	
}

function computeCache(params, seedWords)
{
	var cache = new Uint32Array(params.cacheSize >> 2);
	var cacheNodeCount = params.cacheSize >> 6;

	// Initialize cache
	var keccak = new Keccak();
	keccak.digestWords(cache, 0, 16, seedWords, 0, seedWords.length);
	for (var n = 1; n < cacheNodeCount; ++n)
	{
		keccak.digestWords(cache, n<<4, 16, cache, (n-1)<<4, 16);
	}
	
	var tmp = new Uint32Array(16);
	
	// Do randmemohash passes
	for (var r = 0; r < params.cacheRounds; ++r)
	{
		for (var n = 0; n < cacheNodeCount; ++n)
		{
			var p0 = mod32(n + cacheNodeCount - 1, cacheNodeCount) << 4;
			var p1 = mod32(cache[n<<4|0], cacheNodeCount) << 4;
			
			for (var w = 0; w < 16; w=(w+1)|0)
			{
				tmp[w] = cache[p0 | w] ^ cache[p1 | w];
			}
			
			keccak.digestWords(cache, n<<4, 16, tmp, 0, tmp.length);
		}
	}	
	return cache;
}

function computeDagNode(o_node, params, cache, keccak, nodeIndex)
{
	var cacheNodeCount = params.cacheSize >> 6;
	var dagParents = params.dagParents;
	
	var c = (nodeIndex % cacheNodeCount) << 4;
	var mix = o_node;
	for (var w = 0; w < 16; ++w)
	{
		mix[w] = cache[c|w];
	}
	mix[0] ^= nodeIndex;
	keccak.digestWords(mix, 0, 16, mix, 0, 16);
	
	for (var p = 0; p < dagParents; ++p)
	{
		// compute cache node (word) index
		c = mod32(fnv(nodeIndex ^ p, mix[p&15]), cacheNodeCount) << 4;
		
		for (var w = 0; w < 16; ++w)
		{
			mix[w] = fnv(mix[w], cache[c|w]);
		}
	}
	
	keccak.digestWords(mix, 0, 16, mix, 0, 16);
}

function computeHashInner(mix, params, cache, keccak, tempNode)
{
	var mixParents = params.mixParents|0;
	var mixWordCount = params.mixSize >> 2;
	var mixNodeCount = mixWordCount >> 4;
	var dagPageCount = (params.dagSize / params.mixSize) >> 0;
	
	// grab initial first word
	var s0 = mix[0];
	
	// initialise mix from initial 64 bytes
	for (var w = 16; w < mixWordCount; ++w)
	{
		mix[w] = mix[w & 15];
	}
	
	for (var a = 0; a < mixParents; ++a)
	{
		var p = mod32(fnv(s0 ^ a, mix[a & (mixWordCount-1)]), dagPageCount);
		var d = (p * mixNodeCount)|0;
		
		for (var n = 0, w = 0; n < mixNodeCount; ++n, w += 16)
		{
			computeDagNode(tempNode, params, cache, keccak, (d + n)|0);
			
			for (var v = 0; v < 16; ++v)
			{
				mix[w|v] = fnv(mix[w|v], tempNode[v]);
			}
		}
	}
}

function convertSeed(seed)
{
	// todo, reconcile with spec, byte ordering?
	// todo, big-endian conversion
	var newSeed = util.toWords(seed);
	if (newSeed === null)
		throw Error("Invalid seed '" + seed + "'");
	return newSeed;
}

exports.defaultParams = function()
{
	return {
		cacheSize: 1048384,
		cacheRounds: 3,
		dagSize: 1073739904,
		dagParents: 256,
		mixSize: 128,
		mixParents: 64,
	};
};

exports.Ethash = function(params, seed)
{
	// precompute cache and related values
	seed = convertSeed(seed);
	var cache = computeCache(params, seed);
	
	// preallocate buffers/etc
	var initBuf = new ArrayBuffer(96);
	var initBytes = new Uint8Array(initBuf);
	var initWords = new Uint32Array(initBuf);
	var mixWords = new Uint32Array(params.mixSize / 4);
	var tempNode = new Uint32Array(16);
	var keccak = new Keccak();
	
	var retWords = new Uint32Array(8);
	var retBytes = new Uint8Array(retWords.buffer); // supposedly read-only
	
	this.hash = function(header, nonce)
	{
		// compute initial hash
		initBytes.set(header, 0);
		initBytes.set(nonce, 32);
		keccak.digestWords(initWords, 0, 16, initWords, 0, 8 + nonce.length/4);
		
		// compute mix
		for (var i = 0; i != 16; ++i)
		{
			mixWords[i] = initWords[i];
		}
		computeHashInner(mixWords, params, cache, keccak, tempNode);
		
		// compress mix and append to initWords
		for (var i = 0; i != mixWords.length; i += 4)
		{
			initWords[16 + i/4] = fnv(fnv(fnv(mixWords[i], mixWords[i+1]), mixWords[i+2]), mixWords[i+3]);
		}
			
		// final Keccak hashes
		keccak.digestWords(retWords, 0, 8, initWords, 0, 24); // Keccak-256(s + cmix)
		return retBytes;
	};
	
	this.cacheDigest = function()
	{
		return keccak.digest(32, new Uint8Array(cache.buffer));
	};
};




