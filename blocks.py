from pybitcointools import *
import rlp
import re
from transactions import Transaction
from trie import Trie

class Block():
    def __init__(self,data=None):

        if not data:
            return

        if re.match('^[0-9a-fA-F]*$',data):
            data = data.decode('hex')

        header,  transaction_list, self.siblings = rlp.decode(data)
        [ number,
          self.prevhash,
          self.siblings_root,
          self.coinbase,
          state_root,
          self.transactions_root,
          diff,
          timestamp,
          nonce,
          self.extra ] = header
        self.number = decode(number,256)
        self.difficulty = decode(difficulty,256)
        self.timestamp = decode(timestamp,256)
        self.nonce = decode(nonce,256)
        self.transactions = [Transaction(x) for x in transaction_list)]
        self.state = Trie('statedb',state_root)

        # Verifications
        if self.state.root != '' and self.state.__get_state(self.state.root,[]) == '':
            raise Exception("State Merkle root not found in database!")
        if bin_sha256(transaction_list) != transactions_root:
            raise Exception("Transaction list root hash does not match!")
        if bin_sha256(sibling_list) != sibling_root:
            raise Exception("Transaction list root hash does not match!")
        for sibling in self.siblings:
            if sibling[0] != self.prevhash:
                raise Exception("Sibling's parent is not my parent!")
        # TODO: check POW
            
    def send(self,tx):
        # Subtract value and fee from sender account and increment nonce if applicable
        sender_state = rlp.decode(self.state.get(tx.from))
        if not sender_state:
            return False
        sender_value = decode(sender_state[1],256)
        if value + fee > sender_value:
            return False
        sender_state[1] = encode(sender_value - value - fee,256)
        # Nonce applies only to key-based addresses
        if decode(sender_state[0],256) == 0:
            if decode(sender_state[2],256) != tx.nonce:
                return False
            sender_state[2] = encode(tx.nonce + 1,256)
        self.state.update(tx.from,sender_state)
        # Add money to receiver
        if tx.to > '':
            receiver_state = rlp.decode(self.state.get(tx.to)) or ['', '', '']
            receiver_state[1] = encode(decode(receiver_state[1],256) + value,256)
            self.state.update(tx.to,receiver_state)
        # Create a new contract
        else:
            addr = tx.hash()[:20]
            contract = block.get_contract(addr)
            if contract.root != '': return False
            for i in range(len(tx.data)):
                contract.update(encode(i,256,32),tx.data[i])
            block.update_contract(addr)
        # Pay fee to miner
        miner_state = rlp_decode(self.state.get(self.coinbase)) or ['','','']
        miner_state[1] = encode(decode(miner_state[1],256) + fee,256)
        self.state.update(self.coinbase,miner_state)
        return True

    def pay_fee(self,address,fee,tominer=True):
        # Subtract fee from sender
        sender_state = rlp.decode(self.state.get(address))
        if not sender_state:
            return False
        sender_value = decode(sender_state[1],256)
        if sender_value < fee:
            return False
        sender_state[1] = encode(sender_value - fee,256)
        self.state.update(address,sender_state)
        # Pay fee to miner
        if tominer:
            miner_state = rlp.decode(self.state.get(self.coinbase)) or ['','','']
            miner_state[1] = encode(decode(miner_state[1],256) + fee,256)
            self.state.update(self.coinbase,miner_state)
        return True

    def get_nonce(self,address):
        state = rlp.decode(self.state.get(address))
        if not state or decode(state[0],256) == 0: return False
        return decode(state[2],256)

    def get_balance(self,address):
        state = rlp.decode(self.state.get(address))
        return decode(state[1] || '',256)

    # Making updates to the object obtained from this method will do nothing. You need
    # to call update_contract to finalize the changes.
    def get_contract(self,address):
        state = rlp.decode(self.state.get(address))
        if not state or decode(state[0],256) == 0: return False
        return Trie('statedb',state[2])

    def update_contract(self,address,contract):
        state = rlp.decode(self.state.get(address))
        if not state or decode(state[0],256) == 0: return False
        state[2] = contract.root
        self.state.update(address,state)

    # Serialization method; should act as perfect inverse function of the constructor
    # assuming no verification failures
    def serialize(self):
        txlist = [x.serialize() for x in self.transactions]
        header = [ encode(self.number,256),
                   self.prevhash,
                   bin_sha256(rlp.encode(self.siblings)),
                   self.coinbase,
                   self.state.root,
                   bin_sha256(rlp.encode(self.txlist)),
                   encode(self.difficulty,256),
                   encode(self.timestamp,256),
                   encode(self.nonce,256),
                   self.extra ]
        return rlp.encode([header, txlist, self.siblings ])

    def hash(self):
        return bin_sha256(self.serialize())
