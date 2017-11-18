// noopTracer is just the barebone boilerplate code required from a JavaScript
// object to be usable as a transaction tracer.
{
	// step is invoked for every opcode that the VM executes.
	step: function(log, db) { },

	// result is invoked when all the opcodes have been iterated over and returns
	// the final result of the tracing.
	result: function(ctx) { }
}
