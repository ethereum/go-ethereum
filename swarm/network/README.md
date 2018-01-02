# Requirements

- eventual consistency: so each chunk historical should be syncable
- since the same chunk can and will arrive from many peers, (network traffic should be
optimised (only one transfer of data per chunk)
- explicit request deliveries should be prioritised higher than recent chunks received
during the ongoing session which in turn should be higher than historical chunks.
- insured chunks should get receipted for finger pointing litigation, the receipts storage
should be organised efficiently, upstream peer should also be able to find these
receipts for a deleted chunk easily to refute their challenge.
- syncing should be resilient to cut connections, metadata should be persisted that
keep track of syncing state across sessions.
- extra data structures to support syncing should be kept at minimum
- syncing is organized separately for chunk types (resource update v content chunk)


When two peers connect, the bidirectional protocol is a result of two identical
syncing protocols mirrored. So take one direction and call the two parties
upstream and downstream peer.

when a new chunk is stored, its hash is appended to a boundless stream maintained for
each kademlia bin. This state is always permanently recorded periodically possibly
using mutable resource update scheme.

At any point in time (with n chunks in total ) the swarm hash of this hash stream state is
well defined. Upstream peer pushes so that the reader reads sequential hashes and
periodically calculates the swarm hash of their stream.

peers syncronise by getting the chunks closer to the downstream peer than to the upstream one.
Consequently peers just sync all stored items for the kad bin the receiving peer falls into.
The special case of nearest neighbour sets is handled by the downstream peer
indicating they want to sync all kademlia bins with proximity equal to or higher
than their depth.

When peers connect upstream peer sends the latest sync state (item index/cursor length)
for each relevant bin downstream peer is interested in.
This sync state represents the initial state of a sync connection session.
Conversely downstream peers maintain the last state (swarm hash with length) which
the ranges of covered offsets.

Retrieval is dictated by downstream peers simply using the the chunker joiner to read certain offsets.

Syncing chunks created during the session by the upstream peer is called session Syncing
while syncing of earlier chunks is historical syncing.

Historical syncing is simply carried out by iteratively requesting ranges of hash offsets.
For simplicity we assume that the minimum unit requested is a chunk. If every 128 chunks  
is considered syncable one can use data chunk index instead of offset in byte length.
Note that data chunk here is a sequence of hashes above one ground level of stream of content.

Once the relevant chunk is retrieved, downstream peer looks up all hash segments in its localstore
and sends to the upstream peer a message with a 128-long bitvector (uint16) to indicate
missing chunks (e.g., for chunk `k`, hash with chunk internal index which case )
new items. In turn upstream peer sends the relevant chunk data alongside their index.

On sending chunks there is a priority queue system. If during looking up hashes in its localstore,
downstream peer hits on an open request then a retrieve request is sent immediately to the upstream peer indicating
that no extra round of checks is needed. If another peers syncer hits the same open request, it is slightly unsafe to not ask
that peer too: if the first one disconnects before delivering or fails to deliver and therefore gets
disconnected, we should still be able to continue with the other. The minimum redundant traffic coming from such simultaneous
eventualities should be sufficiently rare not to warrant more complex treatment.

Session syncing involves downstream peer to request a new state on a bin from upstream.
using the new state, the range (of chunks) between the previous state and the new one are retrieved
and chunks are requested identical to the historical case. After receiving all the missing chunks
from the new hashes, downstream peer will request a new range. If this happens before upstream peer updates a new state,
we say that session syncing is live or the two peers are in sync.

If there is no historical backlog, downstream peer is said to be fully synced with the upstream.
If a peer is fully synced with all its storer peers, it can advertise itself as globally fully synced.

For healthy operation, however, it is expected that the session is regularly in sync. If this is
not the case, that indicates that traffic during the session is continuously more than the peer can cope with and
downstream peer is effectively accumulating a historical backlog.

The downstream peer persists the record of the last synced offset. When the two peers disconnect and
reconnect syncing can start from there.
This situation however can also happen while historical syncing is not yet complete.
Effectively this means that the peer needs to persist a record of an arbitrary array of offset ranges covered.

The swarm  hash over the hash stream has many advantages. It implements a provable data transfer
and provide efficient storage for receipts in the form of inclusion proofs useable for finger pointing litigation.
When challenged on a missing chunk, upstream peer will provide an inclusion proof of a chunk hash against the state of the
sync stream. In order to be able to generate such an inclusion proof, upstream peer needs to store the hash index (counting consecutive hash-size segments) alongside the chunk data and preserve it even when the chunk data is deleted until the chunk is no longer insured.
if there is no valid insurance on the files the entry may be deleted.
As long as the chunk is preserved, no takeover proof will be needed since the node can respond to any challenge.
However, once the node needs to delete an insured chunk for capacity reasons, a receipt should be available to
refute the challenge by finger pointing to a downstream peer.
As part of the deletion protocol then, hashes of insured chunks to be removed are pushed to an infinite stream for every bin.

Downstream peer on the other hand needs to make sure that they can only be finger pointed about a chunk they did receive and store.
For this the check of a state should be exhaustive. If historical syncing finishes on one state, all hashes before are covered, no
surprises. In other words historical syncing this process is self verifying. With session syncing however, it is not enough to check going back covering the range from old offset to new. Continuity (i.e., that the new state is extension of the old) needs to be verified: after downstream peer reads the range into a buffer, it appends the buffer the last known state at the last known offset and verifies the resulting hash matches
the latest state. The same goes with intervals of historical syncing where the state used to verify the preceding interval is different from the one used to cover the current range. In these cases too, verification by append is needed for complete security for downstream peer.

Upstream peer signs the states, downstream peers can use as handover proofs.
Downstream  peers sign off on a state together with an initial offset.
latter needed for reasonable sized not ever growing syncer
possible is eachn chunk needs to be reentered if they remain insured

Once historical syncing is complete and the session does not lag, downstream peer only preserves the latest upstream state and store the signed version.

Upstream peer needs to keep the latest takeover states: each deleted chunk's hash should be covered by takeover proof of at least one peer. If historical syncing is complete, upstream peer typically will store only the latest takeover proof from downstream peer.
Crucially, the structure is totally independent of the number of peers in the bin, so it scales extremely well.

implementation

The simplest protocol just involves upstream peer to prefix the key with the kademlia proximity order (say 0-15 or 0-31)
and simply iterate on index per bin when syncing with a peer.

priority queues are used for sending chunks so that user triggered requests should be responded to first, session syncing second, and historical with lower priority.
The request on chunks remains implemented as a dataless entry in the memory store.
The lifecycle of this object should be more carefully thought through, ie., when it fails to retrieve it should be removed.



Model 1
The main appeal in this model is that downstream driven syncing falls back to the exact same retrieval mechanism as the one used when downloading a file. If the chunks of the hash stream (datachunks as well as intermediate chunks of the swarm tree above it) are themselves distributed by upstream peer in the normal way, then requesting them from swarm is viable.
This requires no extra implementation. However, it is unlikely this is feasible for live syncing since the chunks' delay to arrive at their destination has a lag exactly due to session syncronisation.
Note that if upstream peer handles chunks of the hash stream as normal chunks, there are issues. One is that some of these chunks will fall in the same bin as the one building leading to a situation where the hash stream grows even though there are no external chunks received. If upstream peer wishes to use finger pointing proofs, it has to either store these chunks themselves or insure them.


Model 2
In this model, when retrieving the hash stream from the state, requests are targeted to the upstream peer.
The simplest way to generate the right requests from a sync state is to have a  
peer specific dpa chunkstore for chunker join. This can be used by all chunk types and all bins.
All it does is when picking retrieve tasks off the chunk channel, it marks the chunk with a reference to the
upstream peer so that when netstore does not find it, the request is sent to the upstream peer (only).
(Normally the request would be routed based on its address).
We need to introduce a special field on the chunk to indicate that the data should be requested from a particular peer.
Upstream peer could still distribute these chunks used in the hash stream in swarm as usual, e.g., long non-synced early
history.
This model does not suffer from the availability lag of the first one, correctly puts the burden on upstream peer to preserve
chunks of the hash stream either in swarm or not.

Model 3
in another alternative upstream peer sends only the data level (the hash stream interval) not all the intermediate chunks. This saves on traffic and downstream peer can calculate the state by append to verify against the state (root hash).
This mode of operation is anyhow feasible in cases where the same peer having the top request will be expected to have all the children chunks
of an intermediate one. This is the case for syncing and hash stream or when light swarm clients channel all their requests to a proxy node (public database lookup or unencrypted content).

This model would require no peer specific dpa and would involve the chunker split only to get the right hand side for the append for verification.

Delivery requests

once the appropriate ranges of the hashstream are retrieved and buffered, downstream peer just scans the hashes, looks them up in localstore, if not found, create a request entry with specific reference to the upstream peer as source.
The range is referenced by the chunk index. Alongside the name (indicating the stream, e.g., content chunks for bin 6) and the range
downstream peer sends a 128 long bitvector indicating which chunks are needed.
Newly created requests are satisfied bound together in a waitgroup which when done, will prompt sending the next one.
to be able to do check and storage concurrently, we keep a buffer of one, we start with two chunks.
For session syncing too, if it has not arrived  sby the time the next chunk.
