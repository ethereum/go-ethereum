import pyethash
from random import randint

def test_get_cache_size_not_None():
    for _ in range(100):
        block_num = randint(0,12456789)
        out = pyethash.get_cache_size(block_num)
        assert out != None

def test_get_full_size_not_None():
    for _ in range(100):
        block_num = randint(0,12456789)
        out = pyethash.get_full_size(block_num)
        assert out != None

def test_get_cache_size_based_on_EPOCH():
    for _ in range(100):
        block_num = randint(0,12456789)
        out1 = pyethash.get_cache_size(block_num)
        out2 = pyethash.get_cache_size((block_num // pyethash.EPOCH_LENGTH) * pyethash.EPOCH_LENGTH)
        assert out1 == out2

def test_get_full_size_based_on_EPOCH():
    for _ in range(100):
        block_num = randint(0,12456789)
        out1 = pyethash.get_full_size(block_num)
        out2 = pyethash.get_full_size((block_num // pyethash.EPOCH_LENGTH) * pyethash.EPOCH_LENGTH)
        assert out1 == out2

# See light_and_full_client_checks in test.cpp
def test_mkcache_is_as_expected():
    actual = pyethash.mkcache_bytes(
        1024,
        "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~").encode('hex')
    expected = "2da2b506f21070e1143d908e867962486d6b0a02e31d468fd5e3a7143aafa76a14201f63374314e2a6aaf84ad2eb57105dea3378378965a1b3873453bb2b78f9a8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995ca8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995ca8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995c259440b89fa3481c2c33171477c305c8e1e421f8d8f6d59585449d0034f3e421808d8da6bbd0b6378f567647cc6c4ba6c434592b198ad444e7284905b7c6adaf70bf43ec2daa7bd5e8951aa609ab472c124cf9eba3d38cff5091dc3f58409edcc386c743c3bd66f92408796ee1e82dd149eaefbf52b00ce33014a6eb3e50625413b072a58bc01da28262f42cbe4f87d4abc2bf287d15618405a1fe4e386fcdafbb171064bd99901d8f81dd6789396ce5e364ac944bbbd75a7827291c70b42d26385910cd53ca535ab29433dd5c5714d26e0dce95514c5ef866329c12e958097e84462197c2b32087849dab33e88b11da61d52f9dbc0b92cc61f742c07dbbf751c49d7678624ee60dfbe62e5e8c47a03d8247643f3d16ad8c8e663953bcda1f59d7e2d4a9bf0768e789432212621967a8f41121ad1df6ae1fa78782530695414c6213942865b2730375019105cae91a4c17a558d4b63059661d9f108362143107babe0b848de412e4da59168cce82bfbff3c99e022dd6ac1e559db991f2e3f7bb910cefd173e65ed00a8d5d416534e2c8416ff23977dbf3eb7180b75c71580d08ce95efeb9b0afe904ea12285a392aff0c8561ff79fca67f694a62b9e52377485c57cc3598d84cac0a9d27960de0cc31ff9bbfe455acaa62c8aa5d2cce96f345da9afe843d258a99c4eaf3650fc62efd81c7b81cd0d534d2d71eeda7a6e315d540b4473c80f8730037dc2ae3e47b986240cfc65ccc565f0d8cde0bc68a57e39a271dda57440b3598bee19f799611d25731a96b5dbbbefdff6f4f656161462633030d62560ea4e9c161cf78fc96a2ca5aaa32453a6c5dea206f766244e8c9d9a8dc61185ce37f1fc804459c5f07434f8ecb34141b8dcae7eae704c950b55556c5f40140c3714b45eddb02637513268778cbf937a33e4e33183685f9deb31ef54e90161e76d969587dd782eaa94e289420e7c2ee908517f5893a26fdb5873d68f92d118d4bcf98d7a4916794d6ab290045e30f9ea00ca547c584b8482b0331ba1539a0f2714fddc3a0b06b0cfbb6a607b8339c39bcfd6640b1f653e9d70ef6c985b"
    assert actual == expected

def test_calc_dataset_is_not_None():
    cache = pyethash.mkcache_bytes(
              1024,
              "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
    assert pyethash.calc_dataset_bytes(1024 * 32, cache) != None

def test_light_and_full_agree():
    cache = pyethash.mkcache_bytes(
              1024,
              "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
    full_size = 1024 * 32
    header = "~~~~~X~~~~~~~~~~~~~~~~~~~~~~~~~~"
    light_result = pyethash.hashimoto_light(full_size, cache, header, 0)
    dataset = pyethash.calc_dataset_bytes(full_size, cache)
    full_result = pyethash.hashimoto_full(dataset, header, 0)
    assert light_result["mix digest"] != None
    assert len(light_result["mix digest"]) == 32
    assert light_result["mix digest"] == full_result["mix digest"]
    assert light_result["result"] != None
    assert len(light_result["result"]) == 32
    assert light_result["result"] == full_result["result"]

def int_to_bytes(i):
    b = []
    for _ in range(32):
        b.append(chr(i & 0xff))
        i >>= 8
    b.reverse()
    return "".join(b)

def test_mining_basic():
    easy_difficulty = int_to_bytes(2**256 - 1)
    assert easy_difficulty.encode('hex') == 'f' * 64
    cache = pyethash.mkcache_bytes(
              1024,
              "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
    full_size = 1024 * 32
    header = "~~~~~X~~~~~~~~~~~~~~~~~~~~~~~~~~"
    dataset = pyethash.calc_dataset_bytes(full_size, cache)
    # Check type of outputs
    assert type(pyethash.mine(dataset,header,easy_difficulty)) == dict
    assert type(pyethash.mine(dataset,header,easy_difficulty)["nonce"]) == long
    assert type(pyethash.mine(dataset,header,easy_difficulty)["mix digest"]) == str
    assert type(pyethash.mine(dataset,header,easy_difficulty)["result"]) == str

def test_mining_doesnt_always_return_the_same_value():
    easy_difficulty1 = int_to_bytes(int(2**256 * 0.999))
    # 1 in 1000 difficulty
    easy_difficulty2 = int_to_bytes(int(2**256 * 0.001))
    assert easy_difficulty1 != easy_difficulty2
    cache = pyethash.mkcache_bytes(
              1024,
              "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
    full_size = 1024 * 32
    header = "~~~~~X~~~~~~~~~~~~~~~~~~~~~~~~~~"
    dataset = pyethash.calc_dataset_bytes(full_size, cache)
    # Check type of outputs
    assert pyethash.mine(dataset, header, easy_difficulty1)['nonce'] != pyethash.mine(dataset, header, easy_difficulty2)['nonce']

def test_get_seedhash():
    assert pyethash.get_seedhash(0).encode('hex') == '0' * 64
    import hashlib, sha3
    expected = pyethash.get_seedhash(0)
    #print "checking seed hashes:",
    for i in range(0, 30000*2048, 30000):
        #print i // 30000,
        assert pyethash.get_seedhash(i) == expected
        expected = hashlib.sha3_256(expected).digest()
