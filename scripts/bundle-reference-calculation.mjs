// Bundles the pinned miao-plugin calculation modules for Honkai: Star Rail.
// Only numerical modules are transpiled; nothing here starts Yunzai, reads
// configuration, fetches resources or loads user code.
//
// Usage: node scripts/bundle-reference-calculation.mjs <参考项目/2026-09-15>
//
// Writes the miao runtime to internal/reference/miao/common.js and the calc
// directory internal/assets/calc: catalog.json, game.js (light cone and set
// effects plus the relic scoring tables), characters/<id>-<name>.js from
// calc.js and scores/<id>-<name>.js from artis.js. Characters with data but no
// calc.js enter the catalog without a damage script so they can still be
// scored. The relic set names are also written to internal/assets/catalog.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { calcScriptName, transpile, walk } from './calc-bundle.mjs'

const references = process.argv[2] && path.resolve(process.argv[2])
if (!references) throw new Error('Usage: bundle-reference-calculation.mjs <references>')
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const typescript = createRequire(path.join(root, 'ui/package.json'))('typescript')
const miao = path.join(references, 'miao-plugin')
const game = 'sr'

const allowedImports = ['lodash', '#miao', './DmgCalcMeta.js', './DmgMastery.js', './AttrItem.js', '../Base.js', '../index.js', '#miao.models', './AttrData.js', './common/Elem.js', './ArtisMark.js', './ArtisMarkCfg.js', '../../artifact/artis-mark.js']

// Two pinned formulas look up misnamed table rows; each patch fails loudly once
// upstream changes, see internal/reference/miao/PATCHES.md.
function patch(file, source) {
  if (file === 'resources/meta-sr/character/真珠/calc.js') {
    for (const label of ['百分比生命', '固定值']) {
      const before = '目标治疗·' + label
      if (source.split(before).length !== 3) throw new Error('Pearl compatibility patch no longer applies')
      source = source.replaceAll(before, '治疗·' + label)
    }
  }
  if (file === 'resources/meta-sr/character/刻律德菈/calc.js') {
    const before = "talent.t['攻击力提高上限']"
    if (source.split(before).length !== 2) throw new Error('Cerydra compatibility patch no longer applies')
    source = source.replace(before, "talent.t['攻击力']")
  }
  return source
}

async function module(file, provided) {
  const source = patch(file, provided ?? await fs.readFile(path.join(miao, file), 'utf8'))
  return transpile(typescript, file, source, allowedImports)
}
const relative = file => path.relative(miao, file).replaceAll('\\', '/')

let common = '// Fixed miao-plugin 7f6f1c84; MIT, see LICENSES. Generated numeric modules.\n'
common += `const Elem=${await module('components/common/Elem.js')}.default;\nFormat=${await module('components/Format.js')}.default;\n`
for (const [name, file] of [['AttrItem', 'dmg/AttrItem'], ['DmgMastery', 'dmg/DmgMastery'], ['DmgAttr', 'dmg/DmgAttr'], ['DmgCalc', 'dmg/DmgCalc'], ['AttrData', 'attr/AttrData'], ['Attr', 'attr/Attr']]) {
  if (name === 'DmgMastery') common += `const {erType,erTitle,eleBaseDmg,breakBaseDmg,cryBaseDmg,elationBaseDmg}=${await module('models/dmg/DmgCalcMeta.js')};\n`
  common += `const ${name}=${await module('models/' + file + '.js')}.default;\n`
}
for (const name of ['ArtisMarkCfg', 'ArtisMark']) common += `const ${name}=${await module('models/artis/' + name + '.js')}.default;\n`
await fs.writeFile(path.join(root, 'internal/reference/miao/common.js'), common)

const resources = path.join(miao, 'resources/meta-sr')
const calc = path.join(root, 'internal/assets/calc')
for (const directory of ['characters', 'scores']) {
  await fs.rm(path.join(calc, directory), { recursive: true, force: true })
  await fs.mkdir(path.join(calc, directory), { recursive: true })
}
const catalog = { version: 'miao-7f6f1c84-reference-v1', characters: [], weapons: [], coverage: [] }

// Showcase answers name a relic only by ID, and 面板换装 swaps a relic's set:
// the set and slot give the relic's name.
const sets = JSON.parse(await fs.readFile(path.join(resources, 'artifact/data.json'), 'utf8'))
const setNames = {}
const pieces = {}
for (const set of Object.values(sets)) {
  for (const [slot, part] of Object.entries(set.idxs || {})) {
    for (const id of Object.keys(part.ids || {})) setNames[id] = set.name
    if (part.name) {
      setNames[part.name] = set.name
      ;(pieces[set.name] ||= [])[slot - 1] = part.name
    }
  }
}
const pluginCatalog = path.join(root, 'internal/assets/catalog.json')
const local = JSON.parse(await fs.readFile(pluginCatalog, 'utf8'))
local.artifact_sets = setNames
local.artifact_pieces = pieces
await fs.writeFile(pluginCatalog, JSON.stringify(local, null, 2) + '\n')

