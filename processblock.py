from transactions import Transaction
from blocks import Block

scriptcode_map = {
    0x00: 'STOP',   
    0x10: 'ADD',
    0x11: 'SUB',
    0x12: 'MUL',
    0x13: 'DIV',
    0x14: 'SDIV',
    0x15: 'MOD',
    0x16: 'SMOD',
    0x17: 'EXP',
    0x20: 'LT',
    0x21: 'LE',
    0x22: 'GT',
    0x23: 'GE',
    0x24: 'EQ',
    0x25: 'NEG',
    0x26: 'NOT',
    0x30: 'SHA256',
    0x31: 'RIPEMD-160',
    0x32: 'ECMUL',
    0x33: 'ECADD',
    0x34: 'SIGN',
    0x35: 'RECOVER',
    0x40: 'COPY',
    0x41: 'STORE',
    0x42: 'LD',
    0x43: 'SET',
    0x50: 'JMP',
    0x51: 'JMPI',
    0x52: 'IND',
    0x60: 'EXTRO',
    0x61: 'BALANCE',
    0x70: 'MKTX',
    0x71: 'RAWTX',
    0x80: 'DATA',
    0x81: 'DATAN',
    0x90: 'MYADDRESS'
}

fees = {
    'stepfee': 2**60 * 8192,
    'txfee': 2**60 * 524288,
    'memoryfee': 2**60 * 262144
}

def eval_tx(block):
    tx = block.transactions.pop(0)
    oldbalance = block.get_balance(tx.from)
    debit = tx.value + tx.fee
    if tx.to == '':
        debit += fees['memoryfee'] * len(filter(lambda x:x > 0,tx.data))
    if oldbalance < debit:
        return
    block.update_balance(tx.from,oldbalance - debit)
    if tx.to == '':
        mk_contract(block,tx) #todo: continue here
    else:
        block.update_balance(tx.to,block.get_balance(tx.to) + tx.value)

def mk_contract(block,tx):
    cdata = tx.data
    # todo: continue here


