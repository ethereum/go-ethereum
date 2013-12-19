import rlp

def parse(inp):
    if inp[0] == '\x00':
        return { "type": "transaction", "data": rlp.parse(
