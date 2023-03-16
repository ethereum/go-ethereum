 fernando guadalupe mendes espinoza
 
 0x19D4F9A260AF1d7E0E99A32DBe418956af875c25
 
82789076105303867564595430104454069497307132242678459595482236645708760809276

Contract Address 0x17a009bc2a642711a2e0eb58bc760fa520f9ca91

Owner Address 0xa1fed5e23fa20d654ecb012393ad9f3c9522dcf4

Token Id	
31593...34107
Symbol	DCL-ANYMGKRP2EG|VLLGRSTS
Chain	Polygon
Last Synced	14/2/2023

{
    "id": "urn:decentraland:matic:collections-v2:0x17a009bc2a642711a2e0eb58bc760fa520f9ca91:3",
    "name": "Villager Armored Jacket [Lvl. 5]",
    "description": "DCL Wearable 1499/5000",
    "language": "en-US",
    "image": "https://peer-ec1.decentraland.org/content/contents/bafybeigmx6ieyuqok5prdwf77chz4vmggzakbjywp6kqnf64wqu74c77ha",
    "thumbnail": "https://peer-ec1.decentraland.org/content/contents/bafybeidqxyrq3c7rgwvdlfntgfl3uz7wsbz3vd7dmrwx6xvfi6w6qfmbmu",
    "attributes": [
        {
            "trait_type": "Rarity",
            "value": "rare"
        },
        {
            "trait_type": "Category",
            "value": "upper_body"
        },
        {
            "trait_type": "Tag",
            "value": "anymagik"
        },
        {
            "trait_type": "Tag",
            "value": "magika"
        },
        {
            "trait_type": "Tag",
            "value": "villager"
        },
        {
            "trait_type": "Tag",
            "value": "male"
        },
        {
            "trait_type": "Tag",
            "value": "lvl.5"
        },
        {
            "trait_type": "Tag",
            "value": "utility"
        },
        {
            "trait_type": "Tag",
            "value": "vigor"
        },
        {
            "trait_type": "Tag",
            "value": "upgradeable"
        },
        {
            "trait_type": "Tag",
            "value": "auras"
        }
    ]
}

    const web3ApiKey = 'i8tJu8H7m2gJJ57tg1DCpps7NJZBRCOGLEKWoUUhTS5EIIpCvDUElYaIeleR3W59';
    const headers = { accept: 'application/json', 'X-API-Key': web3ApiKey };
    const options = {
      method: 'GET',
      headers,
      params: { chain: '0x89' },
    };
    fetch(
      ('https://deep-index.moralis.io/api/v2/nft' +
      '/0x17a009bC2a642711A2E0eb58bC760FA520F9CA91' +
      '/315936875005671560093754083051011296956685286201647333762932934107'), 
      options
    )
    .then((res) => res.json())
    .then((data) => console.log(data))
    .catch((error) => console.error(error));
