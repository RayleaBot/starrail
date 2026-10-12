<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import type { BuildConditions } from './build'
import CharacterSelect from './shell/CharacterSelect.vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'

// BuildConditionsEditor builds the custom candidate: teammates, bonuses and
// the reference buffs to switch off, saved as named presets per character.
interface Field { key: string; label: string; unit: string; max: number }
interface Preset { ref: string; revision: number; name: string; character_id: string; conditions: BuildConditions }
const props = defineProps<{ characterId: string; modelValue: BuildConditions | null; disabled: boolean; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const emit = defineEmits<{ 'update:modelValue': [value: BuildConditions | null] }>()
const shell = useShell()
const fields = ref<Field[]>([]), presets = ref<Preset[]>([]), preset = ref(''), name = ref(''), teammate = ref(''), source = ref(-1), field = ref('atkPct'), amount = ref(0), error = ref(''), busy = ref(false), notice = ref(''), confirm = ref<HTMLDialogElement | null>(null)
let disposed = false
const available = computed(() => presets.value.filter(p => p.character_id === props.characterId))
const selected = computed(() => presets.value.find(p => p.ref === preset.value))
const rows = computed(() => props.modelValue ? [{ name: '自身额外增益', bonuses: props.modelValue.bonuses, index: -1 }, ...props.modelValue.team.map((t, index) => ({ ...t, index }))] : [])
function update(fn: (c: BuildConditions) => void) { const c = JSON.parse(JSON.stringify(props.modelValue ?? { disable_character: false, disable_weapon: false, disable_equipment: false, bonuses: {}, team: [] })) as BuildConditions; fn(c); emit('update:modelValue', c); notice.value = '' }
function enabled(event: Event) { if ((event.target as HTMLInputElement).checked) update(() => {}); else emit('update:modelValue', null) }
async function load() { try { const [schema, list] = await Promise.all([props.invoke<{ fields: Field[] }>('build.conditions'), props.invoke<{ items: Preset[] }>('build.presets.list')]); if (!disposed) { fields.value = schema.fields; presets.value = list.items } } catch (e) { if (!disposed) error.value = e instanceof Error ? e.message : '方案资料读取失败。' } }
void load(); onUnmounted(() => { disposed = true })
watch(() => props.characterId, () => { preset.value = ''; name.value = ''; teammate.value = ''; source.value = -1; error.value = ''; emit('update:modelValue', null) })
function loadPreset() { if (!selected.value) return; name.value = selected.value.name; emit('update:modelValue', JSON.parse(JSON.stringify(selected.value.conditions))); source.value = -1; notice.value = '方案已载入，请重新计算。' }
// The teammate is picked from the catalog the shell holds, so its name comes
// from there; the picker already leaves out this character and the team.
function addTeammate() {
  const found = shell.catalog.value.find(entry => entry.id === teammate.value)
  if (!props.modelValue || !found || found.id === props.characterId || props.modelValue.team.some(t => t.character_id === found.id)) return
  update(c => c.team.push({ character_id: found.id, name: found.name, bonuses: {} })); teammate.value = ''; error.value = ''
}
function addBonus() { const schema = fields.value.find(f => f.key === field.value); if (!schema || !Number.isFinite(amount.value) || amount.value < 0 || amount.value > schema.max) { error.value = '请在所列范围内填写增益。'; return } update(c => { const target = source.value < 0 ? c.bonuses : c.team[source.value]?.bonuses; if (target) target[field.value] = Number(amount.value) }); error.value = '' }
async function save(asNew: boolean) { if (!props.modelValue || busy.value) return; busy.value = true; error.value = ''; try { const existing = asNew ? undefined : selected.value; const result = await props.invoke<{ items: Preset[]; preset: Preset }>('build.presets.save', { name: name.value, character_id: props.characterId, conditions: props.modelValue, ...(existing ? { ref: existing.ref, revision: existing.revision } : {}) }); if (!disposed) { presets.value = result.items; preset.value = result.preset.ref; notice.value = '方案已保存。' } } catch (e) { if (!disposed) error.value = e instanceof Error ? e.message : '方案保存失败。' } finally { busy.value = false } }
async function remove() { const p = selected.value; if (!p) return; busy.value = true; error.value = ''; try { const result = await props.invoke<{ items: Preset[] }>('build.presets.remove', { ref: p.ref, revision: p.revision }); if (!disposed) { presets.value = result.items; preset.value = ''; notice.value = '方案已删除。'; confirm.value?.close() } } catch (e) { error.value = e instanceof Error ? e.message : '方案删除失败。' } finally { busy.value = false } }
</script>

<template>
  <div class="subsection">
    <h3>自定义队伍与条件</h3>
    <p class="hint">全局方案：所有机器人共用，不保存账号或面板。</p>
    <label class="check"><input type="checkbox" :checked="modelValue !== null" :disabled="disabled || busy" @change="enabled">建立自定义队伍与条件候选</label>
    <div class="field-row">
      <label class="field"><span>本角色已保存方案</span><select v-model="preset" :disabled="disabled || busy"><option value="">选择方案</option><option v-for="p in available" :key="p.ref" :value="p.ref">{{ p.name }}</option></select></label>
      <button type="button" :disabled="!selected || disabled || busy" @click="loadPreset">载入方案</button>
    </div>
    <fieldset v-if="modelValue" :disabled="disabled || busy">
      <p class="hint">基准保留原参考情境。候选应用下列数值；队友只标记增益来源，请按实际技能与装备填写，未填写的增益不会自动补满。静态面板和装备主副属性保留。</p>
      <div class="checks">
        <label class="check"><input type="checkbox" :checked="modelValue.disable_character" @change="update(c => c.disable_character = ($event.target as HTMLInputElement).checked)">关闭角色局内参考增益</label>
        <label class="check"><input type="checkbox" :checked="modelValue.disable_weapon" @change="update(c => c.disable_weapon = ($event.target as HTMLInputElement).checked)">关闭光锥局内参考增益</label>
        <label class="check"><input type="checkbox" :checked="modelValue.disable_equipment" @change="update(c => c.disable_equipment = ($event.target as HTMLInputElement).checked)">关闭套装局内参考增益</label>
      </div>
      <div class="field-grid">
        <label class="field"><span>候选敌人抗性（%，留空沿用参考）</span><input type="number" min="-100" max="100" step="any" :value="modelValue.enemy_resistance ?? ''" @input="update(c => { const v = ($event.target as HTMLInputElement).value; if (v === '') delete c.enemy_resistance; else c.enemy_resistance = Number(v) })"></label>
      </div>
      <div class="field-row">
        <CharacterSelect v-model="teammate" source="catalog" label="添加模拟队友" none="选择队友" :exclude="[characterId, ...modelValue.team.map(t => t.character_id)]" />
        <button type="button" :disabled="!teammate || modelValue.team.length >= 3" @click="addTeammate">添加队友</button>
      </div>
      <ul v-if="modelValue.team.length" class="rows">
        <li v-for="(t, index) in modelValue.team" :key="t.character_id"><div class="row-main"><strong>{{ t.name }}</strong></div><div class="row-actions"><button type="button" class="compact danger" :aria-label="`移除队友${t.name}`" @click="update(c => c.team.splice(index, 1)); source = -1">移除</button></div></li>
      </ul>
      <div class="field-row">
        <label class="field"><span>增益来源</span><select v-model.number="source"><option :value="-1">自身额外增益</option><option v-for="(t, index) in modelValue.team" :key="t.character_id" :value="index">{{ t.name }}</option></select></label>
        <label class="field"><span>增益类型</span><select v-model="field"><option v-for="f in fields" :key="f.key" :value="f.key">{{ f.label }}（{{ f.unit }}）</option></select></label>
        <label class="field"><span>增益数值</span><input v-model.number="amount" type="number" min="0" :max="fields.find(f => f.key === field)?.max" step="any"></label>
        <button type="button" @click="addBonus">应用此增益</button>
      </div>
      <div v-for="row in rows.filter(r => Object.keys(r.bonuses).length)" :key="row.index" class="bonus-group">
        <h4>{{ row.name }}</h4>
        <ul class="rows">
          <li v-for="(value, key) in row.bonuses" :key="key"><div class="row-main"><span>{{ fields.find(f => f.key === key)?.label }}：{{ value }}{{ fields.find(f => f.key === key)?.unit }}</span></div><div class="row-actions"><button type="button" class="compact danger" :aria-label="`移除${row.name}的${fields.find(f => f.key === key)?.label}`" @click="update(c => { const b = row.index < 0 ? c.bonuses : c.team[row.index]!.bonuses; delete b[key] })">移除</button></div></li>
        </ul>
      </div>
      <div class="field-row">
        <label class="field"><span>方案名称</span><input v-model="name" maxlength="40" placeholder="例如：日常队伍"></label>
        <div class="actions"><button type="button" :disabled="!name.trim()" @click="save(true)">保存为新方案</button><button type="button" :disabled="!selected || !name.trim()" @click="save(false)">更新所选方案</button><button type="button" class="danger" :disabled="!selected" @click="confirm?.showModal()">删除所选方案</button></div>
      </div>
    </fieldset>
    <StatusLine kind="error" :text="error" />
    <StatusLine kind="success" :text="notice" />
    <dialog ref="confirm"><h2>删除方案 {{ selected?.name }}</h2><p>只删除本地模拟方案，不修改游戏角色或账号。</p><div class="actions"><button type="button" autofocus :disabled="busy" @click="confirm?.close()">取消</button><button type="button" class="destructive" :disabled="busy" @click="remove">确认删除</button></div></dialog>
  </div>
</template>

<style scoped>
.bonus-group { display: grid; gap: 4px; }
</style>
