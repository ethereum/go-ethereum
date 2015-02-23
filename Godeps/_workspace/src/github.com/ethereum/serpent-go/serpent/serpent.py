import serpent_pyext as pyext
import sys
import re

VERSION = '1.7.7'


class Metadata(object):
    def __init__(self, li):
        self.file = li[0]
        self.ln = li[1]
        self.ch = li[2]

    def out(self):
        return [self.file, self.ln, self.ch]


class Token(object):
    def __init__(self, val, metadata):
        self.val = val
        self.metadata = Metadata(metadata)

    def out(self):
        return [0, self.val, self.metadata.out()]

    def __repr__(self):
        return str(self.val)


class Astnode(object):
    def __init__(self, val, args, metadata):
        self.val = val
        self.args = map(node, args)
        self.metadata = Metadata(metadata)

    def out(self):
        o = [1, self.val, self.metadata.out()]+[x.out() for x in self.args]
        return o

    def __repr__(self):
        o = '(' + self.val
        subs = map(repr, self.args)
        k = 0
        out = " "
        while k < len(subs) and o != "(seq":
            if '\n' in subs[k] or len(out + subs[k]) >= 80:
                break
            out += subs[k] + " "
            k += 1
        if k < len(subs):
            o += out + "\n  "
            o += '\n  '.join('\n'.join(subs[k:]).split('\n'))
            o += '\n)'
        else:
            o += out[:-1] + ')'
        return o


def node(li):
    if li[0]:
        return Astnode(li[1], li[3:], li[2])
    else:
        return Token(li[1], li[2])


def take(x):
    return pyext.parse_lll(x) if isinstance(x, (str, unicode)) else x.out()


def takelist(x):
    return map(take, parse(x).args if isinstance(x, (str, unicode)) else x)


compile = lambda x: pyext.compile(x)
compile_chunk = lambda x: pyext.compile_chunk(x)
compile_to_lll = lambda x: node(pyext.compile_to_lll(x))
compile_chunk_to_lll = lambda x: node(pyext.compile_chunk_to_lll(x))
compile_lll = lambda x: pyext.compile_lll(take(x))
parse = lambda x: node(pyext.parse(x))
rewrite = lambda x: node(pyext.rewrite(take(x)))
rewrite_chunk = lambda x: node(pyext.rewrite_chunk(take(x)))
pretty_compile = lambda x: map(node, pyext.pretty_compile(x))
pretty_compile_chunk = lambda x: map(node, pyext.pretty_compile_chunk(x))
pretty_compile_lll = lambda x: map(node, pyext.pretty_compile_lll(take(x)))
serialize = lambda x: pyext.serialize(takelist(x))
deserialize = lambda x: map(node, pyext.deserialize(x))

is_numeric = lambda x: isinstance(x, (int, long))
is_string = lambda x: isinstance(x, (str, unicode))
tobytearr = lambda n, L: [] if L == 0 else tobytearr(n / 256, L - 1)+[n % 256]


# A set of methods for detecting raw values (numbers and strings) and
# converting them to integers
def frombytes(b):
    return 0 if len(b) == 0 else ord(b[-1]) + 256 * frombytes(b[:-1])


def fromhex(b):
    hexord = lambda x: '0123456789abcdef'.find(x)
    return 0 if len(b) == 0 else hexord(b[-1]) + 16 * fromhex(b[:-1])


def numberize(b):
    if is_numeric(b):
        return b
    elif b[0] in ["'", '"']:
        return frombytes(b[1:-1])
    elif b[:2] == '0x':
        return fromhex(b[2:])
    elif re.match('^[0-9]*$', b):
        return int(b)
    elif len(b) == 40:
        return fromhex(b)
    else:
        raise Exception("Cannot identify data type: %r" % b)


def enc(n):
    if is_numeric(n):
        return ''.join(map(chr, tobytearr(n, 32)))
    elif is_string(n) and len(n) == 40:
        return '\x00' * 12 + n.decode('hex')
    elif is_string(n):
        return '\x00' * (32 - len(n)) + n
    elif n is True:
        return 1
    elif n is False or n is None:
        return 0


def encode_datalist(*args):
    if isinstance(args, (tuple, list)):
        return ''.join(map(enc, args))
    elif not len(args) or args[0] == '':
        return ''
    else:
        # Assume you're getting in numbers or addresses or 0x...
        return ''.join(map(enc, map(numberize, args)))


def decode_datalist(arr):
    if isinstance(arr, list):
        arr = ''.join(map(chr, arr))
    o = []
    for i in range(0, len(arr), 32):
        o.append(frombytes(arr[i:i + 32]))
    return o


def encode_abi(funid, *args):
    len_args = ''
    normal_args = ''
    var_args = ''
    for arg in args:
        if isinstance(arg, str) and len(arg) and \
                arg[0] == '"' and arg[-1] == '"':
            len_args += enc(numberize(len(arg[1:-1])))
            var_args += arg[1:-1]
        elif isinstance(arg, list):
            for a in arg:
                var_args += enc(numberize(a))
            len_args += enc(numberize(len(arg)))
        else:
            normal_args += enc(numberize(arg))
    return chr(int(funid)) + len_args + normal_args + var_args


def decode_abi(arr, *lens):
    o = []
    pos = 1
    i = 0
    if len(lens) == 1 and isinstance(lens[0], list):
        lens = lens[0]
    while pos < len(arr):
        bytez = int(lens[i]) if i < len(lens) else 32
        o.append(frombytes(arr[pos: pos + bytez]))
        i, pos = i + 1, pos + bytez
    return o


def main():
    if len(sys.argv) == 1:
        print "serpent <command> <arg1> <arg2> ..."
    else:
        cmd = sys.argv[2] if sys.argv[1] == '-s' else sys.argv[1]
        if sys.argv[1] == '-s':
            args = [sys.stdin.read()] + sys.argv[3:]
        elif sys.argv[1] == '-v':
            print VERSION
            sys.exit()
        else:
            cmd = sys.argv[1]
            args = sys.argv[2:]
        if cmd in ['deserialize', 'decode_datalist', 'decode_abi']:
            args[0] = args[0].strip().decode('hex')
        o = globals()[cmd](*args)
        if isinstance(o, (Token, Astnode, list)):
            print repr(o)
        else:
            print o.encode('hex')
