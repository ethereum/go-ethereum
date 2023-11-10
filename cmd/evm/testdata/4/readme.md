<<<<<<< HEAD
These files examplify a transition where a transaction (executed on block 5) requests
=======
These files exemplify a transition where a transaction (executed on block 5) requests
>>>>>>> bed84606583893fdb698cc1b5058cc47b4dbd837
the blockhash for block `4`, but where the hash for that block is missing. 
It's expected that executing these should cause `exit` with errorcode `4`.
