{
  // TODO: return vals, storage
  function basic() {
    mstore(0x80, 0x42)
  }
  
  function func1(val) {
    mstore(0xa0, val)
  }
  
  function func2(val1, val2, str1) {
    mstore(0xc0, add(val1, val2))
    mstore(0xe0, str1)
  }

  function main() {
    basic()
    func1(0x32)
  }
}
