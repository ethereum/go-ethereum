// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.


// callFrameTracer uses the new call frame tracing methods to report useful information
// about internal messages of a transaction.
{
	callstack: [{}],
	fault: function(log, db) {},
	result: function(ctx, db) {
		// Prepare outer message info
		var result = {
			type:    ctx.type,
			from:    toHex(ctx.from),
			to:      toHex(ctx.to),
			value:   '0x' + ctx.value.toString(16),
			gas:     '0x' + bigInt(ctx.gas).toString(16),
			gasUsed: '0x' + bigInt(ctx.gasUsed).toString(16),
			input:   toHex(ctx.input),
			output:  toHex(ctx.output),
		}
		if (this.callstack[0].calls !== undefined) {
			result.calls = this.callstack[0].calls
		}
		if (this.callstack[0].error !== undefined) {
			result.error = this.callstack[0].error
		} else if (ctx.error !== undefined) {
			result.error = ctx.error
		}
		if (result.error !== undefined && (result.error !== "execution reverted" || result.output ==="0x")) {
			delete result.output
		}

		return this.finalize(result)
	},
	enter: function(frame) {
		var call = {
			type: frame.getType(),
			from: toHex(frame.getFrom()),
			to: toHex(frame.getTo()),
			input: toHex(frame.getInput()),
			gas: '0x' + bigInt(frame.getGas()).toString('16'),
		}
		if (frame.getValue() !== undefined){
			call.value='0x' + bigInt(frame.getValue()).toString(16)
		}
		this.callstack.push(call)
	},
	exit: function(frameResult) {
		var len = this.callstack.length
		if (len > 1) {
			var call = this.callstack.pop()
			call.gasUsed = '0x' + bigInt(frameResult.getGasUsed()).toString('16')
			var error = frameResult.getError()
			if (error === undefined) {
				call.output = toHex(frameResult.getOutput())
			} else {
				call.error = error
				if (call.type === 'CREATE' || call.type === 'CREATE2') {
					delete call.to
				}
			}
			len -= 1
			if (this.callstack[len-1].calls === undefined) {
				this.callstack[len-1].calls = []
			}
			this.callstack[len-1].calls.push(call)
		}
	},
	// finalize recreates a call object using the final desired field oder for json
	// serialization. This is a nicety feature to pass meaningfully ordered results
	// to users who don't interpret it, just display it.
	finalize: function(call) {
		var sorted = {
			type:    call.type,
			from:    call.from,
			to:      call.to,
			value:   call.value,
			gas:     call.gas,
			gasUsed: call.gasUsed,
			input:   call.input,
			output:  call.output,
			error:   call.error,
			time:    call.time,
			calls:   call.calls,
		}
		for (var key in sorted) {
			if (sorted[key] === undefined) {
				delete sorted[key]
			}
		}
		if (sorted.calls !== undefined) {
			for (var i=0; i<sorted.calls.length; i++) {
				sorted.calls[i] = this.finalize(sorted.calls[i])
			}
		}
		return sorted
	}
}
