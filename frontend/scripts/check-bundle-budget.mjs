import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { gzipSync } from 'node:zlib'

const distDir = resolve(process.cwd(), 'dist')
const html = readFileSync(resolve(distDir, 'index.html'), 'utf8')
const assetPaths = new Set()

for (const match of html.matchAll(/<(?:script|link)\b[^>]+(?:src|href)="([^"]+\.js)"[^>]*>/g)) {
  assetPaths.add(match[1].replace(/^\//, ''))
}

if (assetPaths.size === 0) throw new Error('No initial JavaScript assets found in dist/index.html')

const files = [...assetPaths].map((assetPath) => ({
  assetPath,
  bytes: gzipSync(readFileSync(resolve(distDir, assetPath)), { level: 9 }).byteLength,
}))
const total = files.reduce((sum, file) => sum + file.bytes, 0)
const limit = 120 * 1024

for (const file of files) process.stdout.write(`${file.assetPath}: ${(file.bytes / 1024).toFixed(2)} KiB gzip\n`)
process.stdout.write(`Initial JavaScript: ${(total / 1024).toFixed(2)} KiB / ${(limit / 1024).toFixed(0)} KiB gzip\n`)

if (total > limit) {
  process.stderr.write(`Initial JavaScript exceeds budget by ${((total - limit) / 1024).toFixed(2)} KiB\n`)
  process.exit(1)
}
