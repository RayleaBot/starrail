<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { alignBuildRows, damageGain, weaponPromotions, type BuildPrepared, type BuildSkill, type CharacterBuild, type BuildConditions } from './build'
import BuildConditionsEditor from './BuildConditionsEditor.vue'
import CharacterSelect from './shell/CharacterSelect.vue'
import StatusLine from './shell/StatusLine.vue'
import type { BuildInvoke } from './build'

// 配装计算: the character's reference damage, read and computed as soon as
// the character is shown, and what a weapon or another character's whole
// equipment would change.
const props = defineProps<{ characterId: string; characterName: string; accountRef: string; roleRef: string; invoke: BuildInvoke }>()
const prepared = ref<BuildPrepared>(), result = ref<CharacterBuild>(), busy = ref(false), error = ref('')
const enemyLevel = ref(103), changeWeapon = ref(false), weaponID = ref(''), weaponLevel = ref(90), promote = ref(6), refinement = ref(1), critical = ref(false)
const equipmentSource = ref(''), changeEquipment = ref(false)
const conditions = ref<BuildConditions | null>(null)
let disposed = false, revision = 0
onUnmounted(() => { disposed = true })
const choice = () => ({ account_ref: props.accountRef, role_ref: props.roleRef, character_id: props.characterId })
const promotions = computed(() => weaponPromotions(weaponLevel.value))
watch(promotions, values => { if (!values.includes(promote.value)) promote.value = values.at(-1) ?? 0 }, { flush: 'sync' })
watch([enemyLevel, changeWeapon, weaponID, weaponLevel, promote, refinement, changeEquipment, equipmentSource], () => { result.value = undefined; revision++ }, { flush: 'sync' })
watch(conditions, () => { result.value = undefined; revision++ }, { deep: true, flush: 'sync' })
async function load() {
  if (busy.value) return
  busy.value = true; error.value = ''; result.value = undefined
  try {
    const value = await props.invoke<BuildPrepared>('build.prepare', choice())
    if (disposed) return
    conditions.value = null; prepared.value = value; enemyLevel.value = value.enemy_level; weaponID.value = value.weapon.id
    weaponLevel.value = value.weapon.level || 80
    promote.value = value.weapon.id ? value.weapon.promote ?? 6 : 6; refinement.value = value.weapon.refinement; changeWeapon.value = false; changeEquipment.value = false; equipmentSource.value = ''
    result.value = value.build
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '自动计算未完成，请刷新面板后重试。' }
  finally { if (!disposed) busy.value = false }
}
async function calculate() {
  if (busy.value) return
  busy.value = true; error.value = ''; result.value = undefined
  const started = revision
  try {
    if (!Number.isInteger(enemyLevel.value) || enemyLevel.value < 1 || enemyLevel.value > 200) throw new Error('敌人等级应为 1–200 的整数。')
    const candidate = weaponID.value ? { id: weaponID.value, level: weaponLevel.value, promote: promote.value, refinement: refinement.value } : { id: '', level: 0, promote: 0, refinement: 1 }
    if (changeWeapon.value && weaponID.value && (!Number.isInteger(weaponLevel.value) || !promotions.value.includes(promote.value) || !Number.isInteger(refinement.value) || refinement.value < 1 || refinement.value > 5)) throw new Error('请检查武器等级、突破阶段和精炼。')
    const sourceID = equipmentSource.value
    if (changeEquipment.value && !sourceID) throw new Error('请选择装备来源角色。')
    const output = await props.invoke<{ build: CharacterBuild }>('build.compare', { ...choice(), enemy_level: enemyLevel.value, ...(changeWeapon.value ? { candidate_weapon: candidate } : {}), ...(changeEquipment.value ? { equipment_from_character_id: sourceID } : {}), ...(conditions.value ? { conditions: conditions.value } : {}) })
    if (!disposed && revision === started) result.value = output.build
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '换装计算未完成，请稍后重试。' }
  finally { if (!disposed) busy.value = false }
}
void load()
const rows = computed(() => result.value ? alignBuildRows(result.value) : [])
const format = (v: number, digits = 1) => new Intl.NumberFormat('zh-CN', { maximumFractionDigits: digits }).format(v)
function display(row?: BuildSkill) { if (!row) return '—'; if (row.kind === 'text') return row.text || '—'; const value = critical.value ? row.critical : row.expected; return value == null ? '—' : format(value) }
function gain(row: ReturnType<typeof alignBuildRows>[number]) { const key = critical.value ? 'critical' : 'expected'; const value = damageGain(row.before?.[key], row.after?.[key]); return value == null ? '—' : `${value >= 0 ? '+' : ''}${format(value, 2)}%` }
</script>

