# The MIT License (MIT)
#
# Copyright (c) 2014 Richard Moore
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.


# This file is a modified version of the test suite for pyaes (https://www.github.com/ricmoo/pyaes/)


import json

class NoIndent(object):
    def __init__(self, value):
        self.value = value

def default(o, encoder=json.JSONEncoder()):
    if isinstance(o, NoIndent):
        return '__' + json.dumps(o.value) + '__'
    return encoder.default(o)


import os, time

Tests = []

# compare against a known working implementation
from Crypto.Cipher import AES as KAES
from Crypto.Util import Counter as KCounter
for mode in [ 'CBC', 'CTR',  'CFB', 'ECB', 'OFB' ]:

    (tt_ksetup, tt_kencrypt, tt_kdecrypt) = (0.0, 0.0, 0.0)
    (tt_setup, tt_encrypt, tt_decrypt) = (0.0, 0.0, 0.0)
    count = 0

    for key_size in (128, 192, 256):

        for test in xrange(1, 8):
            key = os.urandom(key_size // 8)

            iv = None
            segment_size = None

            if mode == 'CBC':
                iv = os.urandom(16)

                text_length = [None, 16, 16, 16, 32, 48, 64, 64, 64][test]
                if test == 1:
                    plaintext = [ '' ]
                else:
                    plaintext = [ os.urandom(text_length) for x in xrange(0, test) ]

                kaes = KAES.new(key, KAES.MODE_CBC, IV = iv)
                kaes2 = KAES.new(key, KAES.MODE_CBC, IV = iv)

            elif mode == 'CFB':
                iv = os.urandom(16)
                plaintext = [ os.urandom(test * 5) for x in xrange(0, test) ]

                kaes = KAES.new(key, KAES.MODE_CFB, IV = iv, segment_size = test * 8)
                kaes2 = KAES.new(key, KAES.MODE_CFB, IV = iv, segment_size = test * 8)

                segment_size = test

            elif mode == 'ECB':
                text_length = [None, 16, 16, 16, 32, 48, 64, 64, 64][test]
                if test == 1:
                    plaintext = [ '' ]
                else:
                    plaintext = [ os.urandom(text_length) for x in xrange(0, test) ]

                kaes = KAES.new(key, KAES.MODE_ECB)
                kaes2 = KAES.new(key, KAES.MODE_ECB)

            elif mode == 'OFB':
                iv = os.urandom(16)
                plaintext = [ os.urandom(16) for x in xrange(0, test) ]

                kaes = KAES.new(key, KAES.MODE_OFB, IV = iv)
                kaes2 = KAES.new(key, KAES.MODE_OFB, IV = iv)

            elif mode == 'CTR':
                text_length = [None, 3, 16, 127, 128, 129, 1500, 10000, 100000, 10001, 10002, 10003, 10004, 10005, 10006, 10007, 10008][test]
                if test < 6:
                    plaintext = [ os.urandom(text_length) ]
                else:
                    plaintext = [ os.urandom(text_length) for x in xrange(0, test) ]

                kaes = KAES.new(key, KAES.MODE_CTR, counter = KCounter.new(128, initial_value = 0))
                kaes2 = KAES.new(key, KAES.MODE_CTR, counter = KCounter.new(128, initial_value = 0))

            count += 1

            kenc = [kaes.encrypt(p) for p in plaintext]

            iv_enc = None
            if iv:
                iv_enc = NoIndent([ord(x) for x in iv])
            Tests.append(dict(
                encrypted = [NoIndent([ord(x) for x in chunk]) for chunk in kenc],
                iv = iv_enc,
                key = NoIndent([ord(x) for x in key]),
                modeOfOperation = mode.lower(),
                plaintext = [NoIndent([ord(x) for x in chunk]) for chunk in plaintext],
                segmentSize = segment_size,
            ))

            dt1 = [kaes2.decrypt(k) for k in kenc]

print json.dumps(Tests, indent = 4, sort_keys = True, default = default).replace('"__', '').replace('__"', '')

