<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { useReader } from './reader'
import { intervalText, rarityName, poolNames, type History } from './history'

// 抽卡明细: one archive's records, filtered and paged, with its analytics.
// The first page is read when the archive opens and again as the filters
// change; the version summary is read when its panel is first expanded.
interface Versions { items: { version: string; half: string; total: number; top: number; unknown: number; estimated: number }[]; total: number; unclassified: number; version: string; timezone: string }
const props = defineProps<{ uid: string; region: string; disabled: boolean }>()
const shell = useShell()
const operation = useOperation('记录读取失败，请重新查询。')
const { busy, error } = operation
const pool = ref(''), rank = ref(''), query = ref(''), from = ref(''), to = ref('')
const data = ref<History | null>(null), offset = ref(0), versions = ref<Versions | null>(null)
let target = 0
const reader = useReader(operation, async stale => {
  if (props.disabled) return
  const next = target, revision = next ? data.value?.revision : undefined
  try {
    const result = await shell.invoke<{ history: History }>('gacha.history', { uid: props.uid, region: props.region, pool: pool.value, rank: rank.value, query: query.value, from: from.value, to: to.value, offset: next, limit: 50, ...(revision ? { revision } : {}) })
    if (stale()) return
    data.value = result.history; offset.value = next
  } catch (cause) {
    if (!stale()) data.value = null
    throw cause
  }
})
const versionsReader = useReader(operation, async stale => {
  if (props.disabled) return
  const result = await shell.invoke<Versions>('gacha.versions', { uid: props.uid, region: props.region })
  if (!stale()) versions.value = result
})
const reading = computed(() => reader.loading.value || versionsReader.loading.value)
function load(next = 0, delay = 0) { target = next; reader.request(delay) }
function toggleVersions(event: Event) { if ((event.target as HTMLDetailsElement).open && !versions.value) versionsReader.request() }
// The archive list mounts this while its own operation is still busy, so
// the first read waits until the parent enables it.
watch(() => props.disabled, disabled => { if (!disabled && !data.value) load() }, { immediate: true })
</script>