<template>
  <section class="subsection" aria-labelledby="build-heading">
    <div class="section-heading">
      <div class="titles"><h3 id="build-heading">{{ characterName }} · 配装计算</h3><p>按参考情境计算技能伤害，可试换光锥或另一角色的整套装备；试算不改变游戏装备。</p></div>
      <button type="button" :disabled="busy" @click="load">{{ busy ? '正在读取并计算…' : '重新读取并计算' }}</button>
    </div>
    <StatusLine v-if="busy && !prepared" kind="info" text="正在读取面板并计算参考伤害…" />
    <StatusLine kind="error" :text="error"><template v-if="!prepared" #actions><button class="compact" type="button" :disabled="busy" @click="load">重试</button></template></StatusLine>
    <form v-if="prepared" class="section" @submit.prevent="calculate">
      <fieldset :disabled="busy">
        <div class="field-grid">
          <label class="field"><span>敌人等级</span><input v-model.number="enemyLevel" type="number" min="1" max="200" step="1" required></label>
          <div class="checks"><label class="check"><input v-model="changeWeapon" type="checkbox">试换光锥</label><label class="check"><input v-model="changeEquipment" type="checkbox">调入整套装备</label></div>
        </div>
        <div v-if="changeWeapon" class="field-grid">
          <label class="field wide"><span>候选光锥</span><select v-model="weaponID"><option value="">未装备</option><option v-for="weapon in prepared.weapons" :key="weapon.id" :value="weapon.id">{{ weapon.name }}</option></select></label>
          <template v-if="weaponID">
            <label class="field"><span>光锥等级</span><input v-model.number="weaponLevel" type="number" min="1" max="80" step="1" required></label>
            <label class="field"><span>突破阶段</span><select v-model.number="promote"><option v-for="value in promotions" :key="value" :value="value">阶段 {{ value }}</option></select></label>
            <label class="field"><span>叠影</span><select v-model.number="refinement"><option v-for="value in 5" :key="value" :value="value">{{ value }}</option></select></label>
          </template>
        </div>
        <div v-if="changeEquipment" class="field-grid">
          <CharacterSelect v-model="equipmentSource" class="wide" source="role" label="装备来源角色" :exclude="[characterId]" />
          <p class="hint wide">只调入同一游戏账号下该角色已查询到的整套装备用于模拟；角色、技能和光锥仍按当前选择计算。</p>
        </div>
        <BuildConditionsEditor v-model="conditions" :character-id="characterId" :disabled="busy" :invoke="invoke" />
        <div class="actions"><button class="primary" type="submit">{{ conditions ? '计算自定义候选' : changeWeapon || changeEquipment ? '计算换装收益' : '按当前条件重算' }}</button></div>
      </fieldset>
    </form>
    <div v-if="result" class="section" aria-live="polite">
      <div class="section-heading">
        <div class="titles"><h3>参考情境与计算结果</h3><p>当前：{{ result.baseline.weapon.name || '未装备' }}<template v-if="result.candidate"> → 候选：{{ result.candidate.weapon.name || '未装备' }}</template><template v-if="result.equipment_from">；候选整套装备来自 {{ result.equipment_from.name }}</template></p></div>
        <label class="check"><input v-model="critical" type="checkbox">查看暴击值</label>
      </div>
      <p class="hint">半血、满层、击杀、队伍等条件沿用各条参考规则。展开情境查看增益及适用技能；它们不代表角色实际处于该状态。按弱点敌人、韧性减伤系数 0.9 计算。</p>
      <div class="table-wrap">
        <table>
          <thead><tr><th scope="col">参考情境</th><th scope="col" class="num">{{ critical ? '当前暴击值' : '当前期望 / 效果' }}</th><template v-if="result.candidate"><th scope="col" class="num">换装后</th><th scope="col" class="num">变化</th></template></tr></thead>
          <tbody>
            <tr v-for="row in rows" :key="row.id">
              <th scope="row" class="scenario"><details><summary>{{ row.title }}</summary><div class="build-buffs"><p v-if="row.before">当前参考增益</p><ul v-if="row.before"><li v-for="(buff, index) in row.before.buffs" :key="index">{{ buff }}</li></ul><p v-if="row.after">候选参考增益</p><ul v-if="row.after"><li v-for="(buff, index) in row.after.buffs" :key="index">{{ buff }}</li></ul></div></details></th>
              <td class="num">{{ display(row.before) }}</td>
              <template v-if="result.candidate"><td class="num">{{ display(row.after) }}</td><td class="num">{{ gain(row) }}</td></template>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="hint">“—”表示该情境未适用、不支持暴击或无法计算变化比例。公式版本：{{ result.version }}。</p>
    </div>
  </section>
</template>

<style scoped>
.scenario { position: static; background: transparent; color: var(--ivory); font-weight: 500; min-width: 160px; white-space: normal; }
.scenario summary { font-weight: 500; min-height: 0; }
.build-buffs { font-size: 12px; font-weight: 400; color: var(--ivory-muted); max-width: 56ch; display: grid; gap: 6px; padding-top: 6px; }
.build-buffs ul { padding-left: 18px; display: grid; gap: 4px; }
</style>
