"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.CharStreams = void 0;
const CodePointBuffer_1 = require("./CodePointBuffer");
const CodePointCharStream_1 = require("./CodePointCharStream");
const IntStream_1 = require("./IntStream");
// const DEFAULT_BUFFER_SIZE: number = 4096;
/** This class represents the primary interface for creating {@link CharStream}s
 *  from a variety of sources as of 4.7.  The motivation was to support
 *  Unicode code points > U+FFFF.  {@link ANTLRInputStream} and
 *  {@link ANTLRFileStream} are now deprecated in favor of the streams created
 *  by this interface.
 *
 *  DEPRECATED: {@code new ANTLRFileStream("myinputfile")}
 *  NEW:        {@code CharStreams.fromFileName("myinputfile")}
 *
 *  WARNING: If you use both the deprecated and the new streams, you will see
 *  a nontrivial performance degradation. This speed hit is because the
 *  {@link Lexer}'s internal code goes from a monomorphic to megamorphic
 *  dynamic dispatch to get characters from the input stream. Java's
 *  on-the-fly compiler (JIT) is unable to perform the same optimizations
 *  so stick with either the old or the new streams, if performance is
 *  a primary concern. See the extreme debugging and spelunking
 *  needed to identify this issue in our timing rig:
 *
 *      https://github.com/antlr/antlr4/pull/1781
 *
 *  The ANTLR character streams still buffer all the input when you create
 *  the stream, as they have done for ~20 years. If you need unbuffered
 *  access, please note that it becomes challenging to create
 *  parse trees. The parse tree has to point to tokens which will either
 *  point into a stale location in an unbuffered stream or you have to copy
 *  the characters out of the buffer into the token. That defeats the purpose
 *  of unbuffered input. Per the ANTLR book, unbuffered streams are primarily
 *  useful for processing infinite streams *during the parse.*
 *
 *  The new streams also use 8-bit buffers when possible so this new
 *  interface supports character streams that use half as much memory
 *  as the old {@link ANTLRFileStream}, which assumed 16-bit characters.
 *
 *  A big shout out to Ben Hamilton (github bhamiltoncx) for his superhuman
 *  efforts across all targets to get true Unicode 3.1 support for U+10FFFF.
 *
 *  @since 4.7
 */
var CharStreams;
(function (CharStreams) {
    // /**
    //  * Creates a {@link CharStream} given a path to a UTF-8
    //  * encoded file on disk.
    //  *
    //  * Reads the entire contents of the file into the result before returning.
    //  */
    // export function fromFile(file: File): CharStream;
    // export function fromFile(file: File, charset: Charset): CharStream;
    // export function fromFile(file: File, charset?: Charset): CharStream {
    // 	if (charset === undefined) {
    // 		charset = Charset.forName("UTF-8");
    // 	}
    function fromString(s, sourceName) {
        if (sourceName === undefined || sourceName.length === 0) {
            sourceName = IntStream_1.IntStream.UNKNOWN_SOURCE_NAME;
        }
        // Initial guess assumes no code points > U+FFFF: one code
        // point for each code unit in the string
        let codePointBufferBuilder = CodePointBuffer_1.CodePointBuffer.builder(s.length);
        // TODO: CharBuffer.wrap(String) rightfully returns a read-only buffer
        // which doesn't expose its array, so we make a copy.
        let cb = new Uint16Array(s.length);
        for (let i = 0; i < s.length; i++) {
            cb[i] = s.charCodeAt(i);
        }
        codePointBufferBuilder.append(cb);
        return CodePointCharStream_1.CodePointCharStream.fromBuffer(codePointBufferBuilder.build(), sourceName);
    }
    CharStreams.fromString = fromString;
    // export function bufferFromChannel(
    // 	channel: ReadableByteChannel,
    // 	charset: Charset,
    // 	bufferSize: number,
    // 	decodingErrorAction: CodingErrorAction,
    // 	inputSize: number): CodePointBuffer {
    // 	try {
    // 		let utf8BytesIn: Uint8Array = new Uint8Array(bufferSize);
    // 		let utf16CodeUnitsOut: Uint16Array = new Uint16Array(bufferSize);
    // 		if (inputSize === -1) {
    // 			inputSize = bufferSize;
    // 		} else if (inputSize > Integer.MAX_VALUE) {
    // 			// ByteBuffer et al don't support long sizes
    // 			throw new RangeError(`inputSize ${inputSize} larger than max ${Integer.MAX_VALUE}`);
    // 		}
    // 		let codePointBufferBuilder: CodePointBuffer.Builder = CodePointBuffer.builder(inputSize);
    // 		let decoder: CharsetDecoder = charset
    // 				.newDecoder()
    // 				.onMalformedInput(decodingErrorAction)
    // 				.onUnmappableCharacter(decodingErrorAction);
    // 		let endOfInput: boolean = false;
    // 		while (!endOfInput) {
    // 			let bytesRead: number = channel.read(utf8BytesIn);
    // 			endOfInput = (bytesRead === -1);
    // 			utf8BytesIn.flip();
    // 			let result: CoderResult = decoder.decode(
    // 				utf8BytesIn,
    // 				utf16CodeUnitsOut,
    // 				endOfInput);
    // 			if (result.isError() && decodingErrorAction === CodingErrorAction.REPORT) {
    // 				result.throwException();
    // 			}
    // 			utf16CodeUnitsOut.flip();
    // 			codePointBufferBuilder.append(utf16CodeUnitsOut);
    // 			utf8BytesIn.compact();
    // 			utf16CodeUnitsOut.compact();
    // 		}
    // 		// Handle any bytes at the end of the file which need to
    // 		// be represented as errors or substitution characters.
    // 		let flushResult: CoderResult = decoder.flush(utf16CodeUnitsOut);
    // 		if (flushResult.isError() && decodingErrorAction === CodingErrorAction.REPORT) {
    // 			flushResult.throwException();
    // 		}
    // 		utf16CodeUnitsOut.flip();
    // 		codePointBufferBuilder.append(utf16CodeUnitsOut);
    // 		return codePointBufferBuilder.build();
    // 	}
    // 	finally {
    // 		channel.close();
    // 	}
    // }
})(CharStreams = exports.CharStreams || (exports.CharStreams = {}));
//# sourceMappingURL=CharStreams.js.map