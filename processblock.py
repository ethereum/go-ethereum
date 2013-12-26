from transactions import Transaction
from blocks import Block
import time
import sys
import rlp

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
    0x18: 'NEG',
    0x20: 'LT',
    0x21: 'LE',
    0x22: 'GT',
    0x23: 'GE',
    0x24: 'EQ',
    0x25: 'NOT',
    0x30: 'SHA256',
    0x31: 'RIPEMD-160',
    0x32: 'ECMUL',
    0x33: 'ECADD',
    0x34: 'ECSIGN',
    0x35: 'ECRECOVER',
    0x40: 'COPY',
    0x41: 'STORE',
    0x42: 'LOAD',
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
    0x90: 'MYADDRESS',
    0x91: 'BLKHASH',
    0xff: 'SUICIDE'
}

params = {
    'stepfee': 2**64 / 64,
    'txfee': 2**64,
    'newcontractfee': 2**64,
    'memoryfee': 2**64 / 4,
    'datafee': 2**64 / 16,
    'cryptofee': 2**64 / 16,
    'extrofee': 2**64 / 16,
    'blocktime': 60,
    'period_1_reward': 2**80 * 1024,
    'period_1_duration': 57600,
    'period_2_reward': 2**80 * 512,
    'period_2_duration': 57600,
    'period_3_reward': 2**80 * 256,
    'period_3_duration': 57600,
    'period_4_reward': 2**80 * 128
}

def process_transactions(block,transactions):
    while len(transactions) > 0:
        tx = transactions.pop(0)
        enc = (tx.value, tx.fee, tx.sender.encode('hex'), tx.to.encode('hex'))
        sys.stderr.write("Attempting to send %d plus fee %d from %s to %s\n" % enc)
        # Grab data about sender, recipient and miner
        sdata = rlp.decode(block.state.get(tx.sender)) or [0,0,0]
        tdata = rlp.decode(block.state.get(tx.to)) or [0,0,0]
        # Calculate fee
        if tx.to == '\x00'*20:
            fee = params['newcontractfee'] + len(tx.data) * params['memoryfee']
        else:
            fee = params['txfee']
        # Insufficient fee, do nothing
        if fee > tx.fee:
            sys.stderr.write("Insufficient fee\n")
            continue
        # Too much data, do nothing
        if len(tx.data) > 256:
            sys.stderr.write("Too many data items\n")
            continue
        if not sdata or sdata[1] < tx.value + tx.fee:
            sys.stderr.write("Insufficient funds to send fee\n")
            continue
        elif tx.nonce != sdata[2] and sdata[0] == 0:
            sys.stderr.write("Bad nonce\n")
            continue
        # Try to send the tx
        if sdata[0] == 0: sdata[2] += 1
        sdata[1] -= (tx.value + tx.fee)
        block.reward += tx.fee
        if tx.to != '':
            tdata[1] += tx.value
        else:
            addr = tx.hash()[-20:]
            adata = rlp.decode(block.state.get(addr))
            if adata[2] != '':
                sys.stderr.write("Contract already exists\n")
                continue
            block.state.update(addr,rlp.encode([1,tx.value,'']))
            contract = block.get_contract(addr)
            for i in range(len(tx.data)):
                contract.update(encode(i,256,32),tx.data[i])
            block.update_contract(addr)
        print sdata, tdata
        block.state.update(tx.sender,rlp.encode(sdata))
        block.state.update(tx.to,rlp.encode(tdata))
        # Evaluate contract if applicable
        if tdata[0] == 1:
            eval_contract(block,transactions,tx)
        sys.stderr.write("tx processed\n")

def eval(block,transactions,timestamp,coinbase):
    h = block.hash()
    # Process all transactions
    process_transactions(block,transactions)
    # Pay miner fee
    miner_state = rlp.decode(block.state.get(block.coinbase)) or [0,0,0]
    block.number += 1
    reward = 0
    if block.number < params['period_1_duration']:
        reward = params['period_1_reward']
    elif block.number < params['period_2_duration']:
        reward = params['period_2_reward']
    elif block.number < params['period_3_duration']:
        reward = params['period_3_reward']
    else:
        reward = params['period_4_reward']
    print reward
    miner_state[1] += reward + block.reward
    for uncle in block.uncles:
        sib_miner_state = rlp_decode(block.state.get(uncle[3]))
        sib_miner_state[1] = encode(decode(sib_miner_state[1],256)+reward*7/8,256)
        block.state.update(uncle[3],sib_miner_state)
        miner_state[1] += reward/8
    block.state.update(block.coinbase,rlp.encode(miner_state))
    # Check timestamp
    if timestamp < block.timestamp or timestamp > int(time.time()) + 3600:
        raise Exception("timestamp not in valid range!")
    # Update difficulty
    if timestamp >= block.timestamp + 42:
        block.difficulty += int(block.difficulty / 1024)
    else:
        block.difficulty -= int(block.difficulty / 1024)
    block.prevhash = h
    block.coinbase = coinbase
    block.transactions = []
    block.uncles = []
    return block