<template>
  <section class="subsection" aria-labelledby="gacha-history-title">
    <h3 id="gacha-history-title">抽卡明细 · {{ uid }}</h3>
    <p class="hint">仅统计本地保存的记录。间隔按完整档案计算，筛选不会重新起算；历史仍可能存在缺口。</p>
    <details @toggle="toggleVersions">
      <summary>按版本与半期汇总整个档案</summary>
      <p class="hint">独立于下方筛选。按资料中的国服卡池区间换算档案时区；估算起点、版本冲突和未覆盖记录明确列出。展开后自动读取。</p>
      <div class="actions"><button type="button" :disabled="busy || disabled" @click="versionsReader.request()">{{ versionsReader.loading.value ? '正在读取…' : '重新读取版本统计' }}</button></div>
      <template v-if="versions">
        <p class="hint">共 {{ versions.total }} 抽，未分类 {{ versions.unclassified }} 抽 · {{ versions.timezone }} → UTC+8 · {{ versions.version }}</p>
        <div class="table-wrap"><table><thead><tr><th>版本</th><th>半期</th><th class="num">抽数</th><th class="num">已知最高稀有度</th><th class="num">稀有度未知</th><th class="num">涉及估算起点</th></tr></thead><tbody><tr v-for="v in versions.items" :key="v.version + v.half"><td>{{ v.version }}</td><td>{{ v.half }}</td><td class="num">{{ v.total }}</td><td class="num">{{ v.top }}</td><td class="num">{{ v.unknown }}</td><td class="num">{{ v.estimated }}</td></tr></tbody></table></div>
      </template>
    </details>
    <form class="field-row" @submit.prevent="load(0)">
      <fieldset :disabled="(busy && !reading) || disabled">
        <label class="field"><span>明细卡池</span><select v-model="pool" @change="load(0, 250)"><option value="">全部卡池</option><option v-for="(name, id) in poolNames" :key="id" :value="id">{{ name }}</option></select></label>
        <label class="field"><span>稀有度</span><select v-model="rank" @change="load(0, 250)"><option value="">全部稀有度</option><option v-for="value in ['5', '4', '3']" :key="value" :value="value">{{ rarityName(value) }}</option><option value="unknown">未知稀有度</option></select></label>
        <label class="field"><span>物品名或 ID</span><input v-model="query" type="search" maxlength="128" @input="load(0, 500)"></label>
        <label class="field"><span>开始日期</span><input v-model="from" type="date" @change="load(0, 250)"></label>
        <label class="field"><span>结束日期</span><input v-model="to" type="date" @change="load(0, 250)"></label>
        <button type="submit">{{ reader.loading.value ? '正在读取…' : '重新读取' }}</button>
      </fieldset>
    </form>
    <StatusLine v-if="reader.loading.value" kind="info" text="正在读取明细…" />
    <StatusLine kind="error" :text="error"><template v-if="reader.failed.value || versionsReader.failed.value" #actions><button class="compact" type="button" :disabled="busy || disabled" @click="reader.failed.value ? reader.request() : versionsReader.request()">重试</button></template></StatusLine>
    <template v-if="data">
      <dl class="metrics"><div><dt>符合筛选</dt><dd>{{ data.total }} 抽</dd></div><div><dt>最高稀有度占比</dt><dd>{{ data.analytics.top_rate === null ? '无法确定' : data.analytics.top_rate.toFixed(2) + '%' }}</dd></div><div><dt>完整间隔均值</dt><dd>{{ data.analytics.average_interval === null ? '无完整间隔' : data.analytics.average_interval.toFixed(2) + ' 抽' }}</dd></div><div><dt>完整间隔样本</dt><dd>{{ data.analytics.complete_intervals }}</dd></div></dl>
      <p class="hint">占比是筛选结果的记录占比，不代表游戏抽卡概率；首段或稀有度不完整的间隔不计入均值。</p>
      <details>
        <summary>按月份统计</summary>
        <p class="hint">沿用档案区服本地时间。</p>
        <div class="table-wrap"><table><thead><tr><th>月份</th><th class="num">抽数</th><th class="num">最高稀有度</th><th class="num">未知</th></tr></thead><tbody><tr v-for="item in data.analytics.months" :key="item.month"><td>{{ item.month }}</td><td class="num">{{ item.total }}</td><td class="num">{{ item.top }}</td><td class="num">{{ item.unknown }}</td></tr></tbody></table></div>
      </details>
      <details>
        <summary>物品分布 · {{ data.analytics.item_kinds }} 种</summary>
        <p v-if="data.analytics.item_kinds > 100" class="hint">显示数量最多的 100 种，可按物品名查询其余记录。</p>
        <div class="table-wrap"><table><thead><tr><th>物品</th><th>稀有度</th><th class="num">数量</th></tr></thead><tbody><tr v-for="item in data.analytics.items" :key="`${item.id}:${item.rank}`"><td>{{ item.name || item.id }}</td><td>{{ rarityName(item.rank) }}</td><td class="num">{{ item.count }}</td></tr></tbody></table></div>
      </details>
      <div v-if="data.records.length" class="table-wrap"><table><thead><tr><th>时间</th><th>物品</th><th>稀有度</th><th>卡池</th><th>最高稀有度间隔</th></tr></thead><tbody><tr v-for="item in data.records" :key="`${item.gacha_type}:${item.id}`"><td>{{ item.time }}</td><td>{{ item.name || item.item_id }}</td><td>{{ rarityName(item.rank_type ?? '') }}</td><td>{{ poolNames[item.gacha_type] || item.gacha_type }}</td><td>{{ intervalText(item.interval) }}</td></tr></tbody></table></div>
      <EmptyState v-else icon="search" title="没有符合筛选的记录" text="调整卡池、稀有度、物品或日期后自动重新读取。" />
      <div class="actions"><button type="button" :disabled="busy || disabled || offset === 0" @click="load(Math.max(0, offset - 50))">上一页明细</button><span class="hint">第 {{ Math.floor(offset / 50) + 1 }} 页</span><button type="button" :disabled="busy || disabled || data.next_offset === null" @click="load(data.next_offset!)">下一页明细</button><button type="button" :disabled="busy || disabled" @click="load(0)">重新读取第一页</button></div>
    </template>
  </section>
</template>
