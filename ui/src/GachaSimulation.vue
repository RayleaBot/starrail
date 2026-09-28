<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'

// 模拟抽卡: the management page's own simulation archive, drawn against a
// reference banner; a draw keeps its request ref so a retry never double-counts.
type Kind = 'character' | 'weapon' | 'standard'
interface Selection { kind: Kind; banner_id: string; featured: string; fate_target: string; fate_limit: number }
interface Banner { id: string; version: string; half: string; from: string; characters5: string[]; weapons5: string[] }
interface Draw { name: string; rarity: number; kind: Kind; featured: boolean; guaranteed: boolean; fate: boolean; interval: number; index: number; time_ms: number }
interface Status { model: string; active: Kind; selections: Record<Kind, Selection>; pools: Record<Kind, { five: number; four: number; up_five: boolean; up_four: boolean; fate: number }>; deck: { banners: Banner[] }; total: number; daily_used: number; daily_limit: number; daily_remaining: number; history: Draw[]; owned: Record<string, number>; batch: { draws: Draw[]; selection: Selection } | null }
const shell = useShell()
const { busy, error, notice, run } = useOperation('模拟操作未完成。')
const status = ref<Status | null>(null), draft = ref<Selection>({ kind: 'character', banner_id: '', featured: '', fate_target: '', fate_limit: 2 }), quota = ref(100), last = ref<Draw[]>([]), confirmReset = ref(false), pending = ref<{ count: number; request_ref: string } | null>(null)
const names: Record<Kind, string> = { character: '角色', weapon: '光锥', standard: '常驻' }
const banner = computed(() => status.value?.deck.banners.find(item => item.id === draft.value.banner_id))
const dirty = computed(() => { const current = status.value?.selections[draft.value.kind]; return !current || status.value!.active !== draft.value.kind || current.banner_id !== draft.value.banner_id || current.featured !== draft.value.featured })
const inventory = computed(() => Object.entries(status.value?.owned ?? {}).sort((a, b) => b[1] - a[1]))
watch(() => draft.value.kind, kind => { if (status.value) draft.value = { ...status.value.selections[kind] }; last.value = [] })
watch([() => draft.value.kind, () => draft.value.banner_id, () => draft.value.featured, () => draft.value.fate_target, () => draft.value.fate_limit], () => { last.value = [] })
function chooseBanner() { if (!banner.value) return; draft.value.featured = draft.value.kind === 'character' ? banner.value.characters5[0]! : banner.value.weapons5[0]!; draft.value.fate_target = ''; last.value = [] }
function accept(value: Status) { status.value = value; draft.value = { ...value.selections[value.active] }; quota.value = value.daily_limit; if (value.batch) last.value = value.batch.draws }
async function load() { accept(await shell.invoke<Status>('simulation.status')) }
async function select() { await run(async () => { accept(await shell.invoke<Status>('simulation.select', { ...draft.value })) }, '模拟卡池已保存。') }
async function draw(count: number) { await run(async () => { if (!pending.value) last.value = []; pending.value ??= { count, request_ref: globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random()}` }; accept(await shell.invoke<Status>('simulation.draw', { ...pending.value })); pending.value = null }) }
async function endRetry() { await run(async () => { await load(); pending.value = null }, '已读取实际保存的模拟统计。本次尝试已结束，已完成的模拟抽数保留。') }
async function reset() { if (!confirmReset.value) return; await run(async () => { accept(await shell.invoke<Status>('simulation.reset', { confirm: true })); last.value = []; confirmReset.value = false }, '模拟记录与保底已重置，当天已用次数保留。') }
async function configure() { await run(async () => { accept(await shell.invoke<Status>('simulation.configure', { daily_limit: Number(quota.value) })) }, '每日模拟额度已保存。') }
void run(load)
</script>

<template>
  <section class="section" aria-labelledby="simulation-title">
    <div class="section-heading"><div class="titles"><h2 id="simulation-title">模拟抽卡</h2><p>固定参考娱乐模型，不代表官方概率、保底规则或当前卡池。不消耗真实货币，模拟记录不会进入 UIGF。管理页使用独立档案，与聊天模拟分开。</p></div><button type="button" :disabled="busy || !!pending" @click="run(load)">刷新模拟统计</button></div>
    <StatusLine v-if="busy && !status" kind="info" text="正在读取模拟统计…" />
    <StatusLine kind="error" :text="error"><template v-if="!status" #actions><button class="compact" type="button" :disabled="busy" @click="run(load)">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <template v-if="status">
      <dl class="metrics"><div><dt>今日剩余</dt><dd>{{ status.daily_remaining }} / {{ status.daily_limit }} 抽</dd></div></dl>
      <p class="hint">参考模型 <span class="mono">{{ status.model }}</span>；额度按国服时间 04:00 重置。</p>
      <form class="field-grid" @submit.prevent="select">
        <fieldset :disabled="busy || !!pending">
          <label class="field"><span>模拟种类</span><select v-model="draft.kind"><option v-for="(name, kind) in names" :key="kind" :value="kind">{{ name }}</option></select></label>
          <label v-if="draft.kind !== 'standard'" class="field"><span>参考期次</span><select v-model="draft.banner_id" @change="chooseBanner"><option v-for="item in status.deck.banners" :key="item.id" :value="item.id">{{ item.version }} · {{ item.half }} · {{ item.from.slice(0, 10) }}</option></select></label>
          <label v-if="draft.kind === 'character' || draft.kind === 'weapon'" class="field"><span>模拟目标</span><select v-model="draft.featured"><option v-for="name in (draft.kind === 'character' ? banner?.characters5 : banner?.weapons5) ?? []" :key="name" :value="name">{{ name }}</option></select></label>
          <div class="actions"><button type="submit" :disabled="!dirty">保存模拟卡池</button><span v-if="dirty" class="hint">先保存卡池选择，再进行模拟。</span></div>
        </fieldset>
      </form>
      <div class="actions"><button class="primary" type="button" :disabled="busy || dirty || !!pending || status.daily_remaining < 1" @click="draw(1)">模拟单抽</button><button class="primary" type="button" :disabled="busy || dirty || !!pending || status.daily_remaining < 10" @click="draw(10)">模拟十连</button></div>
      <StatusLine v-if="pending" kind="attention" text="上次请求结果尚未确认，可用同一请求重试以避免重复计数。"><template #actions><div class="actions"><button class="compact" type="button" :disabled="busy" @click="draw(pending!.count)">重试同一模拟请求</button><button class="compact" type="button" :disabled="busy" @click="endRetry">刷新统计并结束本次尝试</button></div></template></StatusLine>
      <ul v-if="last.length" class="draw-results" aria-label="本次模拟结果"><li v-for="(item, index) in last" :key="index"><strong>{{ item.name }}</strong><span>{{ item.rarity }} 星{{ item.featured ? ' · 目标池' : '' }}{{ item.fate ? ' · 定轨' : '' }}{{ item.guaranteed ? ' · 保证目标池' : '' }}</span></li></ul>
      <div class="table-wrap"><table><thead><tr><th>卡池</th><th class="num">连续未出五星</th><th class="num">连续未出四星</th><th>下次五星</th></tr></thead><tbody><tr v-for="(name, kind) in names" :key="kind"><td>{{ name }}</td><td class="num">{{ status.pools[kind].five }} 抽</td><td class="num">{{ status.pools[kind].four }} 抽</td><td>{{ status.pools[kind].up_five ? '保证来自目标池' : '按参考模型判定' }}</td></tr></tbody></table></div>
      <details>
        <summary>模拟获得物品 · {{ inventory.length }} 种</summary>
        <div class="table-wrap"><table><thead><tr><th>物品</th><th class="num">数量</th></tr></thead><tbody><tr v-for="[name, count] in inventory" :key="name"><td>{{ name }}</td><td class="num">{{ count }}</td></tr></tbody></table></div>
      </details>
      <details>
        <summary>最近模拟历史 · {{ status.history.length }} 抽</summary>
        <div class="table-wrap"><table><thead><tr><th>时间</th><th>物品</th><th>稀有度</th><th>种类</th></tr></thead><tbody><tr v-for="(item, index) in [...status.history].reverse()" :key="index"><td>{{ formatTime(item.time_ms) }}</td><td>{{ item.name }}</td><td>{{ item.rarity }} 星</td><td>{{ names[item.kind] }}</td></tr></tbody></table></div>
      </details>
      <div class="subsection">
        <h3>额度与重置</h3>
        <form class="field-row" @submit.prevent="configure">
          <fieldset :disabled="busy || !!pending">
            <label class="field"><span>每日模拟额度</span><input v-model.number="quota" type="number" min="10" max="1000" step="1" required></label>
            <button type="submit">保存模拟额度</button>
          </fieldset>
        </form>
        <p class="hint">额度适用于本游戏每个模拟档案。重置历史不会恢复当天已用次数。</p>
        <label class="check"><input v-model="confirmReset" type="checkbox" :disabled="busy || !!pending">清空当前模拟档案的历史、物品和保底</label>
        <div class="actions"><button type="button" class="destructive" :disabled="busy || !!pending || !confirmReset" @click="reset">确认重置模拟</button></div>
      </div>
    </template>
  </section>
</template>

<style scoped>
/* Draw results: one small tile per pull, in reading order. */
.draw-results { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 8px; padding: 0; margin: 0; list-style: none; }
.draw-results li { display: grid; gap: 2px; padding: 10px 12px; border: 1px solid var(--line); border-radius: var(--radius-md); background: var(--space-800); min-width: 0; }
.draw-results strong { font-weight: 600; overflow-wrap: anywhere; }
.draw-results span { font-size: 12px; color: var(--ivory-muted); }
</style>
