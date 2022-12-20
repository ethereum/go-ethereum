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
