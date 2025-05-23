# Change Log

## v0.8.0 / 2018-08-05
### Added
- TypeScript definitions.

### Changed
- throw error if update after finalize

## v0.7.0 / 2017-12-01
### Added
- AMD support.
- support for web worker. #13

### Changed
- throw error if input type is incorrect when cSHAKE and KMAC.
- freeze hash after finalize.

## v0.6.1 / 2017-07-03
### Fixed
- Typo on variable kmac_256 type definition. #12

## v0.6.0 / 2017-06-15
### Added
- cSHAKE method.
- KMAC method.
- alias methods without underscore like shake128, keccak512.

### Changed
- throw error if input type is incorrect.

## v0.5.7 / 2016-12-30
### Fixed
- ArrayBuffer detection in old browsers.

## v0.5.6 / 2016-12-29
### Fixed
- ArrayBuffer dosen't work in Webpack.

## v0.5.5 / 2016-09-26
### Added
- TypeScript support.
- ArrayBuffer method.

### Deprecated
- Buffer method.

## v0.5.4 / 2016-09-12
### Fixed
- CommonJS detection.

## v0.5.3 / 2016-09-08
### Added
- Some missing files to npm package.

## v0.5.2 / 2016-06-06
### Fixed
- Shake output incorrect in the special length.

## v0.5.1 / 2015-10-27
### Fixed
- Version in package.json and bower.json.

## v0.5.0 / 2015-09-23
### Added
- Hash object with create/update interface.

## v0.4.1 / 2015-09-18
### Added
- Integer array output.

### Fixed
- Shake output incorrect when it's greater than 1088.

## v0.4.0 / 2015-09-17
### Added
- ArrayBuffer output.
- Shake alogirthms.

## v0.3.1 / 2015-05-22
### Fixed
- Some bugs.

## v0.3.0 / 2015-05-21
### Added
- Integer array input.
- ArrayBuffer input.

## v0.2.0 / 2015-04-04
### Added
- NIST's May 2014 SHA-3 version.

### Changed
- Rename original methods to keccak.

## v0.1.2 / 2015-02-27
### Changed
- Improve performance.

## v0.1.1 / 2015-02-26
### Changed
- Improve performance.

## v0.1.0 / 2015-02-23
### Added
- First version implementation.
