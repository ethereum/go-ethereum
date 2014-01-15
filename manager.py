import rlp
import leveldb
from blocks import Block
from transactions import Transaction
import processblock
import hashlib
from pybitcointools import *

txpool = {}

genesis_header = [
    0,
    '',
    bin_sha256(rlp.encode([])),
    '',
    '',
    bin_sha256(rlp.encode([])),
    2**36,
    0,
    0,
    ''
]

genesis = [ genesis_header, [], [] ]

mainblk = Block(rlp.encode(genesis))

db = leveldb.LevelDB("objects")

def genaddr(seed):
    priv = bin_sha256(seed)
    addr = bin_sha256(privtopub(priv)[1:])[-20:]
    return priv,addr

# For testing
k1,a1 = genaddr("123")
k2,a2 = genaddr("456")

def broadcast(obj):
    pass

def receive(obj):
    d = rlp.decode(obj)
    # Is transaction
    if len(d) == 8:
        tx = Transaction(obj)
        if mainblk.get_balance(tx.sender) < tx.value + tx.fee: return
        if mainblk.get_nonce(tx.sender) != tx.nonce: return
        txpool[bin_sha256(blk)] = blk
        broadcast(blk)
    # Is message
    elif len(d) == 2:
        if d[0] == 'getobj':
            try: return db.Get(d[1][0])
            except:
                try: return mainblk.state.db.get(d[1][0])
                except: return None
        elif d[0] == 'getbalance':
            try: return mainblk.state.get_balance(d[1][0])
            except: return None
        elif d[0] == 'getcontractroot':
            try: return mainblk.state.get_contract(d[1][0]).root
            except: return None
        elif d[0] == 'getcontractsize':
            try: return mainblk.state.get_contract(d[1][0]).get_size()
            except: return None
        elif d[0] == 'getcontractstate':
            try: return mainblk.state.get_contract(d[1][0]).get(d[1][1])
            except: return None
    # Is block
    elif len(d) == 3:
        blk = Block(obj)
        p = block.prevhash
        try:
            parent = Block(db.Get(p))
        except:
            return
        uncles = block.uncles
        for s in uncles:
            try:
                sib = db.Get(s)
            except:
                return
        processblock.eval(parent,blk.transactions,blk.timestamp,blk.coinbase)
        if parent.state.root != blk.state.root: return
        if parent.difficulty != blk.difficulty: return
        if parent.number != blk.number: return
        db.Put(blk.hash(),blk.serialize())
    
