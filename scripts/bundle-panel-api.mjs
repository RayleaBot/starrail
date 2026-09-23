// Copies what StarRail-plugin's own panel service runs on from its pinned
// snapshot into internal/panelapi: sign.b64 is resources/fuck/fuck, the
// base64 signing script utils/auth.js evaluates, byte for byte. Nothing here
// runs upstream code.
//
// Usage: node scripts/bundle-panel-api.mjs <参考项目/2026-09-15>
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const references = process.argv[2] && path.resolve(process.argv[2])
if (!references) throw new Error('Usage: bundle-panel-api.mjs <references>')
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const upstream = path.join(references, 'StarRail-plugin')
const target = path.join(root, 'internal/panelapi')

await fs.copyFile(path.join(upstream, 'resources/fuck/fuck'), path.join(target, 'sign.b64'))
const pinned = JSON.parse(await fs.readFile(path.join(references, 'StarRail-plugin.source.json'), 'utf8'))
console.log(`internal/panelapi: StarRail-plugin ${pinned.commit.slice(0, 7)}`)
