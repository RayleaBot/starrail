<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import StatusLine from '../shell/StatusLine.vue'
import EmptyState from '../shell/EmptyState.vue'
import { regions, useShell } from '../shell/context'
import { useOperation } from '../shell/status'
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
// tools that read the same panels (score, skills, build comparison).
const query = useOperation('查询未完成，请重试。')
const operationName = ref(shell.game.value?.operations[0]?.name ?? '')
const period = ref(1), queryMonth = ref(''), summaryPool = ref('11'), character = ref('')
const view = ref<View | null>(null)
const panels = ref<Panel[]>([]), scoreCharacter = ref('')
const selectedOperation = computed(() => shell.game.value?.operations.find(item => item.name === operationName.value))
const detailPanel = computed(() => panels.value.find(panel => panel.id === scoreCharacter.value))
watch(selectedOperation, value => { period.value = value?.input === 'rogue_period' ? 3 : 1; queryMonth.value = ''; panels.value = []; view.value = null })
watch(shell.selection, () => { panels.value = []; view.value = null })
watch([period, queryMonth], () => { view.value = null })

async function resolveCharacter(name: string): Promise<string> {
  if (/^\d+$/.test(name)) return name
  const found = await shell.invoke<{ entries: { id: string; name: string }[] }>('catalog.search', { query: name, kind: 'character' })
  const exact = found.entries.filter(entry => entry.name === name || entry.id === name)
  if (exact.length === 1) return exact[0]!.id
  if (found.entries.length === 1) return found.entries[0]!.id
  throw new Error('角色名未唯一匹配，请在图鉴中确认角色 ID。')
}
async function runQuery() {
  await query.run(async () => {
    view.value = null
    const input: Record<string, unknown> = {}
    const kind = selectedOperation.value?.input ?? ''
    if (kind === 'starrail_pool') input.gacha_type = summaryPool.value
    if (kind === 'period' || kind === 'rogue_period') input.schedule_type = period.value
    if (kind === 'year_month' && queryMonth.value.trim()) input.month = queryMonth.value.trim()
    if (kind === 'characters_optional' && character.value.trim()) input.character_ids = [await resolveCharacter(character.value.trim())]
    const result = await shell.invoke<{ view: View; panels?: Panel[] }>('query', { ...shell.chosen(), operation: operationName.value, input })
    view.value = result.view; panels.value = result.panels ?? []; scoreCharacter.value = panels.value[0]?.id ?? ''
  })
}
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
  <Workspace title="查询" description="用已授权的角色查询战绩、便笺与月报，或读取公开展柜；结果直接显示在表单下方。" :sections="sections" roles>
    <template #default="{ active }">
      <section v-if="active === 'query'" class="section" aria-labelledby="query-title">
        <div class="section-heading"><div class="titles"><h2 id="query-title">游戏查询</h2><p>查询内容对应聊天里的同名命令；带角色的查询会附上装备评分、技能与配装计算。</p></div></div>
        <form class="field-grid" @submit.prevent="runQuery">
          <fieldset :disabled="query.busy.value">
            <label class="field"><span>查询内容</span><select v-model="operationName"><option v-for="item in shell.game.value?.operations" :key="item.name" :value="item.name">{{ item.label }}</option></select></label>
            <label v-if="selectedOperation?.input === 'period' || selectedOperation?.input === 'rogue_period'" class="field"><span>期数</span><select v-model="period"><option :value="1">本期</option><option :value="2">上期</option><option v-if="selectedOperation?.input === 'rogue_period'" :value="3">接口默认范围</option></select></label>
            <label v-if="selectedOperation?.input === 'year_month'" class="field"><span>月份</span><input v-model="queryMonth" inputmode="numeric" placeholder="YYYYMM，留空为默认月份"></label>
            <label v-if="selectedOperation?.input === 'starrail_pool'" class="field"><span>卡池</span><select v-model="summaryPool"><option value="11">角色活动跃迁</option><option value="12">光锥活动跃迁</option><option value="21">联动角色跃迁</option><option value="22">联动光锥跃迁</option><option value="1">常驻跃迁</option><option value="2">始发跃迁</option></select></label>
            <label v-if="selectedOperation?.input === 'characters_optional'" class="field"><span>角色名称或 ID</span><input v-model="character" placeholder="留空显示角色概览"></label>
            <div class="actions"><button class="primary" type="submit" :disabled="!shell.selectedRole.value">{{ query.busy.value ? '正在查询…' : '查询' }}</button><button type="button" :disabled="!shell.selectedRole.value" @click="setDefault">设为默认角色</button></div>
          </fieldset>
        </form>
        <StatusLine kind="error" :text="query.error.value" />
        <StatusLine kind="success" :text="query.notice.value" />
        <div v-if="panels.length" class="subsection">
          <h3>装备评分</h3>
          <p class="hint">按角色专属权重比较装备词条，权重随当前属性、星魂与配装自动选择；评分不代表实际伤害。</p>
          <form class="field-row" @submit.prevent="scorePanel">
            <label class="field"><span>评分角色</span><select v-model="scoreCharacter" :disabled="query.busy.value"><option v-for="panel in panels" :key="panel.id" :value="panel.id">{{ panel.name }}</option></select></label>
            <button type="submit" :disabled="query.busy.value || !scoreCharacter">计算装备评分</button>
          </form>
        </div>
        <BusinessView v-if="view" :value="view" />
        <MonthlyHistory v-if="operationName.endsWith('.monthly') && shell.selectedRole.value" :key="shell.selection.value" :account-ref="shell.selectedRole.value.account.ref" :role-ref="shell.selectedRole.value.role.ref" :invoke="shell.invoke" />
        <PanelDetails v-if="detailPanel" :panel="detailPanel" />
        <BuildCalculator v-if="detailPanel && shell.selectedRole.value" :key="`build:${shell.selection.value}:${detailPanel.id}`" :character-id="detailPanel.id" :character-name="detailPanel.name" :account-ref="shell.selectedRole.value.account.ref" :role-ref="shell.selectedRole.value.role.ref" :invoke="shell.invoke" />
        <EmptyState v-if="!view && !query.busy.value" icon="search" title="选择查询内容后开始" text="结果会显示在这里；角色相关的查询还会列出装备评分、技能与配装计算。" />
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
