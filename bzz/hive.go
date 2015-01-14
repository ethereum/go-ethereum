package bzz

import ()

/*
Hive is the logistic manager at swarm
It is based on kademlia wisdom and flexible forwarding policies for optimal network health.
Hive implements the PeerPool interface (Thx fjl) and as such plays a role in how peers are selected by the p2p server. Ideally the p2p server regularly polls the registered protocol peer pools for good peers (an ordered wishlist of peers to connect to) and chooses the best one not connected. The Bzz Hive is therefore keeping a persistent record of peers for reputation and proximity considerations (or any other indirect incentive maybe).
*/
