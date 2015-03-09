import pyethash
from random import randint

def test_get_cache_size_not_None():
    for _ in range(100):
        block_num = randint(0,12456789)
        out = pyethash.core.get_cache_size(block_num)
        assert out != None

def test_get_full_size_not_None():
    for _ in range(100):
        block_num = randint(0,12456789)
        out = pyethash.core.get_full_size(block_num)
        assert out != None

def test_get_cache_size_based_on_EPOCH():
    for _ in range(100):
        block_num = randint(0,12456789)
        out1 = pyethash.core.get_cache_size(block_num)
        out2 = pyethash.core.get_cache_size((block_num // pyethash.EPOCH_LENGTH) * pyethash.EPOCH_LENGTH)
        assert out1 == out2

def test_get_full_size_based_on_EPOCH():
    for _ in range(100):
        block_num = randint(0,12456789)
        out1 = pyethash.core.get_full_size(block_num)
        out2 = pyethash.core.get_full_size((block_num // pyethash.EPOCH_LENGTH) * pyethash.EPOCH_LENGTH)
        assert out1 == out2

#def test_get_params_based_on_EPOCH():
#    block_num = 123456
#    out1 = pyethash.core.get_params(block_num)
#    out2 = pyethash.core.get_params((block_num // pyethash.EPOCH_LENGTH) * pyethash.EPOCH_LENGTH)
#    assert out1["DAG Size"] == out2["DAG Size"]
#    assert out1["Cache Size"] == out2["Cache Size"]
#
#def test_get_params_returns_different_values_based_on_different_block_input():
#    out1 = pyethash.core.get_params(123456)
#    out2 = pyethash.core.get_params(12345)
#    assert out1["DAG Size"] != out2["DAG Size"]
#    assert out1["Cache Size"] != out2["Cache Size"]
#
#def test_get_cache_smoke_test():
#    params = pyethash.core.get_params(123456)
#    assert pyethash.core.mkcache(params, "~~~~") != None
