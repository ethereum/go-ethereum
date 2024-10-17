// Copyright 2017 The go-ethereum Authors
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

// 4byteTracer searches for 4byte-identifiers, and collects them for post-processing.
// It collects the methods identifiers along with the size of the supplied data, so
// a reversed signature can be matched against the size of the data.
//
// Example:
//   > debug.traceTransaction( "0x214e597e35da083692f5386141e69f47e973b2c56e7a8073b1ea08fd7571e9de", {tracer: "4byteTracer"})
//   {
//     0x27dc297e-128: 1,
//     0x38cc4831-0: 2,
//     0x524f3889-96: 1,
//     0xadf59f99-288: 1,
//     0xc281d19e-0: 1
//   }
{
	// ids aggregates the 4byte ids found.
	ids : {},

	// store save the given indentifier and datasize.
	store: function(id, size){
		var key = "" + toHex(id) + "-" + size;
		this.ids[key] = this.ids[key] + 1 || 1;
	},

	enter: function(frame) {
		// Skip any pre-compile invocations, those are just fancy opcodes
		if (isPrecompiled(frame.getTo())) {
			return;
		}
		var input = frame.getInput()
		if (input.length >= 4) {
			this.store(slice(input, 0, 4), input.length - 4);
		}
	},

	exit: function(frameResult) {},

	// fault is invoked when the actual execution of an opcode fails.
	fault: function(log, db) {},

	// result is invoked when all the opcodes have been iterated over and returns
	// the final result of the tracing.
	result: function(ctx) {
		// Save the outer calldata also
		if (ctx.input.length >= 4) {
			this.store(slice(ctx.input, 0, 4), ctx.input.length-4)
		}
		return this.ids;
	},
}
