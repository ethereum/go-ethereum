/*
Package feeds defines Swarm Feeds.

A Mutable Resource is an entity which allows updates to a resource
without resorting to ENS on each update.
The update scheme is built on swarm chunks with chunk keys following
a predictable, versionable pattern.

A Resource is tied to a unique identifier that is deterministically generated out of
the chosen topic.

A Resource View is defined as a specific user's point of view about a particular resource.
Thus, a View is a Topic + the user's address (userAddr)

Actual data updates are also made in the form of swarm chunks. The keys
of the updates are the hash of a concatenation of properties as follows:

updateAddr = H(View, Epoch ID)
where H is the SHA3 hash function
View is the combination of Topic and the user address
Epoch ID is a time slot. See the lookup package for more information.

A user looking up a resource would only need to know the View in order to
another user's updates

The resource update data is:
resourcedata = View|Epoch|data

the full update data that goes in the chunk payload is:
resourcedata|sign(resourcedata)

Structure Summary:

Request: Resource update with signature
	ResourceUpdate: headers + data
		Header: Protocol version and reserved for future use placeholders
		ID: Information about how to locate a specific update
			View: Author of the update and what is updating
				Topic: Item that the updates are about
				User: User who updates the resource
			Epoch: time slot where the update is stored

*/
package mru
