export const getChecksum = (contentMD5: string) => {
  // based on https://github.com/ethereum/go-ethereum/blob/7519505d6fbd1fd29a8595aafbf880a04fb3e7e1/downloads.html#L318
  return Buffer.from(contentMD5, 'base64')
    .toString('binary')
    .split('')
    .map(function (char) {
      return ('0' + char.charCodeAt(0).toString(16)).slice(-2);
    })
    .join('');
};
