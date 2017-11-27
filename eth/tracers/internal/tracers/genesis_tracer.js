/**
This tracer-script outputs sufficient information to create a local
execution of the transaction.

@author Martin Holst Swende
**/
{
    //The genesis that we're building
    prestate : null,

    lookupAccount: function(acc,db){
        var addr = toHex(acc);
        if(this.prestate[addr]) return;

        this.prestate[addr] =  {
            'balance': db.getBalance(acc),
            'nonce' : db.getNonce(acc),
            'code' : toHex(db.getCode(acc)),
            'storage' : {}
        };
    },

    lookupStorage: function(addr, index, db){
        //No overwrite
        if (!this.prestate[addr]['storage'][index]){
          this.prestate[addr]['storage'][index] = db.getState(addr,key);
        }
    },

    result: function(ctx,db) {
        // At this point, we need to deduct the 'value' from the
        // outer transaction, and move it back to the origin
        var from = toAddress(ctx.from);
        var to = toAddress(ctx.to);
        var v = ctx.value;
        this.lookupAccount(from, db);

        // Add back value to sender (bigints)
        var from_bal = this.prestate[toHex(from)]['balance'];
        var to_bal =  this.prestate[toHex(to)]['balance'];
        to_bal.Sub(to_bal, v);
        from_bal.Add(from_bal, v);

        // Set back nonce (uint64)
        this.prestate[toHex(from)]['nonce']--;

        //Note, the sender-account still may not be 100% correct,
        // since we haven't accounted for the gas.
        //Homestead
        config = {
            "eip150Block": 2000,
            "eip158Block": 2000,
            "eip155Block": 2000,
            "homesteadBlock": 0,
            "daoForkBlock": 0,
            "byzantiumBlock" : 2000,
        }
        var blnum = ctx['blocknumber']
        if (blnum > 2463000){
            // Tangerine: disable
            config["eip150Block"] = 0;
        }
        if (blnum > 2675000){
            // Spurious
            config["eip155Block"] = 0;
            config["eip158Block"] = 0;
        }
        if (blnum > 4370000){
            //Byzantium
            config["byzantiumBlock"] = 0;
        }
        genesis ={
            nonce:      "0x0000000000000000",
            difficulty: "0x020000",
            mixhash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
            coinbase:   "0x0000000000000000000000000000000000000000",
            timestamp:  "0x00",
            number:      "0x00",
            parentHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
            extraData:  "0x0000000000000000000000000000000000000000000000000000000000000000",
            gasLimit:    toHex(ctx['gas'].Bytes()),
            alloc: this.prestate,
            config: config,
        };

        return genesis;
    },

    step: function(log, db) {

        if (!this.prestate){
            // Add the current account
            this.prestate = {};
            // Balance will potentially be wrong here,
            // since this will include the value sent
            // along with the message
            // We fix that in 'result()'
            this.lookupAccount(log.account,db);
        }

        switch(log.op.toString()){
            case "EXTCODECOPY":
            case "EXTCODESIZE":
            case "BALANCE":
                var addr = log.peek(0).Text(16);
                this.lookupAccount(addr,db);
                break;
            case "CALL":
            case "CALLCODE":
            case "DELEGATECALL":
            case "STATICCALL":
                var addr = log.stack.peek(1).Text(16);
                this.lookupAccount(addr,db);
                break;
            case 'SSTORE':
            case 'SLOAD':
                var key = log.stack.peek(0).Text(16);
                this.lookupStorage(log.account.Text(16), key,db);
            break;
        }
      }
}
/** TODO

Missing BLOCKNUM




Somehow get 'origin'-account in there.
**/
