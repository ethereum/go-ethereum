var addressName = {"0x12378912345789": "Gav", "0x57835893478594739854": "Jeff"};
var nameAddress = {};

for (var prop in addressName) {
  if (addressName.hasOwnProperty(prop)) {
    nameAddress[addressName[prop]]  = prop;
  }
}

var local = {
  addressBook:{
    byName: addressName,
    byAddress: nameAddress
  }
};

if (typeof(module) !== "undefined")
    module.exports = local;
