/**

 4bytes searches for 4byte-identifiers, and collects them for post-processing.
 It collects the methods identifiers along with the size of the supplied data, so that
 a reversed signature can be matched against the size of the data.

Example:

    > debug.traceTransaction( "0x214e597e35da083692f5386141e69f47e973b2c56e7a8073b1ea08fd7571e9de", {tracer: "4bytes"})
    {
      0x27dc297e-128: 1,
      0x38cc4831-0: 2,
      0x524f3889-96: 1,
      0xadf59f99-288: 1,
      0xc281d19e-0: 1
    }

@author MH Swende
 **/
{
	ids : {},

	// callType returns 'false' for non-calls, or the
	// peek-index for the first param after 'value', i.e. meminstart
    callType : function(opstr){
        switch(opstr){
            case "CALL":
			case "CALLCODE":
            	//g,a,v, memin, meminsz, memout, memoutsz
            	return 3 //stack ptr to memin
            case "DELEGATECALL":
            case "STATICCALL":
                //g,a,memin, meminsz, memout, memoutsz
                return 2;//stack ptr to memin
        }
        return false;
    },

	// store save the given indentifier and datasize
	store: function(id, size){
		var key = ""+id+"-"+size;
		this.ids[key] = this.ids[key]+1 || 1;
	},

	// step is invoked for every opcode that the VM executes.
	step: function(log, db) {
        var ct = this.callType(log.op.toString());
        if (!ct) return;
        // Skip any pre-compile invocations, those are just fancy opcodes
        if (isPrecompiled(toAddress(log.stack.peek(1).Bytes()))) {
            return
        }
        // Gather internal call details
        var inSz = log.stack.peek(ct + 1).Int64();
        if (inSz >= 4) {
            var inOff = log.stack.peek(ct).Int64();
            this.store(toHex(log.memory.slice(inOff, inOff + 4)), inSz-4);
        }
	},

	// result is invoked when all the opcodes have been iterated over and returns
	// the final result of the tracing.
	result: function(ctx) {
		//Save the outer calldata also
		if(ctx.input.length > 4){
			this.store(toHex(ctx.input.slice(0,4)),ctx.input.length-4 )
		}
		return this.ids;
	},
}
