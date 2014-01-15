import json, sys, os
import rlp, trie
import random

testdir = sys.argv[1] if len(sys.argv) >= 2 else '../tests'

rlpdata = json.loads(open(os.path.join(testdir,'rlptest.txt')).read())
for x,y in rlpdata:
    yprime = rlp.encode(x).encode('hex')
    if yprime != y: print "RLPEncode Mismatch: ",x,y,yprime
    xprime = rlp.decode(y.decode('hex'))
    jx, jxprime = json.dumps(x), json.dumps(xprime)
    if jx != jxprime: print "RLPDecode Mismatch: ",jx,jxprime,y

hexencodedata = json.loads(open(os.path.join(testdir,'hexencodetest.txt')).read())

for x,y in hexencodedata:
    yprime = trie.hexarraykey_to_bin(x).encode('hex')
    if yprime != y: print "HexEncode Mismatch: ",x,y,yprime
    xprime = trie.bin_to_hexarraykey(y.decode('hex'))
    jx,jxprime = json.dumps(x), json.dumps(xprime)
    if jx != jxprime: print "HexDecode Mismatch: ",jx,jxprime,y

triedata = json.loads(open(os.path.join(testdir,'trietest.txt')).read())

for x,y in triedata:
    t0 = trie.Trie('/tmp/trietest-'+str(random.randrange(1000000000000)))
    for k in x:
        t0.update(k,x[k])
    if t0.root.encode('hex') != y:
        print "Mismatch with adds only"
        continue
    t = trie.Trie('/tmp/trietest-'+str(random.randrange(1000000000000)))
    dummies, reals = [], []
    for k in x:
        reals.append([k,x[k]])
        dummies.append(k[:random.randrange(len(k)-1)])
        dummies.append(k+random.choice(dummies))
        dummies.append(k[:random.randrange(len(k)-1)]+random.choice(dummies))
    dummies_to_pop = set([])
    i = 0
    ops = []
    mp = {}
    success = [True]
    def update(k,v):
        t.update(k,v)
        if v == '' and k in mp: del mp[k]
        else: mp[k] = v
        ops.append([k,v,t.root.encode('hex')])
        tn = trie.Trie('/tmp/trietest-'+str(random.randrange(1000000000000)))
        for k in mp:
            tn.update(k,mp[k])
        if tn.root != t.root:
            print "Mismatch: "
            for op in ops: print op
            success[0] = False
    while i < len(reals):
        s = random.randrange(3)
        if s == 0:
            update(reals[i][0],reals[i][1])
            i += 1
        elif s == 1:
            k,v = random.choice(dummies), random.choice(dummies)
            update(k,v)
            dummies_to_pop.add(k)
        elif s == 2:
            if len(dummies_to_pop) > 0:
                k = random.choice(list(dummies_to_pop))
                update(k,'')
                dummies_to_pop.remove(k)
        if not success[0]:
            break
    if not success[0]:
        continue
    i = len(reals) * 2
    while len(dummies_to_pop) > 0:
        k = random.choice(list(dummies_to_pop))
        update(k,'')
        dummies_to_pop.remove(k)
        if not success[0]:
            break
