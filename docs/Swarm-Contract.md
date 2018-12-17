This one contract regulates the incentive structure of Swarm.

The corresponding solidity code can be browsed [here](https://github.com/ethersphere/go-ethereum/blob/bzz/bzz/bzzcontract/swarm.sol).

# Methods

## Sign up as a node

Pay a deposit in Ether and register public key. Comes with an accessor for checking that a node is signed up.

## Demand penalty for loss of chunk

Present a signed receipt by a signed up node and a deposit covering the upload of a chunk. After a given deadline, the signer node's deposit is taken and the presenting node's deposit refunded, unless the chunk is presented. Comes with an accessor for checking that a given chunk has been reported lost, so that holders of receipts by other swarm nodes can punish them as well for losing the chunk, which, in turn, incentivizes whoever holds the chunk to present it.

## Present chunk to avoid penalty

No penalty is paid for lost chunks, if chunk is presented within the deadline. The cost of uploading the chunk is compensated exactly from the demand's deposit, with the remainder refunded. Comes with an accessor for checking that a given node is liable for penalty, so the node is notified to present the chunk in a timely fashion.

# Price considerations

For the price of accepting a chunk for storing, see [Incentives](https://github.com/ethersphere/swarm/blob/master/doc/incentives.md)

This price should be proportional to the sign-up deposit of the swarm node.

The deposit for compensating the swarm node for uploading the chunk into the block chain should be substantially higher (e.g. a small integer multiple) of the corresponding upload measured with the gas price used to upload the demand to prevent DoS attacks.

# Termination

Users of Swarm should be able to count on the loss of deposit as a disincentive, so it should not be refunded before the term of Swarm membership expires. If penalites were paid out as compensation to holders of receipts of lost chunks, it would provide an avenue of early exit for a Swarm member by "losing" chunks deposited by colluding users. Since users of Swarm are interested in their information being reliably stored, their primary incentive for keeping the receipts is to keep the Swarm motivated, not the potential compensation.

# Receipt circulation

End-users of Swarm keeping important information in it are obviously interested in keeping as many receipts of it as possible available for "litigation". The storage space required for storing a receipt is a sizable fraction of that used for storing the information itself, so end users can reduce their storage requirement further by storing the receipts in Swarm as well. Doing this recursively would result in end users only having to store a single receipt, yet being
able to penalize quite a few Swarm nodes, in case only a small part of their stored information
is lost.

Swarm nodes that use the rest of Swarm as a backup may want to propagate the receipts in the opposite direction of storage requests, so that the cost of storing receipts is eventually paid by the end user either in the form of allocated storage space or as a direct payment to Swarm.