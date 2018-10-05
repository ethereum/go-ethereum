/*
Package feeds defines Swarm Feeds.

Swarm Feeds allows a user to build an update feed about a particular topic
without resorting to ENS on each update.
The update scheme is built on swarm chunks with chunk keys following
a predictable, versionable pattern.

A Feed is tied to a unique identifier that is deterministically generated out of
the chosen topic.

A Feed is defined as the series of updates of a specific user about a particular topic

Actual data updates are also made in the form of swarm chunks. The keys
of the updates are the hash of a concatenation of properties as follows:

updateAddr = H(Feed, Epoch ID)
where H is the SHA3 hash function
Feed is the combination of Topic and the user address
Epoch ID is a time slot. See the lookup package for more information.

A user looking up a the latest update in a Feed only needs to know the Topic
and the other user's address.

The Feed Update data is:
updatedata = Feed|Epoch|data

The full update data that goes in the chunk payload is:
updatedata|sign(updatedata)

Structure Summary:

Request: Feed Update with signature
	Update: headers + data
		Header: Protocol version and reserved for future use placeholders
		ID: Information about how to locate a specific update
			Feed: Represents a user's series of publications about a specific Topic
				Topic: Item that the updates are about
				User: User who updates the Feed
			Epoch: time slot where the update is stored

*/
package feed