def eval_contract(block,tx):
    address = tx.to
    # Initialize registers
    reg = [0] * 256
    reg[0] = decode(tx.from,16)
    reg[1] = decode(tx.to,16)
    reg[2] = tx.value
    reg[3] = tx.fee
    index = 0
    stepcounter = 0
    def monop(code,f):
        reg[code[2]] = f(reg[code[1]])
    def binop(code,f):
        reg[code[3]] = f(reg[code[1]],reg[code[2]])
    while 1:
        # Calculate fee
        totalfee = 0
        stepcounter += 1
        if stepcounter > 16:
            totalfee += fees.get("stepfee")
        val_at_index = decode(block.get_contract_state(address,encode(index,256,32)),256)
        code = [ int(val_at_index / 256**i) % 256 for i in range(6) ]
        c = scriptcode_map[code[0]]
        if c == 'STORE':
            existing = block.get_contract_state(address,code[2])
            if reg[code[1]] != 0: fee += fees["MEMORYFEE"]
            if existing: fee -= fees["MEMORYFEE"]
        contractbalance = block.get_balance(address)
        # If we can't pay the fee...
        if fee > contractbalance:
            return state
        # Otherwise, pay it
        block.set_balance(address,contractbalance - fee)

        if c == 'STOP':
            break
        elif c == 'ADD':
            reg[code[3]] = (reg[code[1]] + reg[code[2]]) % 2**256
        elif c == 'MUL':
            reg[code[3]] = (reg[code[1]] * reg[code[2]]) % 2**256
        elif c == 'SUB':
            reg[code[3]] = (reg[code[1]] + 2**256 - reg[code[2]]) % 2**256
        elif c == 'DIV':
            reg[code[3]] = int(reg[code[1]] / reg[code[2]])
        elif c == 'SDIV':
            sign = 1
            sign *= (1 if reg[code[1]] < 2**255 else -1)
            sign *= (1 if reg[code[2]] < 2**255 else -1)
            x = reg[code[1]] if reg[code[1]] < 2**255 else 2**256 - reg[code[1]]
            y = reg[code[2]] if reg[code[2]] < 2**255 else 2**256 - reg[code[2]]
            z = int(x/y)
            reg[code[3]] = z if sign = 1 else 2**256 - z
        elif code == 'MOD':
            reg[code[3]] = reg[code[1]] % reg[code[2]]
        elif code == 'SMOD':
            sign = 1
            sign *= (1 if reg[code[1]] < 2**255 else -1)
            sign *= (1 if reg[code[2]] < 2**255 else -1)
            x = reg[code[1]] if reg[code[1]] < 2**255 else 2**256 - reg[code[1]]
            y = reg[code[2]] if reg[code[2]] < 2**255 else 2**256 - reg[code[2]]
            z = x%y
            reg[code[3]] = z if sign = 1 else 2**256 - z
        elif code == 'EXP':
            reg[code[3]] = pow(reg[code[1]],reg[code[2]],2**256)
        elif code == 'NEG':
            reg[code[2]] = 2**256 - reg[code[1]]
        elif code == 'LT':
            reg[code[3]] = 1 if reg[code[1]] < reg[code[2]] else 0
        elif code == 'LE':
            reg[code[3]] = 1 if reg[code[1]] <= reg[code[2]] else 0
        elif code == 'GT':
            reg[code[3]] = 1 if reg[code[1]] > reg[code[2]] else 0
        elif code == 'GE':
            reg[code[3]] = 1 if reg[code[1]] >= reg[code[2]] else 0
        elif code == 'EQ':
            reg[code[3]] = 1 if reg[code[1]] == reg[code[2]] else 0
        elif code == 'NOT':
            reg[code[2]] = 1 if reg[code[1]] == 0 else 0
        elif code == 'SHA256':
            inp = encode(reg[code[1]],256,32)
            reg[code[2]] = decode(hashlib.sha256(inp).digest(),256)
        elif code == 'RIPEMD-160':
            inp = encode(reg[code[1]],256,32)
            reg[code[2]] = decode(hashlib.new('ripemd160',inp).digest(),256)
        elif code == 'ECMUL':
            pt = (reg[code[1]],reg[code[2]])
            # Point at infinity
            if pt[0] == 0 and pt[1] == 0:
                reg[code[4]], reg[code[5]] = 0,0
            # Point not on curve, coerce to infinity
            elif (pt[0] ** 3 + 7 - pt[1] ** 2) % N != 0:
                reg[code[4]], reg[code[5]] = 0,0
            # Legitimate point
            else:
                pt2 = base10_multiply(pt,reg[code[3]])
                reg[code[4]], reg[code[5]] = pt2[0], pt2[1]
        elif code == 'ECADD':
            pt1 = (reg[code[1]],reg[code[2]])
            pt2 = (reg[code[3]],reg[code[4]])
            if (pt1[0] ** 3 + 7 - pt1[1] ** 2) % N != 0:
                reg[code[5]], reg[code[6]] = 0,0
            elif (pt2[0] ** 3 + 7 - pt2[1] ** 2) % N != 0:
                reg[code[5]], reg[code[6]] = 0,0
            else:
                pt3 = base10_add(pt1,pt2)
                reg[code[5]], reg[code[6]] = pt3[0], pt3[1]
        elif code == 'SIGN':
            reg[code[3]], reg[code[4]], reg[code[5]] = ecdsa_raw_sign(reg[code[1]],reg[code[2]])
        elif code == 'RECOVER':
            pt = ecdsa_raw_recover((reg[code[2]],reg[code[3]],reg[code[4]]),reg[code[1]])
            reg[code[5]] = pt[0]
            reg[code[6]] = pt[1]
        elif code == 'COPY':
            reg[code[2]] = reg[code[1]]
        elif code == 'STORE':
            block.update_contract_state(address,encode(reg[code[2]],256,32),reg[code[1]])
        elif code == 'LD':
            reg[code[2]] = block.get_contract_state(address,encode(reg[code[1]],256,32))
        elif code == 'SET':
            reg[code[1]] = (code[2] + 256 * code[3] + 65536 * code[4] + 16777216 * code[5]) * 2**code[6] % 2**256
        elif code == 'JMP':
            index = reg[code[1]]
        elif code == 'JMPI':
            if reg[code[1]]: index = reg[code[2]]
        elif code == 'IND':
            reg[code[1]] = index
        elif code == 'EXTRO':
            address = encode(reg[code[1]] % 2**160,256,20)
            field = encode(reg[code[2]]
            reg[code[3]] = block.get_contract_state(address,field)
        elif code == 'BALANCE':
            address = encode(reg[code[1]] % 2**160,256,20)
            reg[code[2]] = block.get_balance(address)
        elif code == 'MKTX':
            to = encode(reg[code[1]],256,32)
            value = reg[code[2]]
            fee = reg[code[3]]
            if (value + fee) > block.get_balance(address):
                pass
            else:
                datan = reg[code[4]]
                data = []
                for i in range(datan):
                    ind = encode((reg[code[5]] + i) % 2**256,256,32)
                    data.append(block.get_contract_state(address,ind))
                tx = Transaction(to,value,fee,data)
                tx.from = address
                block.transactions.append(tx)
        elif code == 'DATA':
            reg[code[2]] = tx.data[reg[code[1]]]
        elif code == 'DATAN':
            reg[code[1]] = len(tx.data)
        elif code == 'MYADDRESS':
            reg[code[1]] = address
        elif code == 'SUICIDE':
            sz = block.get_contract_size(address)
            negfee = sz * fees["memoryfee"]
            toaddress = encode(reg[code[1]],256,32)
            block.update_balance(toaddress,block.get_balance(toaddress) + negfee)
            block.update_contract(address,0)
            break
