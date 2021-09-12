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

// callTracer is a full blown transaction tracer that extracts and reports all
// the internal calls made by a transaction, along with any useful information.
{
    // callstack is the current recursive call stack of the EVM execution.
    callstack: [{}],

        // descended tracks whether we've just descended from an outer transaction into
        // an inner call.
        descended: false,

    enter: function(log, db){
        // Capture any errors immediately
        var call = {
            type:    log.type,
            from:    toHex(log.from),
            to:      toHex(log.to),
            input:   toHex(log.input),
            gas:   '0x' + bigInt(log.gas).toString(16),
        };
        if (typeof log.value !== 'undefined'){
            call.value='0x' + bigInt(log.value).toString(16)
        }
        if(log.error){
            call.error = log.error
        }

    this.callstack.push(call);
    },
    exit: function(log, db){
      var op = log.type
        var call = this.callstack.pop();
        if(log.error){
            call.error = log.error
        }else{
            call.output = toHex(log.output)
        }
        call.gasUsed = '0x' + bigInt(log.gasUsed).toString(16)
        // Inject the call into the previous one
        var left = this.callstack.length;
        if (this.callstack[left-1].calls === undefined) {
            this.callstack[left-1].calls = [];
        }
        this.callstack[left-1].calls.push(call);
    },

    // fault is invoked when the actual execution of an opcode fails.
    fault: function(log, db) {
        // If the topmost call already reverted, don't handle the additional fault again
        if (this.callstack[this.callstack.length - 1].error !== undefined) {
            return;
        }
        // Pop off the just failed call
        var call = this.callstack.pop();
        call.error = log.getError();

        // Consume all available gas and clean any leftovers
        if (call.gas !== undefined) {
            call.gas = '0x' + bigInt(call.gas).toString(16);
            call.gasUsed = call.gas
        }
        delete call.gasIn; delete call.gasCost;
        delete call.outOff; delete call.outLen;

        // Flatten the failed call into its parent
        var left = this.callstack.length;
        if (left > 0) {
            if (this.callstack[left-1].calls === undefined) {
                this.callstack[left-1].calls = [];
            }
            this.callstack[left-1].calls.push(call);
            return;
        }
        // Last call failed too, leave it in the stack
        this.callstack.push(call);
    },

    // result is invoked when all the opcodes have been iterated over and returns
    // the final result of the tracing.
    result: function(ctx, db) {
        var result = {
            type:    ctx.type,
            from:    toHex(ctx.from),
            to:      toHex(ctx.to),
            value:   '0x' + ctx.value.toString(16),
            gas:     '0x' + bigInt(ctx.gas).toString(16),
            gasUsed: '0x' + bigInt(ctx.gasUsed).toString(16),
            input:   toHex(ctx.input),
            output:  toHex(ctx.output),
            time:    ctx.time,
        };
        if (this.callstack[0].calls !== undefined) {
            result.calls = this.callstack[0].calls;
        }
        if (this.callstack[0].error !== undefined) {
            result.error = this.callstack[0].error;
        } else if (ctx.error !== undefined) {
            result.error = ctx.error;
        }
        if (result.error !== undefined && (result.error !== "execution reverted" || result.output ==="0x")) {
            delete result.output;
        }
        return this.finalize(result);
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
                delete sorted[key];
            }
        }
        if (sorted.calls !== undefined) {
            for (var i=0; i<sorted.calls.length; i++) {
                sorted.calls[i] = this.finalize(sorted.calls[i]);
            }
        }
        return sorted;
    }
}
