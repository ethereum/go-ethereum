import React from 'react';
import { IconProps } from '@chakra-ui/react';
import { WindowsLogo, MacosLogo, LinuxPenguin, SourceBranch } from './components/UI/icons';

export const BORDER_WIDTH = '2px';

// internal pages
export const DOWNLOADS_PAGE = '/downloads';
export const DOCS_PAGE = '/docs';
export const FAQ_PAGE = '/docs/faq';
export const CONTRIBUTING_PAGE = `${DOCS_PAGE}/developers/contributing`;

// external links
export const ETHEREUM_ORG_URL = 'https://ethereum.org';
export const ETHEREUM_ORG_RUN_A_NODE_URL = 'https://ethereum.org/en/run-a-node/';
export const ETHEREUM_FOUNDATION_URL = 'https://ethereum.foundation';
export const GETH_REPO_URL = 'https://github.com/ethereum/go-ethereum';
export const GETH_TWITTER_URL = 'https://twitter.com/go_ethereum';
export const GETH_DISCORD_URL = 'https://discord.com/invite/nthXNEv';
export const GO_URL = 'https://go.dev/';

// analytics
export const DO_NOT_TRACK_URL =
  'http://matomo.ethereum.org/piwik/index.php?module=CoreAdminHome&action=optOut';

// Downloads
export const DEFAULT_BUILD_AMOUNT_TO_SHOW = 12;
export const DOWNLOAD_HEADER_BUTTONS: {
  [index: string]: {
    name: string;
    ariaLabel: string;
    buildURL: string;
    Svg: React.FC<IconProps>;
  };
} = {
  linuxBuild: {
    name: 'Linux',
    ariaLabel: 'Linux logo',
    Svg: LinuxPenguin,
    buildURL: ''
  },
  macOSBuild: {
    name: 'macOS',
    ariaLabel: 'macOS logo',
    Svg: MacosLogo,
    buildURL: ''
  },
  windowsBuild: {
    name: 'Windows',
    ariaLabel: 'Windows logo',
    Svg: WindowsLogo,
    buildURL: ''
  },
  sourceCode: {
    name: 'Sources',
    ariaLabel: 'Source branch logo',
    Svg: SourceBranch,
    buildURL: ''
  }
};
export const DOWNLOADS_TABLE_TABS = ['Linux', 'macOS', 'Windows', 'iOS', 'Android'];
export const DOWNLOADS_TABLE_TAB_COLUMN_HEADERS = [
  'Release',
  'Commit',
  'Kind',
  'Arch',
  'Size',
  'Published',
  'Signature',
  'Checksum (MD5)'
];
export const DOWNLOADS_OPENPGP_BUILD_HEADERS = [
  'Build Server',
  'Unique ID',
  'OpenPGP Key',
  'Fingerprint'
];
export const DOWNLOADS_OPENPGP_SIGNATURES = [
  {
    'build server': 'Android Builder',
    'unique id': 'Go Ethereum Android Builder <geth-ci@ethereum.org>',
    'openpgp key': {
      label: 'F9585DE6',
      url: 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x70AD154BF9585DE6'
    },
    fingerprint: '8272 1824 F4D7 46E0 B5A7  AB95 70AD 154B F958 5DE6'
  },
  {
    'build server': 'iOS Builder',
    'unique id': 'Go Ethereum iOS Builder <geth-ci@ethereum.org>',
    'openpgp key': {
      label: 'C2FF8BBF',
      url: 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0xF29DEFAFC2FF8BBF'
    },
    fingerprint: '70AD EB8F 3BC6 6F69 0256  4D88 F29D EFAF C2FF 8BBF'
  },
  {
    'build server': 'Linux Builder',
    'unique id': 'Go Ethereum Linux Builder <geth-ci@ethereum.org>',
    'openpgp key': {
      label: '9BA28146',
      url: 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0xA61A13569BA28146'
    },
    fingerprint: 'FDE5 A1A0 44FA 13D2 F7AD  A019 A61A 1356 9BA2 8146'
  },
  {
    'build server': 'macOS Builder',
    'unique id': 'Go Ethereum macOS Builder <geth-ci@ethereum.org>',
    'openpgp key': {
      label: '7B9E2481',
      url: 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x558915E17B9E2481'
    },
    fingerprint: '6D1D AF5D 0534 DEA6 1AA7  7AD5 5589 15E1 7B9E 2481'
  },
  {
    'build server': 'Windows Builder',
    'unique id': 'Go Ethereum Windows Builder <geth-ci@ethereum.org>',
    'openpgp key': {
      label: 'D2A67EAC',
      url: 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x9417309ED2A67EAC'
    },
    fingerprint: 'C4B3 2BB1 F603 4241 A9E6  50A1 9417 309E D2A6 7EAC'
  }
];
export const DOWNLOADS_DEVELOPERS_DATA = [
  {
    developer: 'Felix Lange',
    'unique id': 'Felix Lange <fjl@ethereum.org>',
    'openpgp key': {
      label: 'E058A81C',
      url: 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x337E68FCE058A81C'
    },
    fingerprint: '6047 0B71 5865 392D E43D 75A3 337E 68FC E058 A81C'
  },
  {
    developer: 'Martin Holst Swende',
    'unique id': 'Martin Holst Swende <martin.swende@ethereum.org>',
    'openpgp key': {
      label: '05A5DDF0',
      url: 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x683B438C05A5DDF0'
    },
    fingerprint: 'CA99 ABB5 B36E 24AD 5DA0 FD40 683B 438C 05A5 DDF0'
  },
  {
    developer: 'Péter Szilágyi',
    'unique id': 'Péter Szilágyi <peter@ethereum.org>',
    'openpgp key': {
      label: '1CCB7DD2',
      url: 'https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x119A76381CCB7DD2'
    },
    fingerprint: '4948 43FC E822 1C4C 86AB 5E2F 119A 7638 1CCB 7DD2'
  }
];

