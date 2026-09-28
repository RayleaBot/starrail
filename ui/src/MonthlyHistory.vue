<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import { formatTime, useOperation } from './shell/status'

// 月报历史: the months saved for a role and their summed income.
interface Item { month: string; year_estimated: boolean; saved_ms: number; amounts: Record<string, number> }
const props = defineProps<{ accountRef: string; roleRef: string; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const { busy, error, notice, run } = useOperation('月报操作未完成，请重试。')
const items = ref<Item[]>([]), totals = ref<Record<string, number>>({}), coverage = ref<Record<string, number>>({}), revision = ref(0), month = ref(''), removeMonth = ref(''), view = ref<View | null>(null), stopping = ref(false), loaded = ref(false)
let disposed = false
function selection() { return { account_ref: props.accountRef, role_ref: props.roleRef } }
async function load() {
  const r = await props.invoke<{ items: Item[]; totals: Record<string, number>; coverage: Record<string, number>; revision: number }>('monthly.list', selection())
  if (!disposed) { items.value = r.items; totals.value = r.totals; coverage.value = r.coverage; revision.value = r.revision; loaded.value = true }
}
async function fetchMonths(all = false) {
  stopping.value = false
  await run(async () => {
    let saved = 0
    try {
      const r = await props.invoke<{ month: string; available_months: string[] }>('monthly.fetch', { ...selection(), month: all ? '' : month.value })
      saved++
      if (all) {
        const current = r.month.replace('-', '')
        for (const next of [...new Set(r.available_months)].slice(0, 12)) {
          if (stopping.value || disposed) break
          if (next === current) continue
          await props.invoke('monthly.fetch', { ...selection(), month: next })
          saved++
        }
      }
    } finally {
      if (!disposed) { notice.value = `本次已保存 ${saved} 个月${stopping.value ? '，已停止后续读取' : ''}。`; await load() }
    }
  })
}
async function open(item: Item) { await run(async () => { view.value = null; const r = await props.invoke<{ view: View }>('monthly.get', { ...selection(), month: item.month }); if (!disposed) view.value = r.view }) }
async function remove() { await run(async () => { await props.invoke('monthly.remove', { ...selection(), month: removeMonth.value, revision: revision.value }); removeMonth.value = ''; view.value = null; await load() }, '已移除该月的本地月报。') }
void run(load)
onUnmounted(() => { disposed = true; stopping.value = true })
</script>

<template>
  <section class="subsection" aria-labelledby="monthly-title">
    <h3 id="monthly-title">保存月报与累计收入</h3>
    <p class="hint">只统计已保存月份，未保存或官方已失效的月份不补算。</p>
    <form class="field-row" @submit.prevent="fetchMonths()">
      <label class="field"><span>保存月份</span><input v-model="month" :disabled="busy" placeholder="YYYYMM，留空为默认月份"></label>
      <div class="actions"><button :disabled="busy">读取并保存此月</button><button type="button" :disabled="busy" @click="fetchMonths(true)">保存官方可用月份</button><button v-if="busy" type="button" @click="stopping = true">停止后续月份</button><button type="button" :disabled="busy" @click="run(load)">刷新</button></div>
    </form>
    <StatusLine v-if="!loaded && busy" kind="info" text="正在读取已保存月报…" />
    <StatusLine kind="error" :text="error"><template v-if="!loaded" #actions><button class="compact" type="button" :disabled="busy" @click="run(load)">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <dl v-if="Object.keys(totals).length" class="metrics"><div v-for="(value, key) in totals" :key="key"><dt>{{ key }} · 覆盖 {{ coverage[key] }} 个月</dt><dd>{{ value.toLocaleString() }}</dd></div></dl>
    <ul v-if="items.length" class="rows">
      <li v-for="item in items" :key="item.month">
        <div class="row-main"><strong>{{ item.month }}<span v-if="item.year_estimated" class="muted">（年份推定）</span></strong><div class="row-meta"><span>{{ Object.entries(item.amounts).map(([k, v]) => `${k} ${v.toLocaleString()}`).join(' · ') || '未提供可累计的收入' }}</span><span>保存于 {{ formatTime(item.saved_ms) }}</span></div></div>
        <div class="row-actions"><button class="compact" :disabled="busy" @click="open(item)">查看</button><button class="compact danger" :disabled="busy" @click="removeMonth = item.month">移除</button></div>
        <div v-if="removeMonth === item.month" class="confirm"><p>移除 {{ removeMonth }} 的本地月报？官方不再提供的月份将无法重新获取。</p><div class="actions"><button class="compact" :disabled="busy" @click="removeMonth = ''">保留</button><button class="compact destructive" :disabled="busy" @click="remove">确认移除</button></div></div>
      </li>
    </ul>
    <p v-else-if="loaded && !busy" class="hint">尚未保存月报。</p>
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
