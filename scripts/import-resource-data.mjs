// Converts miao-plugin's material and banner data. Only the data-only
// modules are evaluated; no plugin code is loaded.
//
// Usage: node scripts/import-resource-data.mjs <参考项目/2026-09-15>
// Writes internal/assets/data/resources.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const meta = path.join(path.resolve(process.argv[2]), 'miao-plugin/resources/meta-sr')

// dataModule evaluates a module without imports and returns expression.
async function dataModule(file, expression) {
  let source = await fs.readFile(file, 'utf8')
  if (/^\s*import\s/m.test(source)) {
    throw Error('Data-only module required')
  }
  source = source.replace(/^export\s+(?=const\s)/gm, '').replace(/^export default/m, 'const defaultData =')
  const context = vm.createContext(Object.create(null), { codeGeneration: { strings: false, wasm: false } })
  return JSON.parse(new vm.Script(`${source}\nJSON.stringify(${expression})`).runInContext(context, { timeout: 1000 }))
}

// Materials are listed by kind; Star Rail has no weekday rotation.
const byKind = JSON.parse(await fs.readFile(path.join(meta, 'material/data.json'), 'utf8'))
const materials = new Map()
for (const [kind, list] of Object.entries(byKind)) {
  for (const item of Object.values(list)) {
    materials.set(item.name, { id: String(item.id), name: item.name, kind, rarity: item.star, family: item.name, days: [], city: '', sources: Array.isArray(item.source) ? item.source : [], description: item.desc ?? '' })
  }
}

const pools = await dataModule(path.join(meta, 'info/index.js'), 'poolDetailSr.map(p=>({...p,kind:"event"}))')

// Star Rail's character data gives no birthdays.
const out = {
  version: 'miao-7f6f1c84c891',
  materials: [...materials.values()].sort((a, b) => a.name.localeCompare(b.name, 'zh')),
  pools: pools.map(p => ({ version: p.version, half: p.half, from: p.from, to: p.to, kind: p.kind, characters5: p.char5 ?? [], characters4: p.char4 ?? [], weapons5: p.weapon5 ?? [], weapons4: p.weapon4 ?? [] })),
  birthdays: [],
}
await fs.writeFile(path.join(root, 'internal/assets/data/resources.json'), JSON.stringify(out) + '\n')
console.log(JSON.stringify({ materials: materials.size, pools: pools.length }))
