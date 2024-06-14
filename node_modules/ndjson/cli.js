#!/usr/bin/env node

const fs = require('fs')
const { pipeline } = require('readable-stream')
const minimist = require('minimist')
const ndjson = require('./index.js')

const args = minimist(process.argv.slice(2))
const first = args._[0]

if (!first) {
  console.error('Usage: ndjson [input] <options>')
  process.exit(1)
}

const inputStream = first === '-'
  ? process.stdin
  : fs.createReadStream(first)

pipeline(
  inputStream,
  ndjson.parse(args),
  ndjson.stringify(args),
  process.stdout,
  (err) => {
    err ? process.exit(1) : process.exit(0)
  }
)