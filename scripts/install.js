#!/usr/bin/env node
'use strict'

const https = require('https')
const fs = require('fs')
const path = require('path')

const version = require('../package.json').version

const PLATFORM_MAP = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows',
}

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64',
}

const platform = PLATFORM_MAP[process.platform]
const arch = ARCH_MAP[process.arch]

if (!platform || !arch) {
  console.error(`beakon: unsupported platform ${process.platform}/${process.arch}`)
  process.exit(1)
}

const ext = process.platform === 'win32' ? '.exe' : ''
const filename = `beakon_${platform}_${arch}${ext}`
const url = `https://github.com/codesharpdev/beakon/releases/download/v${version}/${filename}`
const dest = path.join(__dirname, '..', 'bin', `beakon${ext}`)

console.log(`beakon: downloading ${filename} from GitHub Releases...`)

function download(url, dest, cb) {
  const file = fs.createWriteStream(dest)
  function get(url) {
    https.get(url, (res) => {
      if (res.statusCode === 301 || res.statusCode === 302) {
        return get(res.headers.location)
      }
      if (res.statusCode !== 200) {
        fs.unlink(dest, () => {})
        cb(new Error(`HTTP ${res.statusCode} downloading ${url}`))
        return
      }
      res.pipe(file)
      file.on('finish', () => file.close(cb))
    }).on('error', (err) => {
      fs.unlink(dest, () => {})
      cb(err)
    })
  }
  get(url)
}

download(url, dest, (err) => {
  if (err) {
    console.error(`beakon: download failed: ${err.message}`)
    console.error(`  You can build from source: go build -o beakon ./cmd/beakon`)
    process.exit(1)
  }
  if (process.platform !== 'win32') {
    fs.chmodSync(dest, 0o755)
  }
  console.log(`beakon: installed to ${dest}`)
})
