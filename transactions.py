from pybitcointools import *
import rlp
import re

class Transaction():
    def __init__(*args):

    def lpad(inp,L): return '\x00' * max(0,L - len(inp)) + inp

    def parse(self,data):
        if re.match('^[0-9a-fA-F]*$',data):
            data = data.decode('hex')
        o = rlp.unparse(data)
        self.to = lpad(o[0],20)
        self.value = decode(o[1],256)
        self.fee = decode(o[2],256)
        self.data = rlp.unparse(o[-3])
        self.sig = o[-4]
        rawhash = sha256(rlp.encode([self.to,self.value,self.fee,self.data]))
        v,r,s = ord(self.sig[0]), decode(self.sig[1:33],256), decode(self.sig[33:],256)
        self.from = hash160(ecdsa_raw_recover(rawhash,(v,r,s)))
    def sign(self,key):
        rawhash = sha256(rlp.parse([self.to,self.value,self.fee,self.data]))
        v,r,s = ecdsa_raw_sign(rawhash,args[5])
        self.sig = chr(v)+encode(r,256,32)+encode(s,256,32)
        self.from = hash160(privtopub(args[5]))
    def serialize(self):
        return rlp.parse([self.to, self.value, self.fee, self.data, self.sig]).encode('hex')
    def hash(self):
        return bin_sha256(self.serialize())
