import leveldb
import rlp
import hashlib

def sha256(x): return hashlib.sha256(x).digest()

class DB():
    def __init__(self,dbfile): self.db = leveldb.LevelDB(dbfile)
    def get(self,key):
        try: return self.db.Get(key)
        except: return ''
    def put(self,key,value): return self.db.Put(key,value)
    def delete(self,key): return self.db.Delete(key)

databases = {}

class Trie():
    def __init__(self,dbfile,root='',debug=False):
        self.root = root
        self.debug = debug
        if dbfile not in databases:
            databases[dbfile] = DB(dbfile)
        self.db = databases[dbfile]

    def __encode_key(self,key):
        term = 1 if key[-1] == 16 else 0
        oddlen = (len(key) - term) % 2
        prefix = ('0' if oddlen else '')
        main = ''.join(['0123456789abcdef'[x] for x in key[:len(key)-term]])
        return chr(2 * term + oddlen) + (prefix+main).decode('hex')

    def __decode_key(self,key):
        o = ['0123456789abcdef'.find(x) for x in key[1:].encode('hex')]
        if key[0] == '\x01' or key[0] == '\x03': o = o[1:]
        if key[0] == '\x02' or key[0] == '\x03': o.append(16)
        return o
        
    def __get_state(self,node,key):
        if self.debug: print 'nk',node.encode('hex'),key
        if len(key) == 0 or not node:
            return node
        curnode = rlp.decode(self.db.get(node))
        if self.debug: print 'cn', curnode
        if not curnode:
            raise Exception("node not found in database")
        elif len(curnode) == 2:
            (k2,v2) = curnode
            k2 = self.__decode_key(k2)
            if len(key) >= len(k2) and k2 == key[:len(k2)]:
                return self.__get_state(v2,key[len(k2):])
            else:
                return ''
        elif len(curnode) == 17:
            return self.__get_state(curnode[key[0]],key[1:])

    def __put(self,node):
        rlpnode = rlp.encode(node)
        h = sha256(rlpnode)
        self.db.put(h,rlpnode)
        return h

    def __update_state(self,node,key,value):
        if value != '': return self.__insert_state(node,key,value)
        else: return self.__delete_state(node,key)

    def __insert_state(self,node,key,value):
        if self.debug: print 'ink', node.encode('hex'), key
        if len(key) == 0:
            return value
        else:
            if not node:
                newnode = [ self.__encode_key(key), value ]
                return self.__put(newnode)
            curnode = rlp.decode(self.db.get(node))
            if self.debug: print 'icn', curnode
            if not curnode:
                raise Exception("node not found in database")
            if len(curnode) == 2:
                (k2, v2) = curnode
                k2 = self.__decode_key(k2)
                if key == k2:
                    newnode = [ self.__encode_key(key), value ]
                    return self.__put(newnode)
                else:
                    i = 0
                    while key[:i+1] == k2[:i+1] and i < len(k2): i += 1
                    if i == len(k2):
                        newhash3 = self.__insert_state(v2,key[i:],value)
                    else:
                        newnode1 = self.__insert_state('',key[i+1:],value)
                        newnode2 = self.__insert_state('',k2[i+1:],v2)
                        newnode3 = [ '' ] * 17
                        newnode3[key[i]] = newnode1
                        newnode3[k2[i]] = newnode2
                        newhash3 = self.__put(newnode3)
                    if i == 0:
                        return newhash3
                    else:
                        newnode4 = [ self.__encode_key(key[:i]), newhash3 ]
                        return self.__put(newnode4)
            else:
                newnode = [ curnode[i] for i in range(17) ]
                newnode[key[0]] = self.__insert_state(curnode[key[0]],key[1:],value)
                return self.__put(newnode)
    
    def __delete_state(self,node,key):
        if self.debug: print 'dnk', node.encode('hex'), key
        if len(key) == 0 or not node:
            return ''
        else:
            curnode = rlp.decode(self.db.get(node))
            if not curnode:
                raise Exception("node not found in database")
            if self.debug: print 'dcn', curnode
            if len(curnode) == 2:
                (k2, v2) = curnode
                k2 = self.__decode_key(k2)
                if key == k2:
                    return ''
                elif key[:len(k2)] == k2:
                    newhash = self.__delete_state(v2,key[len(k2):])
                    childnode = rlp.decode(self.db.get(newhash))
                    if len(childnode) == 2:
                        newkey = k2 + self.__decode_key(childnode[0])
                        newnode = [ self.__encode_key(newkey), childnode[1] ]
                    else:
                        newnode = [ curnode[0], newhash ]
                    return self.__put(newnode)
                else: return node
            else:
                newnode = [ curnode[i] for i in range(17) ]
                newnode[key[0]] = self.__delete_state(newnode[key[0]],key[1:])
                onlynode = -1
                for i in range(17):
                    if newnode[i]:
                        if onlynode == -1: onlynode = i
                        else: onlynode = -2
                if onlynode >= 0:
                    childnode = rlp.decode(self.db.get(newnode[onlynode]))
                    if not childnode:
                        raise Exception("?????")
                    if len(childnode) == 17:
                        newnode2 = [ key[0], newnode[onlynode] ]
                    elif len(childnode) == 2:
                        newkey = [onlynode] + self.__decode_key(childnode[0])
                        newnode2 = [ self.__encode_key(newkey), childnode[1] ]
                else:
                    newnode2 = newnode
                return self.__put(newnode2)

    def __get_size(self,node):
        if not node: return 0
        curnode = self.db.get(node)
        if not curnode:
            raise Exception("node not found in database")
        if len(curnode) == 2:
            key = self.__decode_key(curnode[0])
            if key[-1] == 16: return 1
            else: return self.__get_size(curnode[1])
        elif len(curnode) == 17:
            total = 0
            for i in range(16):
                total += self.__get_size(curnode[i])
            if curnode[16]: total += 1
            return total

    def get(self,key):
        key2 = ['0123456789abcdef'.find(x) for x in key.encode('hex')] + [16]
        return self.__get_state(self.root,key2)

    def get_size(self): return self.__get_size(self.root)

    def update(self,key,value):
        if not isinstance(key,str) or not isinstance(value,str):
            raise Exception("Key and value must be strings")
        key2 = ['0123456789abcdef'.find(x) for x in key.encode('hex')] + [16]
        self.root = self.__update_state(self.root,key2,value)
