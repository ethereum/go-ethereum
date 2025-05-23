import { bytesToHex } from 'web3-utils';
import { setLengthLeft, toUint8Array } from '../common/utils.js';
import { isAccessList } from './types.js';
export const checkMaxInitCodeSize = (common, length) => {
    const maxInitCodeSize = common.param('vm', 'maxInitCodeSize');
    if (maxInitCodeSize && BigInt(length) > maxInitCodeSize) {
        throw new Error(`the initcode size of this transaction is too large: it is ${length} while the max is ${common.param('vm', 'maxInitCodeSize')}`);
    }
};
export const getAccessListData = (accessList) => {
    let AccessListJSON;
    let uint8arrayAccessList;
    if (isAccessList(accessList)) {
        AccessListJSON = accessList;
        const newAccessList = [];
        // eslint-disable-next-line @typescript-eslint/prefer-for-of
        for (let i = 0; i < accessList.length; i += 1) {
            const item = accessList[i];
            const addressBytes = toUint8Array(item.address);
            const storageItems = [];
            // eslint-disable-next-line @typescript-eslint/prefer-for-of
            for (let index = 0; index < item.storageKeys.length; index += 1) {
                storageItems.push(toUint8Array(item.storageKeys[index]));
            }
            newAccessList.push([addressBytes, storageItems]);
        }
        uint8arrayAccessList = newAccessList;
    }
    else {
        uint8arrayAccessList = accessList !== null && accessList !== void 0 ? accessList : [];
        // build the JSON
        const json = [];
        // eslint-disable-next-line @typescript-eslint/prefer-for-of
        for (let i = 0; i < uint8arrayAccessList.length; i += 1) {
            const data = uint8arrayAccessList[i];
            const address = bytesToHex(data[0]);
            const storageKeys = [];
            // eslint-disable-next-line @typescript-eslint/prefer-for-of
            for (let item = 0; item < data[1].length; item += 1) {
                storageKeys.push(bytesToHex(data[1][item]));
            }
            const jsonItem = {
                address,
                storageKeys,
            };
            json.push(jsonItem);
        }
        AccessListJSON = json;
    }
    return {
        AccessListJSON,
        accessList: uint8arrayAccessList,
    };
};
export const verifyAccessList = (accessList) => {
    // eslint-disable-next-line @typescript-eslint/prefer-for-of
    for (let key = 0; key < accessList.length; key += 1) {
        const accessListItem = accessList[key];
        const address = accessListItem[0];
        const storageSlots = accessListItem[1];
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/consistent-type-assertions
        if (accessListItem[2] !== undefined) {
            throw new Error('Access list item cannot have 3 elements. It can only have an address, and an array of storage slots.');
        }
        if (address.length !== 20) {
            throw new Error('Invalid EIP-2930 transaction: address length should be 20 bytes');
        }
        // eslint-disable-next-line @typescript-eslint/prefer-for-of
        for (let storageSlot = 0; storageSlot < storageSlots.length; storageSlot += 1) {
            if (storageSlots[storageSlot].length !== 32) {
                throw new Error('Invalid EIP-2930 transaction: storage slot length should be 32 bytes');
            }
        }
    }
};
export const getAccessListJSON = (accessList) => {
    const accessListJSON = [];
    // eslint-disable-next-line @typescript-eslint/prefer-for-of
    for (let index = 0; index < accessList.length; index += 1) {
        const item = accessList[index];
        const JSONItem = {
            address: bytesToHex(setLengthLeft(item[0], 20)),
            storageKeys: [],
        };
        // eslint-disable-next-line @typescript-eslint/prefer-optional-chain
        const storageSlots = item && item[1];
        // eslint-disable-next-line @typescript-eslint/prefer-for-of
        for (let slot = 0; slot < storageSlots.length; slot += 1) {
            const storageSlot = storageSlots[slot];
            JSONItem.storageKeys.push(bytesToHex(setLengthLeft(storageSlot, 32)));
        }
        accessListJSON.push(JSONItem);
    }
    return accessListJSON;
};
export const getDataFeeEIP2930 = (accessList, common) => {
    const accessListStorageKeyCost = common.param('gasPrices', 'accessListStorageKeyCost');
    const accessListAddressCost = common.param('gasPrices', 'accessListAddressCost');
    let slots = 0;
    // eslint-disable-next-line @typescript-eslint/prefer-for-of
    for (let index = 0; index < accessList.length; index += 1) {
        const item = accessList[index];
        const storageSlots = item[1];
        slots += storageSlots.length;
    }
    const addresses = accessList.length;
    return addresses * Number(accessListAddressCost) + slots * Number(accessListStorageKeyCost);
};
//# sourceMappingURL=utils.js.map