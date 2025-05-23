"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.CodePointBuffer = void 0;
const assert = require("assert");
const Character = require("./misc/Character");
/**
 * Wrapper for `Uint8Array` / `Uint16Array` / `Int32Array`.
 */
class CodePointBuffer {
    constructor(buffer, size) {
        this.buffer = buffer;
        this._position = 0;
        this._size = size;
    }
    static withArray(buffer) {
        return new CodePointBuffer(buffer, buffer.length);
    }
    get position() {
        return this._position;
    }
    set position(newPosition) {
        if (newPosition < 0 || newPosition > this._size) {
            throw new RangeError();
        }
        this._position = newPosition;
    }
    get remaining() {
        return this._size - this.position;
    }
    get(offset) {
        return this.buffer[offset];
    }
    array() {
        return this.buffer.slice(0, this._size);
    }
    static builder(initialBufferSize) {
        return new CodePointBuffer.Builder(initialBufferSize);
    }
}
exports.CodePointBuffer = CodePointBuffer;
(function (CodePointBuffer) {
    let Type;
    (function (Type) {
        Type[Type["BYTE"] = 0] = "BYTE";
        Type[Type["CHAR"] = 1] = "CHAR";
        Type[Type["INT"] = 2] = "INT";
    })(Type || (Type = {}));
    class Builder {
        constructor(initialBufferSize) {
            this.type = 0 /* BYTE */;
            this.buffer = new Uint8Array(initialBufferSize);
            this.prevHighSurrogate = -1;
            this.position = 0;
        }
        build() {
            return new CodePointBuffer(this.buffer, this.position);
        }
        static roundUpToNextPowerOfTwo(i) {
            let nextPowerOfTwo = 32 - Math.clz32(i - 1);
            return Math.pow(2, nextPowerOfTwo);
        }
        ensureRemaining(remainingNeeded) {
            switch (this.type) {
                case 0 /* BYTE */:
                    if (this.buffer.length - this.position < remainingNeeded) {
                        let newCapacity = Builder.roundUpToNextPowerOfTwo(this.buffer.length + remainingNeeded);
                        let newBuffer = new Uint8Array(newCapacity);
                        newBuffer.set(this.buffer.subarray(0, this.position), 0);
                        this.buffer = newBuffer;
                    }
                    break;
                case 1 /* CHAR */:
                    if (this.buffer.length - this.position < remainingNeeded) {
                        let newCapacity = Builder.roundUpToNextPowerOfTwo(this.buffer.length + remainingNeeded);
                        let newBuffer = new Uint16Array(newCapacity);
                        newBuffer.set(this.buffer.subarray(0, this.position), 0);
                        this.buffer = newBuffer;
                    }
                    break;
                case 2 /* INT */:
                    if (this.buffer.length - this.position < remainingNeeded) {
                        let newCapacity = Builder.roundUpToNextPowerOfTwo(this.buffer.length + remainingNeeded);
                        let newBuffer = new Int32Array(newCapacity);
                        newBuffer.set(this.buffer.subarray(0, this.position), 0);
                        this.buffer = newBuffer;
                    }
                    break;
            }
        }
        append(utf16In) {
            this.ensureRemaining(utf16In.length);
            this.appendArray(utf16In);
        }
        appendArray(utf16In) {
            switch (this.type) {
                case 0 /* BYTE */:
                    this.appendArrayByte(utf16In);
                    break;
                case 1 /* CHAR */:
                    this.appendArrayChar(utf16In);
                    break;
                case 2 /* INT */:
                    this.appendArrayInt(utf16In);
                    break;
            }
        }
        appendArrayByte(utf16In) {
            assert(this.prevHighSurrogate === -1);
            let input = utf16In;
            let inOffset = 0;
            let inLimit = utf16In.length;
            let outByte = this.buffer;
            let outOffset = this.position;
            while (inOffset < inLimit) {
                let c = input[inOffset];
                if (c <= 0xFF) {
                    outByte[outOffset] = c;
                }
                else {
                    utf16In = utf16In.subarray(inOffset, inLimit);
                    this.position = outOffset;
                    if (!Character.isHighSurrogate(c)) {
                        this.byteToCharBuffer(utf16In.length);
                        this.appendArrayChar(utf16In);
                        return;
                    }
                    else {
                        this.byteToIntBuffer(utf16In.length);
                        this.appendArrayInt(utf16In);
                        return;
                    }
                }
                inOffset++;
                outOffset++;
            }
            this.position = outOffset;
        }
        appendArrayChar(utf16In) {
            assert(this.prevHighSurrogate === -1);
            let input = utf16In;
            let inOffset = 0;
            let inLimit = utf16In.length;
            let outChar = this.buffer;
            let outOffset = this.position;
            while (inOffset < inLimit) {
                let c = input[inOffset];
                if (!Character.isHighSurrogate(c)) {
                    outChar[outOffset] = c;
                }
                else {
                    utf16In = utf16In.subarray(inOffset, inLimit);
                    this.position = outOffset;
                    this.charToIntBuffer(utf16In.length);
                    this.appendArrayInt(utf16In);
                    return;
                }
                inOffset++;
                outOffset++;
            }
            this.position = outOffset;
        }
        appendArrayInt(utf16In) {
            let input = utf16In;
            let inOffset = 0;
            let inLimit = utf16In.length;
            let outInt = this.buffer;
            let outOffset = this.position;
            while (inOffset < inLimit) {
                let c = input[inOffset];
                inOffset++;
                if (this.prevHighSurrogate !== -1) {
                    if (Character.isLowSurrogate(c)) {
                        outInt[outOffset] = String.fromCharCode(this.prevHighSurrogate, c).codePointAt(0);
                        outOffset++;
                        this.prevHighSurrogate = -1;
                    }
                    else {
                        // Dangling high surrogate
                        outInt[outOffset] = this.prevHighSurrogate;
                        outOffset++;
                        if (Character.isHighSurrogate(c)) {
                            this.prevHighSurrogate = c;
                        }
                        else {
                            outInt[outOffset] = c;
                            outOffset++;
                            this.prevHighSurrogate = -1;
                        }
                    }
                }
                else if (Character.isHighSurrogate(c)) {
                    this.prevHighSurrogate = c;
                }
                else {
                    outInt[outOffset] = c;
                    outOffset++;
                }
            }
            if (this.prevHighSurrogate !== -1) {
                // Dangling high surrogate
                outInt[outOffset] = this.prevHighSurrogate;
                outOffset++;
            }
            this.position = outOffset;
        }
        byteToCharBuffer(toAppend) {
            // CharBuffers hold twice as much per unit as ByteBuffers, so start with half the capacity.
            let newBuffer = new Uint16Array(Math.max(this.position + toAppend, this.buffer.length >> 1));
            newBuffer.set(this.buffer.subarray(0, this.position), 0);
            this.type = 1 /* CHAR */;
            this.buffer = newBuffer;
        }
        byteToIntBuffer(toAppend) {
            // IntBuffers hold four times as much per unit as ByteBuffers, so start with one quarter the capacity.
            let newBuffer = new Int32Array(Math.max(this.position + toAppend, this.buffer.length >> 2));
            newBuffer.set(this.buffer.subarray(0, this.position), 0);
            this.type = 2 /* INT */;
            this.buffer = newBuffer;
        }
        charToIntBuffer(toAppend) {
            // IntBuffers hold two times as much per unit as ByteBuffers, so start with one half the capacity.
            let newBuffer = new Int32Array(Math.max(this.position + toAppend, this.buffer.length >> 1));
            newBuffer.set(this.buffer.subarray(0, this.position), 0);
            this.type = 2 /* INT */;
            this.buffer = newBuffer;
        }
    }
    CodePointBuffer.Builder = Builder;
})(CodePointBuffer = exports.CodePointBuffer || (exports.CodePointBuffer = {}));
//# sourceMappingURL=CodePointBuffer.js.map