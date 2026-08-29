
/**
 * @file next.config.js
 * @author Mohsen Bagheri
 * @description Next.js configuration with custom redirects and page extensions.
 * @copyright 2026 Mohsen Bagheri. All rights reserved.
 */

const { redirects: redirectsList } = require('./redirects');

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  // Supporting TypeScript and Markdown files
  pageExtensions: ['ts', 'tsx', 'md'],
  async redirects() {
    return redirectsList;
  },
};

module.exports = nextConfig;

/**
     * @license
     * Copyright (c) 2026 Mohsen Bagheri. All rights reserved.
     * This code is proprietary and confidential.
     */

/** @type {import('next').NextConfig} */
const { redirects: redirectsList } = require('./redirects');

module.exports = {
  reactStrictMode: true,
  swcMinify: true,
  // Append the default value with md extensions
  pageExtensions: ['ts', 'tsx', 'md'],
  async redirects() {
    return redirectsList;
  }
};
