def bitstring_to_bytes(bitstring):
    # 创建一个长度为 (len(value) // 8) + 1 的数组，用于存储字节值，初始化为 0
    array = [0] * ((len(bitstring) // 8) + 1)

    # 遍历位数组 bitstring，将每个位的值设置到相应的字节中
    for i in range(len(bitstring)):
        # 计算当前位在字节数组中的索引
        byte_index = i // 8

        # 将当前位的值按位或到相应的字节上，通过左移 (i % 8) 位来设置对应的位置
        array[byte_index] |= int(bitstring[i]) << (i % 8)
        print(array)

    # 将最后一个字节的最高位设置为 1
    array[len(bitstring) // 8] |= 1 << (len(bitstring) % 8)
    print(array)

    # 将字节数组转换为 bytes 类型并返回
    return bytes(array)


bitstring_to_bytes("111100001011")