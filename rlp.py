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
    else: return from_binary(b[:-1]) * 256 + ord(b[-1])

def num_to_var_int(n):
    if n < 253:     s = [n]
    elif n < 2**16: s = [253] + list(to_binary_array(n,2))
    elif n < 2**32: s = [254] + list(to_binary_array(n,4))
    elif n < 2**64: s = [255] + list(to_binary_array(n,8))
    else:           raise Exception("number too big")
    return ''.join([chr(x) for x in s])

def __decode(s):
    if s == '': return None
    o = []
    index = [0]
    def read_var_int():
        si = ord(s[index[0]])
        index[0] += 1
        if si < 253: return si
        elif si == 253: read = 2
        elif si == 254: read = 4
        elif si == 255: read = 8
        index[0] += read
        return from_binary(s[index[0]-read:index[0]])
    while index[0] < len(s):
        tp = s[index[0]]
        index[0] += 1
        L = read_var_int()
        item = s[index[0]:index[0]+L]
        if tp == '\x00': o.append(item)
        else: o.append(__decode(item))
        index[0] += L
    return o

def decode(s): return __decode(s)[0]

def encode(s):
    if isinstance(s,(int,long)): return encode(to_binary(s))
    if isinstance(s,str): return '\x00'+num_to_var_int(len(s))+s
    else:
        x = ''.join([encode(x) for x in s])
        return '\x01'+num_to_var_int(len(x))+x
