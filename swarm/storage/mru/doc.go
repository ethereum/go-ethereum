// Package mru defines Mutable resource updates.
// A Mutable Resource is an entity which allows updates to a resource
// without resorting to ENS on each update.
// The update scheme is built on swarm chunks with chunk keys following
// a predictable, versionable pattern.
//
// Updates are defined to be periodic in nature, where the update frequency
// is expressed in seconds.
//
// The root entry of a mutable resource is tied to a unique identifier that
// is deterministically generated out of the metadata content that describes
// the resource. This metadata includes a user-defined resource name, a resource
// start time that indicates when the resource becomes valid,
// the frequency in seconds with which the resource is expected to be updated, both of
// which are stored as little-endian uint64 values in the database (for a
// total of 16 bytes). It also contains the owner's address (ownerAddr)
// This MRU info is stored in a separate content-addressed chunk
// (call it the metadata chunk), with the following layout:
//
// (00|length|startTime|frequency|name|ownerAddr)
//
// (The two first zero-value bytes are used for disambiguation by the chunk validator,
// and update chunk will always have a value > 0 there.)
//
// Each metadata chunk is identified by its rootAddr, calculated as follows:
// metaHash=H(len(metadata), startTime, frequency,name)
// rootAddr = H(metaHash, ownerAddr).
// where H is the SHA3 hash function
// This scheme effectively locks the root chunk so that only the owner of the private key
// that ownerAddr was derived from can sign updates.
//
// The root entry tells the requester from when the mutable resource was
// first added (Unix time in seconds) and in which moments to look for the
// actual updates. Thus, a resource update for identifier "føø.bar"
// starting at unix time 1528800000 with frequency 300 (every 5 mins) will have updates on 1528800300,
// 1528800600, 1528800900 and so on.
//
// Actual data updates are also made in the form of swarm chunks. The keys
// of the updates are the hash of a concatenation of properties as follows:
//
// updateAddr = H(period, version, rootAddr)
// where H is the SHA3 hash function
// The period is (currentTime - startTime) / frequency
//
// Using our previous example, this means that a period 3 will happen when the
// clock hits 1528800900
//
// If more than one update is made in the same period, incremental
// version numbers are used successively.
//
// A user looking up a resource would only need to know the rootAddr in order to get the versions
//
// the resource update data is:
// resourcedata = headerlength|period|version|rootAddr|flags|metaHash
// where flags is a 1-byte flags field. Flag 0 is set to 1 to indicate multihash
//
// the full update data that goes in the chunk payload is:
// resourcedata|sign(resourcedata)
//
// headerlength is a 16 bit value containing the byte length of period|version|rootAddr|flags|metaHash
package mru
