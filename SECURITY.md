# Security Policy

## Supported Versions

Please see [Releases](https://github.com/xpaymentsorg/go-xpayments/releases). We recommend using the [most recently released version](https://github.com/xpaymentsorg/go-xpayments/releases/latest).

## Audit reports

Audit reports are published in the `docs` folder: https://github.com/xpaymentsorg/go-xpayments/tree/master/docs/audits 

| Scope | Date | Report Link |
| ------- | ------- | ----------- |
| `gpay` | 00000000 | [pdf](https://github.com/xpaymentsorg/go-xpayments/blob/master/docs/audits/) |
| `clef` | 00000000 | [pdf](https://github.com/xpaymentsorg/go-xpayments/blob/master/docs/audits/) |
| `Discv5` | 00000000 | [pdf](https://github.com/xpaymentsorg/go-xpayments/blob/master/docs/audits/) |
| `Discv5` | 00000000 | [pdf](https://github.com/xpaymentsorg/go-xpayments/blob/master/docs/audits/) |

## Reporting a Vulnerability

**Please do not file a public ticket** mentioning the vulnerability.

To find out how to disclose a vulnerability in xPayments visit [https://bounty.xpayments.org](https://bounty.xpayments.org) or email bounty@xpayments.org. Please read the [disclosure page](https://github.com/xpaymentsorg/go-xpayments/security/advisories?state=published) for more information about publicly disclosed security vulnerabilities.

Use the built-in `gpay version-check` feature to check whether the software is affected by any known vulnerability. This command will fetch the latest [`vulnerabilities.json`](https://gpay.xpayments.org/docs/vulnerabilities/vulnerabilities.json) file which contains known security vulnerabilities concerning `gpay`, and cross-check the data against its own version number.

The following key may be used to communicate sensitive information to developers.

Fingerprint: `0000 0000 0000 0000 0000 0000 0000 0000 0000 0000`

```
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: SKS 1.1.6
Comment: Hostname: pgp.mit.edu


-----END PGP PUBLIC KEY BLOCK------
```
