// callTracer is a full blown transaction tracer that extracts and reports all
// the internal calls made by a transaction, along with any useful information.
{
	// invocations is a nested data structure containing all the internal contract
	// calls executed by a transaction.
	invocations: [],

	// callstack is the current recursive call stack of the EVM execution.
	callstack: [],

	// descended tracks whether we've just descended from an outer transaction into
	// an inner call.
	descended: false,

	// step is invoked for every opcode that the VM executes.
	step: function(log, db) {
		// We only care about system opcodes, faster if we pre-check once
		var syscall = (log.op.toNumber() & 0xf0) == 0xf0;
		if (syscall) {
			var op = log.op.toString();
		}
		// If a new method invocation is being done, add to the call stack
		if (syscall && op != "RETURN" && op != "REVERT") {
			// Skip any pre-compile invocations, those are just fancy opcodes
			var to = toAddress(log.stack.peek(1).Bytes());
			if (isPrecompiled(to)) {
				return
			}
			// Otherwise gather some internal call details
			var off = (op == 'DELEGATECALL' ? 0 : 1);

			var inOff = log.stack.peek(2 + off).Int64();
			var inEnd = inOff + log.stack.peek(3 + off).Int64();

			// Assemble the internal call report and store for completion
			var call = {
				type:    op,
				from:    log.account,
				to:      to,
				input:   toHex(log.memory.slice(inOff, inEnd)),
				gasDiff: log.gas - log.cost,
				outOff:  log.stack.peek(4 + off).Int64(),
				outLen:  log.stack.peek(5 + off).Int64()
			};
			if (op != 'DELEGATECALL') {
				call.value = '0x' + log.stack.peek(2).Text(16);
			}
			this.callstack.push(call);
			this.descended = true
			return;
		}
		// If we've just descended into an inner call, retrieve it's true allowance. We
		// need to extract if from within the call as there may be funky gas dynamics
		// with regard to requested and actually given gas (2300 stipend, 63/64 rule).
		if (this.descended) {
			if (log.depth > this.callstack.length) {
				this.callstack[this.callstack.length - 1].gas = log.gas;
			} else {
				this.callstack[this.callstack.length - 1].gas = 2300; // TODO (karalabe): Erm...
			}
			this.descended = false;
		}
		// If an existing call is returning, pop off the call stack
		if (syscall && op == 'REVERT') {
			call.revert = true;
			return;
		}
		if (log.depth == this.callstack.length) {
			// Pop off the last call and get the execution results
			var call = this.callstack.pop();

			call.gasUsed = '0x' + big.NewInt(call.gas - log.gas + call.gasDiff).Text(16);
			call.gas     = '0x' + big.NewInt(call.gas).Text(16);
			delete call.gasDiff;

			call.output = toHex(log.memory.slice(call.outOff, call.outOff + call.outLen));
			delete call.outOff;
			delete call.outLen;

			// Append to the invocation list if topmost
			var left = this.callstack.length;

			if (left == 0) {
				this.invocations.push(call);
				return
			}
			// Apparently it was a nested call, inject into the previous one
			if (this.callstack[left-1].calls === undefined) {
				this.callstack[left-1].calls = [];
			}
			this.callstack[left-1].calls.push(call);
		}
	},

	// result is invoked when all the opcodes have been iterated over and returns
	// the final result of the tracing.
	result: function(ctx) {
		var result = {
			type:    ctx.type,
			from:    toAddress(ctx.from),
			to:      toAddress(ctx.to),
			value:   '0x' + ctx.value.Text(16),
			gas:     '0x' + big.NewInt(ctx.gas).Text(16),
			gasUsed: '0x' + big.NewInt(ctx.gasUsed).Text(16),
			input:   toHex(ctx.input),
			output:  toHex(ctx.output),
			time:    ctx.time,
		};
		if (this.invocations.length > 0) {
			result.calls = this.invocations;
		}
		if (ctx.error !== undefined) {
			result.error = ctx.error;
		}
		return result;
	}
}