export const DOWNLOADS_OPENPGP_DEVELOPER_HEADERS = [
  'Developer',
  'Unique ID',
  'OpenPGP Key',
  'Fingerprint'
];

// Metadata
export const SITE_URL = 'https://geth.ethereum.org';
export const SITE_NAME = 'go-ethereum';
export const METADATA = {
  HOME_TITLE: 'Home',
  HOME_DESCRIPTION:
    'Go-ethereum website, home for the official Golang execution layer implementation of the Ethereum protocol',
  DOWNLOADS_TITLE: 'Downloads',
  DOWNLOADS_DESCRIPTION: 'All Geth releases and builds, available for download',
  PAGE_404_TITLE: '404 - Page not found',
  PAGE_404_DESCRIPTION: 'The page you are looking for does not exist'
};

// GitHub urls
export const LATEST_GETH_RELEASE_URL =
  'https://api.github.com/repos/ethereum/go-ethereum/releases/latest';
export const ALL_GETH_COMMITS_URL = 'https://api.github.com/repos/ethereum/go-ethereum/commits/';
export const RELEASE_COMMIT_BASE_URL = 'https://github.com/ethereum/go-ethereum/tree/';
export const LAST_COMMIT_BASE_URL = 'https://api.github.com/repos/ethereum/go-ethereum/commits';

// Binaries urls
export const BINARIES_BASE_URL = 'https://gethstore.blob.core.windows.net/builds/';
export const LINUX_BINARY_BASE_URL =
  'https://gethstore.blob.core.windows.net/builds/geth-linux-amd64-';
export const MACOS_BINARY_BASE_URL =
  'https://gethstore.blob.core.windows.net/builds/geth-darwin-amd64-';
export const WINDOWS_BINARY_BASE_URL =
  'https://gethstore.blob.core.windows.net/builds/geth-windows-amd64-';

// Blobs urls
// linux
export const ALL_LINUX_GETH_RELEASES_URL =
  'https://gethstore.blob.core.windows.net/builds?restype=container&comp=list&prefix=geth-linux';
export const ALL_LINUX_ALLTOOLS_GETH_RELEASES_URL =
  'https://gethstore.blob.core.windows.net/builds?restype=container&comp=list&prefix=geth-alltools-linux';

// macOS
export const ALL_MACOS_GETH_RELEASES_URL =
  'https://gethstore.blob.core.windows.net/builds?restype=container&comp=list&prefix=geth-darwin';
export const ALL_MACOS_ALLTOOLS_GETH_RELEASES_URL =
  'https://gethstore.blob.core.windows.net/builds?restype=container&comp=list&prefix=geth-alltools-darwin';

// windows
export const ALL_WINDOWS_GETH_RELEASES_URL =
  'https://gethstore.blob.core.windows.net/builds?restype=container&comp=list&prefix=geth-windows';
export const ALL_WINDOWS_ALLTOOLS_GETH_RELEASES_URL =
  'https://gethstore.blob.core.windows.net/builds?restype=container&comp=list&prefix=geth-alltools-windows';

// android
export const ALL_ANDROID_GETH_RELEASES_URL =
  'https://gethstore.blob.core.windows.net/builds?restype=container&comp=list&prefix=geth-android-all';

// iOS
export const ALL_IOS_GETH_RELEASES_URL =
  'https://gethstore.blob.core.windows.net/builds?restype=container&comp=list&prefix=geth-ios-all';

// Sources urls
export const LATEST_SOURCES_BASE_URL = 'https://github.com/ethereum/go-ethereum/archive/';

// Release notes urls
export const RELEASE_NOTES_BASE_URL = 'https://github.com/ethereum/go-ethereum/releases/tag/';

// Code snippet class constants
export const CLASSNAME_PREFIX = 'language-';
