from pybitcointools import *
import rlp
import re
from transactions import Transaction

class Block():
    def __init__(self,data=None):
        if not data:
            return
        if re.match('^[0-9a-fA-F]*$',data):
            data = data.decode('hex')
        header, tree_node_list, transaction_list, sibling_list = rlp.decode(data)
        h = rlp.decode(header)
        self.prevhash = encode(h[0],16,64)
        self.coinbase = encode(h[1],16,40)
        self.balance_root = encode(h[2],256,32)
        self.contract_root = encode(h[3],256,32)
        self.difficulty = h[4]
        self.timestamp = h[5]
        transactions_root = encode(h[6],256,32)
        siblings_root = encode(h[7],256,32)
        self.nonce = h[8]
        self.datastore = {}
        for nd in rlp.decode(tree_node_list):
            ndk = bin_sha256(nd)
            self.datastore[ndk] = rlp.decode(nd)
        self.transactions = [Transaction(x) for x in rlp.decode(transaction_list)]
        self.siblings = [rlp.decode(x) for x in rlp.decode(sibling_list)]
        # Verifications
        if self.balance_root != '' and self.balance_root not in self.datastore:
            raise Exception("Balance Merkle root not found!")
        if self.contract_root != '' and self.contract_root not in self.datastore:
            raise Exception("Contract Merkle root not found!")
        if bin_sha256(transaction_list) != transactions_root:
            raise Exception("Transaction list root hash does not match!")
        if bin_sha256(sibling_list) != sibling_root:
            raise Exception("Transaction list root hash does not match!")
        for sibling in self.siblings:
            if sibling[0] != self.prevhash:
                raise Exception("Sibling's parent is not my parent!")
            

    hexalpha = '0123456789abcdef'

    def get_updated_state(self,node,key,value):
        curnode = self.datastore.get(node,None)
        # Insertion case
        if value != 0 and value != '':
            # Base case
            if key == '':
                return value
            # Inserting into an empty trie
            if not curnode:
                newnode = [ key, value ]
                k = sha256(rlp.encode(newnode))
                self.datastore[k] = newnode
                return k
            elif len(curnode) == 2:
                # Inserting into a (k,v), same key
                if key == curnode[0]:
                    newnode = [ key, value ]
                    k = sha256(rlp.encode(newnode))
                    self.datastore[k] = newnode
                    return k
                # Inserting into a (k,v), different key
                else:
                    i = 0
                    while key[:i] == curnode[0][:i]: i += 1
                    k1 = self.get_updated_state(None,curnode[0][i:],curnode[1])
                    k2 = self.get_updated_state(None,key[i:],value)
                    newnode3 = [ None ] * 16
                    newnode3[ord(curnode[0][0])] = k1
                    newnode3[ord(key[0])] = k2
                    k3 = sha256(rlp.encode(newnode3))
                    self.datastore[k3] = newnode3
                    # No prefix sharing
                    if i == 1:
                        return k3
                    # Prefix sharing
                    else:
                        newnode4 = [ key[:i-1], k3 ]
                        k4 = sha256(rlp.encode(newnode4))
                        self.datastore[k4] = newnode4
                        return k4
            else:
                # inserting into a 16-array
                newnode1 = self.get_updated_state(curnode[ord(key[0])],key[1:],value)
                newnode2 = [ curnode[i] for i in range(16) ]
                newnode2[ord(key[0])] = newnode1
                return newnode2
        # Deletion case
        else:
            # Base case
            if key == '':
                return None
            # Deleting from a (k,v); obvious
            if len(curnode) == 2:
                if key == curnode[0]: return None
                else: return node
            else:
                k1 = self.get_updated_state(curnode[ord(key[0])],key[1:],value)
                newnode = [ curnode[i] for i in range(16) ]
                newnode[ord(key[0])] = k1
                totalnodes = sum([ 1 if newnode2[i] else 0 for i in range(16) ])
                if totalnodes == 0:
                    raise Exception("Can't delete from two to zero! Error! Waahmbulance!")
                elif totalnodes == 1:
                    # If only have one node left, we revert to (key, value)
                    node_index = [i for i in range(16) if newnode2[i]][0]
                    node2 = self.datastore[curnode[node_index]]
                    if len(node2) == 2:
                        # If it's a (key, value), we just prepend to the key
                        newnode = [ chr(node_index) + node2[0], node2[1] ]
                    else:
                        # Otherwise, we just make a single-char (key, value) pair
                        newnode = [ chr(node_index), curnode[node_index] ]
                k2 = sha256(rlp.encode(newnode))
                self.datastore[k2] = newnode
                return k2


    def update_balance(self,address,value):
        # Use out internal representation for the key
        key = ''.join([chr(hexalpha.find(x)) for x in address.encode('hex')])
        self.balance_root = self.get_updated_state(self.balance_root,key,value)

    def update_contract_state(self,address,index,value):
        # Use out internal representation for the key
        key = ''.join([chr(hexalpha.find(x)) for x in (address+index).encode('hex')])
        self.contract_root = self.get_updated_state(self.contract_root,key,value)

    def get_state_value(self,node,key):
        if key == '':
            return node
        if not curnode:
            return None
        curnode = self.datastore.get(node,None)
        return self.get_state_value(curnode[ord(key[0])],key[1:])

    def get_balance(self,address):
        # Use out internal representation for the key
        key = ''.join([chr(hexalpha.find(x)) for x in (address).encode('hex')])
        return self.get_state_value(self.balance_root,key)

    def get_contract_state(self,address,index):
        # Use out internal representation for the key
        key = ''.join([chr(hexalpha.find(x)) for x in (address+index).encode('hex')])
        return self.get_state_value(self.contract_root,key)

    def get_state_size(self,node):
        if node is None: return 0
        curnode = self.datastore.get(node,None)
        if not curnode: return 0
        elif len(curnode) == 2:
            return self.get_state_size(curnode[1])
        else:
            total = 0
            for i in range(16): total += self.get_state_size(curnode[i])
            return total

    def get_contract_size(self,address):
        # Use out internal representation for the key
        key = ''.join([chr(hexalpha.find(x)) for x in (address).encode('hex')])
        return self.get_state_size(self.get_state_value(self.contract_root,key))
        
    def serialize(self):
        nodes = {}
        def process(node):
            if node is None: return
            curnode = self.datastore.get(node,None)
            if curnode:
                index = sha256(rlp.encode(curnode))
                nodes[index] = curnode
                if len(node) == 2:
                    process(curnode[1])
                elif len(node) == 16:
                    for i in range(16): process(curnode[i])
        process(self.balance_root)
        process(self.contract_root)
        tree_nodes = [nodes[x] for x in nodes]
        nodelist = rlp.encode(tree_nodes)
        txlist = rlp.encode([x.serialize() for x in self.transactions])
        siblinglist = rlp.encode(self.siblings)
        header = rlp.encode([self.prevhash, self.coinbase, self.balance_root, self.contract_root, self.difficulty, self.timestamp, bin_sha256(txlist), bin_sha256(siblinglist])
        return rlp.encode([header, nodelist, txlist, siblinglist])
