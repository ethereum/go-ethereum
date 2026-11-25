import { XMLParser } from 'fast-xml-parser';

import {
  ALL_ANDROID_GETH_RELEASES_URL,
  ALL_IOS_GETH_RELEASES_URL,
  ALL_LINUX_ALLTOOLS_GETH_RELEASES_URL,
  ALL_LINUX_ARM64_GETH_RELEASES_URL,
  ALL_LINUX_GETH_RELEASES_URL,
  ALL_WINDOWS_ALLTOOLS_GETH_RELEASES_URL,
  ALL_WINDOWS_GETH_RELEASES_URL
} from '../constants';

const fetchXMLPages = async (url: string): Promise<string[]> => {
  const parser = new XMLParser();
  const pages: string[] = [];
  let marker = '';

  do {
    const pageURL = new URL(url);
    if (marker) pageURL.searchParams.set('marker', marker);

    const response = await fetch(pageURL);
    if (!response.ok) {
      throw new Error(`Failed to fetch releases from ${pageURL}: ${response.status}`);
    }
    const xml = await response.text();
    const nextMarker = parser.parse(xml)?.EnumerationResults?.NextMarker;

    pages.push(xml);
    marker = typeof nextMarker === 'string' ? nextMarker : '';
  } while (marker);

  return pages;
};

export const fetchXMLData = () => {
  const urls = [
    ALL_LINUX_GETH_RELEASES_URL,
    ALL_LINUX_ARM64_GETH_RELEASES_URL,
    ALL_LINUX_ALLTOOLS_GETH_RELEASES_URL,
    ALL_WINDOWS_GETH_RELEASES_URL,
    ALL_WINDOWS_ALLTOOLS_GETH_RELEASES_URL,
    ALL_ANDROID_GETH_RELEASES_URL,
    ALL_IOS_GETH_RELEASES_URL
  ];

  return Promise.all(urls.map(fetchXMLPages));
};
