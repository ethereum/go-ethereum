/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
export var FMT_NUMBER;
(function (FMT_NUMBER) {
    FMT_NUMBER["NUMBER"] = "NUMBER_NUMBER";
    FMT_NUMBER["HEX"] = "NUMBER_HEX";
    FMT_NUMBER["STR"] = "NUMBER_STR";
    FMT_NUMBER["BIGINT"] = "NUMBER_BIGINT";
})(FMT_NUMBER || (FMT_NUMBER = {}));
export var FMT_BYTES;
(function (FMT_BYTES) {
    FMT_BYTES["HEX"] = "BYTES_HEX";
    FMT_BYTES["UINT8ARRAY"] = "BYTES_UINT8ARRAY";
})(FMT_BYTES || (FMT_BYTES = {}));
export const DEFAULT_RETURN_FORMAT = {
    number: FMT_NUMBER.BIGINT,
    bytes: FMT_BYTES.HEX,
};
export const ETH_DATA_FORMAT = { number: FMT_NUMBER.HEX, bytes: FMT_BYTES.HEX };
//# sourceMappingURL=data_format_types.js.map