# Security Policy

## Supported Versions

Please see [Releases](https://github.com/ethereum/go-ethereum/releases). We recommend using the [most recently released version](https://github.com/ethereum/go-ethereum/releases/latest).

## Audit reports

Audit reports are published in the `docs` folder: https://github.com/ethereum/go-ethereum/tree/master/docs/audits 

| Scope | Date | Report Link |
| ------- | ------- | ----------- |
| `geth` | 20170425 | [pdf](https://github.com/ethereum/go-ethereum/blob/master/docs/audits/2017-04-25_Geth-audit_Truesec.pdf) |
| `clef` | 20180914 | [pdf](https://github.com/ethereum/go-ethereum/blob/master/docs/audits/2018-09-14_Clef-audit_NCC.pdf) |
| `Discv5` | 20191015 | [pdf](https://github.com/ethereum/go-ethereum/blob/master/docs/audits/2019-10-15_Discv5_audit_LeastAuthority.pdf) |
| `Discv5` | 20200124 | [pdf](https://github.com/ethereum/go-ethereum/blob/master/docs/audits/2020-01-24_DiscV5_audit_Cure53.pdf) |

## Reporting a Vulnerability

**Please do not file a public ticket** mentioning the vulnerability.

To find out how to disclose a vulnerability in Ethereum visit [https://bounty.ethereum.org](https://bounty.ethereum.org) or email bounty@ethereum.org. Please read the [disclosure page](https://github.com/ethereum/go-ethereum/security/advisories?state=published) for more information about publicly disclosed security vulnerabilities.

Use the built-in `geth version-check` feature to check whether the software is affected by any known vulnerability. This command will fetch the latest [`vulnerabilities.json`](https://geth.ethereum.org/docs/vulnerabilities/vulnerabilities.json) file which contains known security vulnerabilities concerning `geth`, and cross-check the data against its own version number.

The following key may be used to communicate sensitive information to developers.

Fingerprint: `AE96 ED96 9E47 9B00 84F3 E17F E88D 3334 FA5F 6A0A`

```
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
6F7hEeTW+gnXEI2gPjfb5w==
=b5eA
-----END PGP PUBLIC KEY BLOCK-----
```
