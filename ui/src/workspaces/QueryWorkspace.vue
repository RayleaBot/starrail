<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import StatusLine from '../shell/StatusLine.vue'
import EmptyState from '../shell/EmptyState.vue'
import { regions, useShell } from '../shell/context'
import { useOperation } from '../shell/status'
import { useReader } from '../reader'
import BusinessView, { type View } from '../BusinessView.vue'
import PanelDetails, { type Panel } from '../PanelDetails.vue'
import BuildCalculator from '../BuildCalculator.vue'
import MonthlyHistory from '../MonthlyHistory.vue'
import PanelHistory from '../PanelHistory.vue'
import GrowthCalculator from '../GrowthCalculator.vue'

const shell = useShell()
const sections: Section[] = [
  { id: 'query', label: '游戏查询', icon: 'search' },
  { id: 'showcase', label: '公开展柜', icon: 'user' },
  { id: 'history', label: '面板历史', icon: 'history' },
  { id: 'growth', label: '养成计算', icon: 'growth' },
]

// 游戏查询: one operation of the selected role, with the results and the
// tools that read the same panels (score, skills, build comparison). The
// query runs by itself once the role and the operation's choices are set;
// the button only repeats it or submits a typed character name.
const query = useOperation('查询未完成，请重试。')
const activeSection = ref('')
const operationName = ref(shell.game.value?.operations[0]?.name ?? '')
const period = ref(1), queryMonth = ref(''), summaryPool = ref('11'), character = ref('')
const view = ref<View | null>(null)
const panels = ref<Panel[]>([]), scoreCharacter = ref('')
const selectedOperation = computed(() => shell.game.value?.operations.find(item => item.name === operationName.value))
const inputKind = computed(() => selectedOperation.value?.input ?? '')
const detailPanel = computed(() => panels.value.find(panel => panel.id === scoreCharacter.value))
const monthValid = computed(() => queryMonth.value.trim() === '' || /^\d{6}$/.test(queryMonth.value.trim()))
watch(selectedOperation, value => { period.value = value?.input === 'rogue_period' ? 3 : 1; queryMonth.value = ''; character.value = ''; panels.value = []; view.value = null })
watch(shell.selection, () => { panels.value = []; view.value = null })
watch([period, queryMonth, summaryPool], () => { view.value = null })

async function resolveCharacter(name: string): Promise<string> {
  if (/^\d+$/.test(name)) return name
  const found = await shell.invoke<{ entries: { id: string; name: string }[] }>('catalog.search', { query: name, kind: 'character' })
  const exact = found.entries.filter(entry => entry.name === name || entry.id === name)
  if (exact.length === 1) return exact[0]!.id
  if (found.entries.length === 1) return found.entries[0]!.id
  throw new Error('角色名未唯一匹配，请在图鉴中确认角色 ID。')
}
const reader = useReader(query, async stale => {
  view.value = null
  const input: Record<string, unknown> = {}
  const kind = inputKind.value
  if (kind === 'starrail_pool') input.gacha_type = summaryPool.value
  if (kind === 'period' || kind === 'rogue_period') input.schedule_type = period.value
  if (kind === 'year_month' && queryMonth.value.trim()) input.month = queryMonth.value.trim()
  if (kind === 'characters_optional' && character.value.trim()) input.character_ids = [await resolveCharacter(character.value.trim())]
  const result = await shell.invoke<{ view: View; panels?: Panel[] }>('query', { ...shell.chosen(), operation: operationName.value, input })
  if (stale()) return
  view.value = result.view; panels.value = result.panels ?? []; scoreCharacter.value = panels.value[0]?.id ?? ''
})
// The read is automatic while the section is shown and a role is selected:
// choices from a select run at once, a typed month runs once it is complete
// and an empty character name shows the overview; a typed name is submitted.
const ready = computed(() => activeSection.value === 'query' && !!shell.selectedRole.value)
watch([ready, shell.selection, operationName, period, summaryPool], ([ok]) => { if (ok) reader.request(100) })
watch(queryMonth, () => { if (ready.value && monthValid.value) reader.request(600) })
watch(character, value => { if (ready.value && value.trim() === '') reader.request(300) })
function runQuery() { if (shell.selectedRole.value) reader.request() }
async function scorePanel() {
  await query.run(async () => {
    const result = await shell.invoke<{ view: View; panels: Panel[] }>('panel.score', { ...shell.chosen(), character_id: scoreCharacter.value })
    view.value = result.view; panels.value = result.panels
  })
}
async function setDefault() { await query.run(async () => { await shell.invoke('accounts.select', shell.chosen()) }, '已设为该用户的默认角色。') }

