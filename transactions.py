from pybitcointools import *
import rlp
import re

class Transaction():
    def __init__(*args):
        if len(args) == 2:
            self.parse(args[1])
        else:
            self.nonce = args[1]
            self.to = args[2]
            self.value = args[3]
            self.fee = args[4]
            self.data = args[5]

    def parse(self,data):
        if re.match('^[0-9a-fA-F]*$',data):
            data = data.decode('hex')
        o = rlp.unparse(data)
        self.nonce = decode(o[0],256)
        self.to = o[1]
        self.value = decode(o[2],256)
        self.fee = decode(o[3],256)
        self.data = rlp.unparse(o[4])
        self.v = o[5]
        self.r = o[6]
        self.s = o[7]
        rawhash = sha256(rlp.encode([self.nonce,self.to,self.value,self.fee,self.data]))
        self.from = hash160(ecdsa_raw_recover(rawhash,(self.v,self.r,self.s)))

    def sign(self,key):
        rawhash = sha256(rlp.parse([self.to,self.value,self.fee,self.data]))
        self.v,self.r,self.s = ecdsa_raw_sign(rawhash,key)
        self.from = hash160(privtopub(key))

    def serialize(self):
        return rlp.parse([self.nonce, self.to, self.value, self.fee, self.data, self.v, self.r, self.s])

    def hex_serialize(self):
        return self.serialize().encode('hex')

    def hash(self):
        return bin_sha256(self.serialize())
