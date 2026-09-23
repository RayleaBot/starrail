// Copies what StarRail-plugin's own panel service runs on from its pinned
// snapshot into internal/panelapi. Nothing here runs upstream code.
//
// sign.b64 is resources/fuck/fuck, the base64 signing script utils/auth.js
// evaluates, byte for byte. rank-skill-up.json is utils/damage/json/
// avatarRankSkillUp.json, the skills each eidolon raises, which apps/panel.js
// reads. damage.js bundles utils/damage (main.js and effect/*.js with the
// tables its data/data.js loads) into one expression whose value is
// damage(charinfo); the character table keeps only the first level's
// properties of each trace, the part main.js reads.
//
// Usage: node scripts/bundle-panel-api.mjs <参考项目/2026-09-15>
import fs from 'node:fs/promises'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { transpile } from './calc-bundle.mjs'

const references = process.argv[2] && path.resolve(process.argv[2])
if (!references) throw new Error('Usage: bundle-panel-api.mjs <references>')
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const typescript = createRequire(path.join(root, 'ui/package.json'))('typescript')
const upstream = path.join(references, 'StarRail-plugin')
const target = path.join(root, 'internal/panelapi')
const pinned = JSON.parse(await fs.readFile(path.join(references, 'StarRail-plugin.source.json'), 'utf8'))

await fs.copyFile(path.join(upstream, 'resources/fuck/fuck'), path.join(target, 'sign.b64'))
await fs.copyFile(path.join(upstream, 'utils/damage/json/avatarRankSkillUp.json'), path.join(target, 'rank-skill-up.json'))

const damage = path.join(upstream, 'utils/damage')
const table = async name => JSON.parse(await fs.readFile(path.join(damage, 'json', name), 'utf8'))
const characters = await table('character.json')
const traces = {}
for (const [avatar, nodes] of Object.entries(characters)) {
  traces[avatar] = {}
  for (const [id, node] of Object.entries(nodes)) {
    traces[avatar][id] = { levels: node.levels.slice(0, 1).map(level => ({ properties: level.properties.slice(0, 1) })) }
  }
}
// The names data/data.js exports that the modules import.
const tables = {
  skilldict: await table('SkillData.json'),
  weapon_effect: await table('weapon_effect.json'),
  char_dict: traces,
  equipment_dict: await table('Equipment.json'),
  relicsetskill: await table('RelicSetSkill.json'),
}
const module = async file => transpile(typescript, file, await fs.readFile(path.join(damage, file), 'utf8'), ['../data/data.js', './data/data.js', './effect/weapon.js', './effect/avatar.js', './effect/relice.js'])

let source = `// StarRail-plugin ${pinned.commit.slice(0, 7)} utils/damage (Apache-2.0, see LICENSES), bundled by\n// scripts/bundle-panel-api.mjs. Its logger.debug calls log nothing here.\n(() => {\nconst logger = { debug () {} };\n`
for (const [name, value] of Object.entries(tables)) source += `const ${name} = ${JSON.stringify(value)};\n`
for (const [name, file] of [['weapon_ability', 'effect/weapon.js'], ['avatar_ability', 'effect/avatar.js'], ['relice_ability', 'effect/relice.js']]) {
  source += `const { ${name} } = ${await module(file)};\n`
}
source += `return ${await module('main.js')}.damage;\n})()\n`
await fs.writeFile(path.join(target, 'damage.js'), source)
console.log(`internal/panelapi: StarRail-plugin ${pinned.commit.slice(0, 7)}, damage.js ${(source.length / 1024).toFixed(0)} KiB`)
