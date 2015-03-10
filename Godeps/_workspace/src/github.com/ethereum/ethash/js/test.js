// test.js
// Tim Hughes <tim@twistedfury.com>

/*jslint node: true, shadow:true */
"use strict";

var ethash = require('./ethash');
var util = require('./util');
var Keccak = require('./keccak');

// sanity check hash functions
var src = util.stringToBytes("");
if (util.bytesToHexString(new Keccak().digest(32, src)) != "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470") throw Error("Keccak-256 failed");
if (util.bytesToHexString(new Keccak().digest(64, src)) != "0eab42de4c3ceb9235fc91acffe746b29c29a8c366b7c60e4e67c466f36a4304c00fa9caf9d87976ba469bcbe06713b435f091ef2769fb160cdab33d3670680e") throw Error("Keccak-512 failed");

src = new Uint32Array(src.buffer);
var dst = new Uint32Array(8);
new Keccak().digestWords(dst, 0, dst.length, src, 0, src.length);
if (util.wordsToHexString(dst) != "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470") throw Error("Keccak-256 Fast failed");

var dst = new Uint32Array(16);
new Keccak().digestWords(dst, 0, dst.length, src, 0, src.length);
if (util.wordsToHexString(dst) != "0eab42de4c3ceb9235fc91acffe746b29c29a8c366b7c60e4e67c466f36a4304c00fa9caf9d87976ba469bcbe06713b435f091ef2769fb160cdab33d3670680e") throw Error("Keccak-512 Fast failed");


// init params
var ethashParams = ethash.defaultParams();
//ethashParams.cacheRounds = 0;

// create hasher
var seed = util.hexStringToBytes("9410b944535a83d9adf6bbdcc80e051f30676173c16ca0d32d6f1263fc246466")
var startTime = new Date().getTime();
var hasher = new ethash.Ethash(ethashParams, seed);
console.log('Ethash startup took: '+(new Date().getTime() - startTime) + "ms");
console.log('Ethash cache hash: ' + util.bytesToHexString(hasher.cacheDigest()));

var testHexString = "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470";
if (testHexString != util.bytesToHexString(util.hexStringToBytes(testHexString)))
	throw Error("bytesToHexString or hexStringToBytes broken");

		
var header = util.hexStringToBytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470");
var nonce = util.hexStringToBytes("0000000000000000");
var hash;

startTime = new Date().getTime();
var trials = 10;
for (var i = 0; i < trials; ++i)
{
	hash = hasher.hash(header, nonce);
}
console.log("Light client hashes averaged: " + (new Date().getTime() - startTime)/trials + "ms");
console.log("Hash = " + util.bytesToHexString(hash));
