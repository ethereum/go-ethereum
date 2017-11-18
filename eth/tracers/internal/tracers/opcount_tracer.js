// opcountTracer is a sample tracer that just counts the number of instructions
// executed by the EVM before the transaction terminated.
{
	// count tracks the number of EVM instructions executed.
	count: 0,

	// step is invoked for every opcode that the VM executes.
	step: function(log, db) { this.count++ },

	// result is invoked when all the opcodes have been iterated over and returns
	// the final result of the tracing.
	result: function(ctx) { return this.count }
}
