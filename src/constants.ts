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
export const GO_URL = 'https://go.dev/';

// Downloads
export const DEFAULT_BUILD_AMOUNT_TO_SHOW = 10;
export const DOWNLOAD_HEADER_BUTTONS: {[index: string]: {name: string; image: string; imageAlt: string; buildURL: string;}} = {
  linuxBuild: {
    name: 'Linux',
    image: '/images/pages/linux-penguin.svg',
    imageAlt: 'Linux logo',
    buildURL: ''
  },
  macOSBuild: {
    name: 'macOS',
    image: '/images/pages/macos-logo.svg',
    imageAlt: 'macOS logo',
    buildURL: ''
  },
  windowsBuild: {
    name: 'Windows',
    image: '/images/pages/windows-logo.svg',
    imageAlt: 'Windows logo',
    buildURL: ''
  },
  sourceCode: {
    name: 'Sources',
    image: '/images/pages/source-branch.svg',
    imageAlt: 'Source branch logo',
    buildURL: ''
  }
}
export const DOWNLOAD_TABS = [
  'Linux',
  'macOS',
  'Windows',
  'iOS',
  'Android'
]
export const DOWNLOAD_TAB_COLUMN_HEADERS = [
  'Release',
  'Commit',
  'Kind',
  'Arch',
  'Size',
  'Published',
  'Signature',
  'Checksum (MD5)'
]
export const DOWNLOAD_OPENPGP_BUILD_HEADERS = [
  'Build Server',
  'Unique ID',
  'OpenPGP Key',
  'Fingerprint'
]
export const DOWNLOAD_OPENPGP_DEVELOPER_HEADERS = [
  'Developer',
  'Unique ID',
  'OpenPGP Key',
  'Fingerprint'
]