#!/usr/bin/env node
'use strict'

const { spawnSync } = require('child_process')
const path = require('path')

const ext = process.platform === 'win32' ? '.exe' : ''
const bin = path.join(__dirname, '..', 'bin', `beakon${ext}`)

const result = spawnSync(bin, process.argv.slice(2), { stdio: 'inherit' })

if (result.error) {
  if (result.error.code === 'ENOENT') {
    console.error('beakon: binary not found. Run: npm install -g beakon')
  } else {
    console.error(`beakon: ${result.error.message}`)
  }
  process.exit(1)
}

process.exit(result.status ?? 0)
