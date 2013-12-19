def binary_length(n):
    if n == 0: return 0
    else: return 1 + binary_length(n / 256)

def to_binary_array(n,L=None):
    if L is None: L = binary_length(n)
    if n == 0: return []
    else:
        x = to_binary_array(n / 256)
        x.append(n % 256)
        return x

def to_binary(n,L=None): return ''.join([chr(x) for x in to_binary_array(n,L)])

def from_binary(b):
    if len(b) == 0: return 0
    else: return ord(from_binary(b[:-1])) * 256 + b[-1]

def num_to_var_int(n):
    if n < 253: s = chr(n)
    else if n < 2**16: s = [253] + list(reversed(to_binary_array(n,2)))
    else if n < 2**32: s = [254] + list(reversed(to_binary_array(n,4)))
    else if n < 2**64: s = [255] + list(reversed(to_binary_array(n,8)))
    else raise Exception("number too big")
    return ''.join([chr(x) for x in s])

def decode(s):
    o = []
    index = 0
    def read_var_int():
        si = ord(s[index])
        index += 1
        if si < 253: return s[index - 1]
        elif si == 253: read = 2
        elif si == 254: read = 4
        elif si == 255: read = 8
        index += read
        return from_binary(s[index-read:index])
    while index < len(s):
        L = read_var_int()
        o.append(s[index:index+L])
    return o

def encode(s):
    if isinstance(s,(int,long)): return encode(to_binary(s))
    if isinstance(s,str): return num_to_var_int(len(s))+s
    else:
        x = ''.join([encode(x) for x in s])
        return num_to_var_int(len(s))+s
    