// 公开展柜: reads that need no bound account.
const showcase = useOperation('展柜读取未完成，请重试。')
const publicUID = ref(''), publicView = ref<View | null>(null)
const profileUID = ref(''), profileRegion = ref(shell.game.value?.region ?? 'prod_gf_cn'), profileView = ref<View | null>(null)
async function readShowcase() { await showcase.run(async () => { publicView.value = (await shell.invoke<{ view: View }>('showcase', { uid: publicUID.value.trim() })).view }) }
async function readProfile() { await showcase.run(async () => { profileView.value = (await shell.invoke<{ view: View }>('public.query', { uid: profileUID.value.trim(), region: profileRegion.value })).view }) }
</script>

<template>
  <Workspace title="查询" description="用已授权的角色查询战绩、便笺与月报，或读取公开展柜；选好内容后结果直接显示在表单下方。" :sections="sections" roles @section="activeSection = $event">
    <template #default="{ active }">
      <section v-if="active === 'query'" class="section" aria-labelledby="query-title">
        <div class="section-heading"><div class="titles"><h2 id="query-title">游戏查询</h2><p>查询内容对应聊天里的同名命令，选择后自动查询；带角色的查询会附上装备评分、技能与配装计算。</p></div></div>
        <form class="field-grid" @submit.prevent="runQuery">
          <label class="field"><span>查询内容</span><select v-model="operationName"><option v-for="item in shell.game.value?.operations" :key="item.name" :value="item.name">{{ item.label }}</option></select></label>
          <label v-if="inputKind === 'period' || inputKind === 'rogue_period'" class="field"><span>期数</span><select v-model="period"><option :value="1">本期</option><option :value="2">上期</option><option v-if="inputKind === 'rogue_period'" :value="3">接口默认范围</option></select></label>
          <label v-if="inputKind === 'year_month'" class="field"><span>月份</span><input v-model="queryMonth" inputmode="numeric" placeholder="YYYYMM，留空为默认月份" :aria-invalid="monthValid ? undefined : 'true'"><span v-if="!monthValid" class="hint danger-text">月份是六位数字，例如 202609。</span></label>
          <label v-if="inputKind === 'starrail_pool'" class="field"><span>卡池</span><select v-model="summaryPool"><option value="11">角色活动跃迁</option><option value="12">光锥活动跃迁</option><option value="21">联动角色跃迁</option><option value="22">联动光锥跃迁</option><option value="1">常驻跃迁</option><option value="2">始发跃迁</option></select></label>
          <label v-if="inputKind === 'characters_optional'" class="field"><span>角色名称或 ID</span><input v-model="character" placeholder="留空显示角色概览，输入后按查询"></label>
          <div class="actions"><button :class="inputKind === 'characters_optional' && character.trim() ? 'primary' : ''" type="submit" :disabled="!shell.selectedRole.value || query.busy.value">{{ query.busy.value ? '正在查询…' : view ? '重新查询' : '查询' }}</button><button type="button" :disabled="!shell.selectedRole.value || query.busy.value" @click="setDefault">设为默认角色</button><span v-if="query.busy.value" class="hint" role="status">正在查询 {{ selectedOperation?.label }}…</span></div>
        </form>
        <StatusLine kind="error" :text="query.error.value" />
        <StatusLine kind="success" :text="query.notice.value" />
        <BusinessView v-if="view" :value="view" />
        <MonthlyHistory v-if="operationName.endsWith('.monthly') && shell.selectedRole.value" :key="shell.selection.value" :account-ref="shell.selectedRole.value.account.ref" :role-ref="shell.selectedRole.value.role.ref" :invoke="shell.invoke" />
        <div v-if="panels.length" class="subsection">
          <h3>装备评分</h3>
          <p class="hint">按角色专属权重比较装备词条，权重随当前属性、星魂与配装自动选择；评分不代表实际伤害。</p>
          <form class="field-row" @submit.prevent="scorePanel">
            <label class="field"><span>评分角色</span><select v-model="scoreCharacter" :disabled="query.busy.value" @change="scorePanel"><option v-for="panel in panels" :key="panel.id" :value="panel.id">{{ panel.name }}</option></select></label>
            <button type="submit" :disabled="query.busy.value || !scoreCharacter">计算装备评分</button>
          </form>
        </div>
        <PanelDetails v-if="detailPanel" :panel="detailPanel" />
        <BuildCalculator v-if="detailPanel && shell.selectedRole.value" :key="`build:${shell.selection.value}:${detailPanel.id}`" :character-id="detailPanel.id" :character-name="detailPanel.name" :account-ref="shell.selectedRole.value.account.ref" :role-ref="shell.selectedRole.value.role.ref" :invoke="shell.invoke" />
        <div v-if="!view && query.busy.value" class="skeleton" aria-hidden="true"><span></span><span></span><span></span></div>
        <EmptyState v-else-if="!shell.selectedRole.value" icon="user" title="选择角色后自动查询" text="在右上角选择一个已绑定的角色，查询内容会立即执行；结果与装备评分、技能和配装计算显示在这里。" />
        <EmptyState v-else-if="!view && !query.error.value" icon="search" title="等待查询条件" text="补全上方的条件后自动查询；角色相关的查询还会列出装备评分、技能与配装计算。" />
      </section>

      <section v-else-if="active === 'showcase'" class="section" aria-labelledby="showcase-title">
        <div class="section-heading"><div class="titles"><h2 id="showcase-title">公开展柜</h2><p>读取游戏内角色展柜并保存为该 UID 的面板，不需要绑定账号。</p></div></div>
        <form class="field-row" @submit.prevent="readShowcase">
          <label class="field"><span>游戏 UID</span><input v-model="publicUID" required inputmode="numeric" pattern="[0-9]{6,12}" placeholder="输入游戏 UID" :disabled="showcase.busy.value"></label>
          <button class="primary" type="submit" :disabled="showcase.busy.value">读取展柜</button>
        </form>
        <StatusLine kind="error" :text="showcase.error.value" />
        <BusinessView v-if="publicView" :value="publicView" />
        <div class="subsection">
          <h3>公共查询池的基础资料</h3>
          <p class="hint">只使用账号所有者主动开启的公共查询用途；没有可用账号或额度时会明确提示，不会借用其他私人账号。</p>
          <form class="field-row" @submit.prevent="readProfile">
            <label class="field"><span>公开资料 UID</span><input v-model="profileUID" inputmode="numeric" pattern="[0-9]{6,12}" required :disabled="showcase.busy.value"></label>
            <label class="field"><span>区服</span><select v-model="profileRegion" :disabled="showcase.busy.value"><option v-for="region in regions.filter(r => r.id.endsWith('_cn'))" :key="region.id" :value="region.id">{{ region.name }}</option></select></label>
            <button type="submit" :disabled="showcase.busy.value">查询基础资料</button>
          </form>
          <BusinessView v-if="profileView" :value="profileView" />
        </div>
      </section>

      <PanelHistory v-else-if="active === 'history'" />
      <GrowthCalculator v-else-if="active === 'growth'" />
    </template>
  </Workspace>
</template>
