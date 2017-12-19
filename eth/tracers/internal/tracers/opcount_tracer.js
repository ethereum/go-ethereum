// opcountTracer is a sample tracer that just counts the number of instructions
// executed by the EVM before the transaction terminated.
{
	// count tracks the number of EVM instructions executed.
	count: 0,

	// step is invoked for every opcode that the VM executes.
	step: function(log, db) { this.count++ },

	// fault is invoked when the actual execution of an opcode fails.
	fault: function(log, db) { },

	// result is invoked when all the opcodes have been iterated over and returns
	// the final result of the tracing.
	result: function(ctx, db) { return this.count }
}
