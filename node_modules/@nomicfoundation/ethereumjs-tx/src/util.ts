import { bytesToHex, hexToBytes, setLengthLeft } from '@nomicfoundation/ethereumjs-util'

import { isAccessList } from './types.js'

import type { AccessList, AccessListBytes, AccessListItem, TransactionType } from './types.js'
import type { Common } from '@nomicfoundation/ethereumjs-common'

export function checkMaxInitCodeSize(common: Common, length: number) {
  const maxInitCodeSize = common.param('vm', 'maxInitCodeSize')
  if (maxInitCodeSize && BigInt(length) > maxInitCodeSize) {
    throw new Error(
      `the initcode size of this transaction is too large: it is ${length} while the max is ${common.param(
        'vm',
        'maxInitCodeSize'
      )}`
    )
  }
}

export class AccessLists {
  public static getAccessListData(accessList: AccessListBytes | AccessList) {
    let AccessListJSON
    let bufferAccessList
    if (isAccessList(accessList)) {
      AccessListJSON = accessList
      const newAccessList: AccessListBytes = []

      for (let i = 0; i < accessList.length; i++) {
        const item: AccessListItem = accessList[i]
        const addressBytes = hexToBytes(item.address)
        const storageItems: Uint8Array[] = []
        for (let index = 0; index < item.storageKeys.length; index++) {
          storageItems.push(hexToBytes(item.storageKeys[index]))
        }
        newAccessList.push([addressBytes, storageItems])
      }
      bufferAccessList = newAccessList
    } else {
      bufferAccessList = accessList ?? []
      // build the JSON
      const json: AccessList = []
      for (let i = 0; i < bufferAccessList.length; i++) {
        const data = bufferAccessList[i]
        const address = bytesToHex(data[0])
        const storageKeys: string[] = []
        for (let item = 0; item < data[1].length; item++) {
          storageKeys.push(bytesToHex(data[1][item]))
        }
        const jsonItem: AccessListItem = {
          address,
          storageKeys,
        }
        json.push(jsonItem)
      }
      AccessListJSON = json
    }

    return {
      AccessListJSON,
      accessList: bufferAccessList,
    }
  }

  public static verifyAccessList(accessList: AccessListBytes) {
    for (let key = 0; key < accessList.length; key++) {
      const accessListItem = accessList[key]
      const address = accessListItem[0]
      const storageSlots = accessListItem[1]
      if ((<any>accessListItem)[2] !== undefined) {
        throw new Error(
          'Access list item cannot have 3 elements. It can only have an address, and an array of storage slots.'
        )
      }
      if (address.length !== 20) {
        throw new Error('Invalid EIP-2930 transaction: address length should be 20 bytes')
      }
      for (let storageSlot = 0; storageSlot < storageSlots.length; storageSlot++) {
        if (storageSlots[storageSlot].length !== 32) {
          throw new Error('Invalid EIP-2930 transaction: storage slot length should be 32 bytes')
        }
      }
    }
  }

  public static getAccessListJSON(accessList: AccessListBytes) {
    const accessListJSON = []
    for (let index = 0; index < accessList.length; index++) {
      const item: any = accessList[index]
      const JSONItem: any = {
        address: bytesToHex(setLengthLeft(item[0], 20)),
        storageKeys: [],
      }
      const storageSlots: Uint8Array[] = item[1]
      for (let slot = 0; slot < storageSlots.length; slot++) {
        const storageSlot = storageSlots[slot]
        JSONItem.storageKeys.push(bytesToHex(setLengthLeft(storageSlot, 32)))
      }
      accessListJSON.push(JSONItem)
    }
    return accessListJSON
  }

  public static getDataFeeEIP2930(accessList: AccessListBytes, common: Common): number {
    const accessListStorageKeyCost = common.param('gasPrices', 'accessListStorageKeyCost')
    const accessListAddressCost = common.param('gasPrices', 'accessListAddressCost')

    let slots = 0
    for (let index = 0; index < accessList.length; index++) {
      const item = accessList[index]
      const storageSlots = item[1]
      slots += storageSlots.length
    }

    const addresses = accessList.length
    return addresses * Number(accessListAddressCost) + slots * Number(accessListStorageKeyCost)
  }
}

export function txTypeBytes(txType: TransactionType): Uint8Array {
  return hexToBytes('0x' + txType.toString(16).padStart(2, '0'))
}
