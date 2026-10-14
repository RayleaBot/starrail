// Converts miao-plugin's material and banner data. Only the data-only
// modules are evaluated; no plugin code is loaded.
//
// Usage: node scripts/import-resource-data.mjs <参考项目/2026-10-13>
// Writes internal/assets/data/resources.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const refs = path.resolve(process.argv[2])
const meta = path.join(refs, 'miao-plugin/resources/meta-sr')
const { commit } = JSON.parse(await fs.readFile(path.join(refs, 'miao-plugin.source.json'), 'utf8'))

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

// Family members in items run from lowest to highest rarity.
const materials = Object.values(JSON.parse(await fs.readFile(path.join(meta, 'material/data.json'), 'utf8'))).map(item => {
  if (!item?.name || !item.type) throw Error('Unexpected material data')
  return { id: String(item.id), name: item.name, kind: item.type, rarity: item.star, family: Object.keys(item.items ?? {})[0] ?? item.name, sources: Array.isArray(item.source) ? item.source : [], description: item.desc ?? '' }
})

const { pools, names } = await dataModule(path.join(meta, 'info/index.js'), '({pools: poolDetailSr, names: poolNameSr})')
const { pools: collabs } = JSON.parse(await fs.readFile(path.join(refs, 'StarRail-plugin/tools/collab-pools.json'), 'utf8'))
const poolData = p => ({ version: p.version, half: p.half, from: p.from, to: p.to, characters5: p.char5 ?? [], characters4: p.char4 ?? [], weapons5: p.weapon5 ?? [], weapons4: p.weapon4 ?? [] })

const out = {
  version: 'miao-' + commit.slice(0, 12),
  materials: materials.sort((a, b) => a.name.localeCompare(b.name, 'zh')),
  pools: pools.map(p => {
    const name = names[p.char5[0]]
    if (!name) throw Error('Missing pool name: ' + p.char5[0])
    return { ...poolData(p), kind: 'event', name }
  }),
  collab_pools: collabs.map(p => ({ ...poolData(p), kind: 'collab', gacha_type: String(p.gachaType), name: p.poolName })),
}
await fs.writeFile(path.join(root, 'internal/assets/data/resources.json'), JSON.stringify(out) + '\n')
console.log(JSON.stringify({ materials: materials.length, pools: pools.length, collab_pools: collabs.length }))
