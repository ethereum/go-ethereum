These files examplify a transition where a transaction (excuted on block 5) requests
the blockhash for block `4`, but where the hash for that block is missing. 
It's expected that executing these should cause `exit` with errorcode `4`.