def eval_contract(block,transaction_list,tx):
    sys.stderr.write("evaluating contract\n")
    address = tx.to
    # Initialize registers
    reg = [0] * 256
    reg[0] = decode(tx.sender,256)
    reg[1] = decode(tx.to,256)
    reg[2] = tx.value
    reg[3] = tx.fee
    index = 0
    stepcounter = 0
    contract = block.get_contract(address)
    if not contract:
        return
    while 1:
        # Convert the data item into a code piece
        val_at_index = decode(contract.get(encode(index,256,32)),256)
        code = [ int(val_at_index / (256**i)) % 256 for i in range(6) ]
        code[0] = scriptcode_map.get(code[0],'INVALID')
        sys.stderr.write("Evaluating: "+ str(code)+"\n")
        # Invalid code instruction or STOP code stops execution sans fee
        if val_at_index >= 256**6 or code[0] in ['STOP','INVALID']:
            sys.stderr.write("stop code, exiting\n")
            break
        # Calculate fee
        minerfee = 0
        nullfee = 0
        stepcounter += 1
        if stepcounter > 16:
            minerfee += params["stepfee"]
        c = scriptcode_map[code[0]]
        if c in ['STORE','LOAD']:
            minerfee += params["datafee"]
        if c in ['EXTRO','BALANCE']:
            minerfee += params["extrofee"]
        if c in ['SHA256','RIPEMD-160','ECMUL','ECADD','ECSIGN','ECRECOVER']:
            minerfee += params["cryptofee"]
        if c == 'STORE':
            existing = block.get_contract_state(address,code[2])
            if reg[code[1]] != 0: nullfee += params["memoryfee"]
            if existing: nullfee -= params["memoryfee"]

        # If we can't pay the fee, break, otherwise pay it
        if block.get_balance(address) < minerfee + nullfee:
            sys.stderr.write("insufficient fee, exiting\n")
            break
        block.set_balance(address,block.get_balance(address) - nullfee - minerfee)
        block.reward += minerfee
        sys.stderr.write("evaluating operation\n") 
        # Evaluate operations
        if c == 'ADD':
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
            reg[code[3]] = z if sign == 1 else 2**256 - z
        elif code == 'MOD':
            reg[code[3]] = reg[code[1]] % reg[code[2]]
        elif code == 'SMOD':
            sign = 1
            sign *= (1 if reg[code[1]] < 2**255 else -1)
            sign *= (1 if reg[code[2]] < 2**255 else -1)
            x = reg[code[1]] if reg[code[1]] < 2**255 else 2**256 - reg[code[1]]
            y = reg[code[2]] if reg[code[2]] < 2**255 else 2**256 - reg[code[2]]
            z = x%y
            reg[code[3]] = z if sign == 1 else 2**256 - z
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
            # Invalid point 1
            if (pt1[0] ** 3 + 7 - pt1[1] ** 2) % N != 0:
                reg[code[5]], reg[code[6]] = 0,0
            # Invalid point 2
            elif (pt2[0] ** 3 + 7 - pt2[1] ** 2) % N != 0:
                reg[code[5]], reg[code[6]] = 0,0
            # Legitimate points
            else:
                pt3 = base10_add(pt1,pt2)
                reg[code[5]], reg[code[6]] = pt3[0], pt3[1]
        elif code == 'ECSIGN':
            reg[code[3]], reg[code[4]], reg[code[5]] = ecdsa_raw_sign(reg[code[1]],reg[code[2]])
        elif code == 'ECRECOVER':
            pt = ecdsa_raw_recover((reg[code[2]],reg[code[3]],reg[code[4]]),reg[code[1]])
            reg[code[5]] = pt[0]
            reg[code[6]] = pt[1]
        elif code == 'COPY':
            reg[code[2]] = reg[code[1]]
        elif code == 'STORE':
            contract.update(encode(reg[code[2]],256,32),reg[code[1]])
        elif code == 'LOAD':
            reg[code[2]] = contract.get(encode(reg[code[1]],256,32))
        elif code == 'SET':
            reg[code[1]] = (code[2] + 256 * code[3] + 65536 * code[4] + 16777216 * code[5]) * 2**code[6] % 2**256
        elif code == 'JMP':
            index = reg[code[1]]
        elif code == 'JMPI':
            if reg[code[1]]: index = reg[code[2]]
        elif code == 'IND':
            reg[code[1]] = index
        elif code == 'EXTRO':
            if reg[code[1]] >= 2**160:
                reg[code[3]] = 0
            else:
                address = encode(reg[code[1]],256,20)
                field = encode(reg[code[2]])
                reg[code[3]] = block.get_contract(address).get(field)
        elif code == 'BALANCE':
            if reg[code[1]] >= 2**160:
                reg[code[2]] = 0
            else:
                address = encode(reg[code[1]],256,20)
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
                    data.append(contract.get(ind))
                tx = Transaction(0,to,value,fee,data)
                tx.sender = address
                transaction_list.insert(0,tx)
        elif code == 'DATA':
            reg[code[2]] = tx.data[reg[code[1]]]
        elif code == 'DATAN':
            reg[code[1]] = len(tx.data)
        elif code == 'MYADDRESS':
            reg[code[1]] = address
        elif code == 'BLKHASH':
            reg[code[1]] = decode(block.hash())
        elif code == 'SUICIDE':
            sz = contract.get_size()
            negfee = -sz * params["memoryfee"]
            toaddress = encode(reg[code[1]],256,20)
            block.pay_fee(roaddress,negfee,False)
            contract.root = ''
            break
    block.update_contract(address,contract)
