// Converts the StarRail-plugin 模拟抽卡 item pools. Neither the odds nor the
// banners are advertised as the current official rules.
//
// Usage: node scripts/import-simulation-data.mjs <参考项目/2026-09-15>
// Reads internal/assets/data/resources.json for the banners and writes
// internal/assets/data/simulation.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const refs = path.resolve(process.argv[2])
const dataDir = path.join(root, 'internal/assets/data')

// utils/gachasimulation.js is a data-only module.
const source = await fs.readFile(path.join(refs, 'StarRail-plugin/utils/gachasimulation.js'), 'utf8')
const gacha = JSON.parse(new vm.Script(source.replace('export default', 'const data =') + '\nJSON.stringify(data)')
  .runInNewContext(Object.create(null), { timeout: 1000 })).gachaPool

// Only event banners with every rarity filled can be drawn, newest first.
const resources = JSON.parse(await fs.readFile(path.join(dataDir, 'resources.json'), 'utf8'))
const banners = resources.pools
  .filter(p => p.kind === 'event' && p.characters5.length && p.weapons5.length && p.characters4.length && p.weapons4.length)
  .map(p => ({ ...p, id: [p.version, p.half, p.from].join('|') }))
  .reverse()

const deck = {
  version: 'entertainment-reference-20260915-v1',
  five_characters: gacha.normal.char,
  five_weapons: gacha.normal.weapon,
  four_characters: gacha['four-char'],
  four_weapons: gacha['four-weapon'],
  three_weapons: gacha['three-weapon'],
  banners,
}
await fs.writeFile(path.join(dataDir, 'simulation.json'), JSON.stringify(deck) + '\n')
await fs.copyFile(path.join(refs, 'StarRail-plugin/LICENSE'), path.join(root, 'LICENSES/StarRail-plugin-Apache-2.0.txt'))
const standard = deck.five_characters.length + deck.five_weapons.length + deck.four_characters.length + deck.four_weapons.length + deck.three_weapons.length
console.log(JSON.stringify({ banners: banners.length, standard_items: standard }))