let gameSource = `const artifactBuffs=${await module('resources/meta-sr/artifact/calc.js')}.default;\nconst weaponBuffs={};\n`
// The tables Meta.getMeta(game, 'arti') serves to the scoring modules.
const artifact = 'resources/meta-sr/artifact/'
gameSource += `const artiMeta=(()=>{const attrs=${await module(artifact + 'meta.js')};const alias=${await module(artifact + 'alias.js')};const mark=${await module(artifact + 'artis-mark.js')};return {mainAttr:attrs.mainAttr,subAttr:attrs.subAttr,attrMap:attrs.attrMap,usefulAttr:mark.usefulAttr,setAbbr:alias.artiSetAbbr}})();\nconst usefulAttr=artiMeta.usefulAttr;\n`
for (const file of (await walk(path.join(resources, 'weapon'))).filter(f => path.basename(f) === 'calc.js')) {
  gameSource += `Object.assign(weaponBuffs,loadWeaponDefinitions(${await module(relative(file))}.default));\n`
}
await fs.writeFile(path.join(calc, 'game.js'), gameSource)

// scoreScript bundles a character directory's artis.js, if it has one.
const scoreScript = async (directory, data) => {
  const file = path.join(directory, 'artis.js')
  try {
    await fs.access(file)
  } catch {
    return undefined
  }
  const script = calcScriptName(game + '_' + data.id, data.name).replace(/^characters\//, 'scores/')
  await fs.writeFile(path.join(calc, script), `const scoreRule=${await module(relative(file))};\n`)
  return script
}
// Upstream names characters through its index, which a per-character
// data.json can disagree with (星·欢愉/data.json says 星·同谐).
const index = JSON.parse(await fs.readFile(path.join(resources, 'character/data.json'), 'utf8'))
const addCharacter = async (data, source, code, authored, score) => {
  const key = game + '_' + data.id
  if (catalog.characters.some(c => c.key === key)) throw new Error('Duplicate reference key ' + key)
  const name = index[data.id]?.name || data.name
  let script = ''
  if (code) {
    script = calcScriptName(key, name)
    await fs.writeFile(path.join(calc, script), `const characterRule=${code};\n`)
    catalog.coverage.push(authored ? { key, source, authored: true } : { key, source })
  }
  catalog.characters.push({ key, game, id: String(data.id), name, element: data.elem, weapon_type: data.weapon, script, score_script: score, data })
}
const readData = async directory => JSON.parse(await fs.readFile(path.join(directory, 'data.json'), 'utf8'))

const calculated = new Set()
for (const file of (await walk(path.join(resources, 'character'))).filter(f => path.basename(f) === 'calc.js')) {
  const directory = path.dirname(file)
  const data = await readData(directory)
  if (!data?.id) throw new Error('Missing character metadata: ' + file)
  await addCharacter(data, relative(file), await module(relative(file)), false, await scoreScript(directory, data))
  calculated.add(directory)
}
for (const file of (await walk(path.join(resources, 'weapon'))).filter(f => path.basename(f) === 'data.json')) {
  const data = JSON.parse(await fs.readFile(file, 'utf8'))
  if (!data.id || !data.name || !data.attr) continue
  const type = path.relative(path.join(resources, 'weapon'), file).split(path.sep)[0]
  catalog.weapons.push({ game, id: String(data.id), name: data.name, type, data })
}
// Enhanced Sparkle and Black Swan have upstream data but no calc.js; the
// authored rules live in scripts/calc-supplements.
for (const directory of ['花火Pro', '黑天鹅Pro']) {
  const full = path.join(resources, 'character', directory)
  const data = await readData(full)
  if (catalog.characters.some(c => c.key === game + '_' + data.id)) throw new Error('Supplement now has an upstream rule; review before bundling')
  const source = 'scripts/calc-supplements/' + calcScriptName(game + '_' + data.id, data.name).slice('characters/'.length)
  const code = await fs.readFile(path.join(root, source), 'utf8')
  await addCharacter(data, source, await module(source, code), true, await scoreScript(full, data))
  calculated.add(full)
}
// Characters upstream can score but not calculate yet.
for (const file of (await walk(path.join(resources, 'character'))).filter(f => path.basename(f) === 'data.json')) {
  const directory = path.dirname(file)
  if (calculated.has(directory) || path.dirname(directory) !== path.join(resources, 'character')) continue
  const data = JSON.parse(await fs.readFile(file, 'utf8'))
  if (!data.id || !data.name) continue
  await addCharacter(data, '', '', false, await scoreScript(directory, data))
}
await fs.writeFile(path.join(calc, 'catalog.json'), JSON.stringify(catalog) + '\n')
console.log(JSON.stringify({ characters: catalog.characters.length, weapons: catalog.weapons.length, version: catalog.version }))
