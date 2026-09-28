<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'
import { useReader } from './reader'

// 资产记录: the official asset change log of the selected role, collected
// page by page and kept locally. The saved archive is read as soon as the
// role, category, direction or filter changes; collecting stays explicit.
interface Row { id: string; time: string; action: string; change: string; deduct?: string; item?: string }
interface Archive { rows: Row[]; total: number; next_offset: number | null; totals: Record<string, string>; coverage: Record<string, number>; unknown_amounts: number; revision: number; saved_ms: number; note: string }
interface Job { ref: string; state: string; sequence: number; pages: number; count: number }
interface Category { id: string; label: string }
const shell = useShell()
const operation = useOperation('资产记录操作未完成，请重试。')
const { busy, error, notice, run, clear } = operation
const categories = ref<Category[]>([]), category = ref(''), direction = ref('all'), month = ref(''), query = ref(''), archive = ref<Archive | null>(null), job = ref<Job | null>(null), confirmRemove = ref(false), stopping = ref(false)
let disposed = false
function choice() { return { ...shell.chosen(), category: category.value, direction: direction.value } }
// The month is typed as YYYYMM and sent as the YYYY-MM prefix the archive filters on.
const monthValid = computed(() => month.value.trim() === '' || /^[0-9]{6}$/.test(month.value.trim()))
function filters() { const m = month.value.trim(); return { month: m ? m.slice(0, 4) + '-' + m.slice(4) : '', query: query.value.trim() } }
async function load(offset = 0) {
  const result = await shell.invoke<Archive>('billing.archive.get', { ...choice(), offset, ...filters() })
  if (disposed) return
  archive.value = offset && archive.value ? { ...result, rows: [...archive.value.rows, ...result.rows] } : result
  confirmRemove.value = false
}
const reader = useReader(operation, async stale => {
  if (!categories.value.length) {
    categories.value = (await shell.invoke<{ categories: Category[] }>('billing.schema')).categories
    category.value = categories.value[0]?.id ?? ''
  }
  if (!shell.selectedRole.value || !category.value) return
  const result = await shell.invoke<Archive>('billing.archive.get', { ...choice(), offset: 0, ...filters() })
  if (stale() || disposed) return
  archive.value = result; confirmRemove.value = false
})
async function collect() {
  await run(async () => {
    stopping.value = false
    let current = (await shell.invoke<{ job: Job }>('billing.sync.start', choice())).job
    job.value = current
    try {
      while (!disposed && !stopping.value && current.state === 'running') { current = (await shell.invoke<{ job: Job }>('billing.sync.step', { ref: current.ref, sequence: current.sequence })).job; job.value = current }
      if (current.state === 'completed') { notice.value = `已收集 ${current.pages} 页，档案共 ${current.count} 条记录。`; if (!disposed) await load() }
      else notice.value = '已停止收集，原档案保持不变。'
    } finally {
      await shell.invoke('billing.sync.cancel', { ref: current.ref }).catch(() => {})
      job.value = null
    }
  })
}
async function removeArchive() { await run(async () => { await shell.invoke('billing.archive.remove', { ...choice(), revision: archive.value!.revision, confirm: true }); await load() }) }
watch(shell.selection, () => { archive.value = null; confirmRemove.value = false; clear(); reader.request() })
watch(month, () => { if (monthValid.value) reader.request(500) })
reader.request()
onUnmounted(() => { disposed = true; stopping.value = true })
</script>

<template>
  <section class="section" aria-labelledby="billing-title">
    <div class="section-heading"><div class="titles"><h2 id="billing-title">资产记录</h2><p>读取官方当前可提供的资产变动记录，成功读完才合并保存到本地档案；已保存的档案在选择角色与类别后自动显示。这里的数值是游戏资产变化，不是实际支付金额。</p></div><button type="button" :disabled="busy || !shell.selectedRole.value" @click="reader.request()">{{ reader.loading.value ? '正在读取…' : '刷新' }}</button></div>
    <form class="field-row" @submit.prevent="reader.request()">
      <fieldset :disabled="busy && !reader.loading.value">
        <label class="field"><span>资产类别</span><select v-model="category" @change="confirmRemove = false; reader.request(250)"><option v-for="item in categories" :key="item.id" :value="item.id">{{ item.label }}</option></select></label>
        <label class="field"><span>方向</span><select v-model="direction" @change="confirmRemove = false; reader.request(250)"><option value="all">全部变动</option><option value="produce">获得</option><option value="consume">消耗</option></select></label>
        <div class="actions"><button class="primary" type="button" :disabled="busy || !shell.selectedRole.value" @click="collect">{{ job ? '正在收集…' : '收集官方记录' }}</button></div>
      </fieldset>
      <button v-if="job" type="button" :disabled="stopping" @click="stopping = true">停止收集</button>
    </form>
    <StatusLine v-if="job" kind="info" :text="`已读取 ${job.pages} 页、${job.count} 条；停止后保留原档案。`" />
    <StatusLine v-else-if="reader.loading.value" kind="info" text="正在读取已保存记录…" />
    <StatusLine kind="error" :text="error"><template v-if="reader.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="reader.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <EmptyState v-if="archive && !archive.saved_ms && !archive.total" icon="wallet" title="尚未保存资产档案" text="点击“收集官方记录”读取官方当前可提供的记录并保存到本地，之后可按月份与来源筛选。" />
    <div v-else-if="archive" class="subsection">
      <h3>已保存档案</h3>
      <form class="field-row" @submit.prevent="reader.request()">
        <fieldset :disabled="busy && !reader.loading.value">
          <label class="field"><span>月份</span><input v-model="month" inputmode="numeric" placeholder="YYYYMM，留空为全部" :aria-invalid="monthValid ? undefined : 'true'"><span v-if="!monthValid" class="hint danger-text">月份是六位数字，例如 202609。</span></label>
          <label class="field"><span>来源或物品</span><input v-model="query" maxlength="128" @input="reader.request(500)"></label>
          <button type="submit">筛选档案</button>
        </fieldset>
      </form>
      <p class="hint">符合条件 {{ archive.total }} 条；保存时间：{{ archive.saved_ms ? formatTime(archive.saved_ms) : '尚未保存' }}。{{ archive.note }}</p>
      <dl v-if="Object.keys(archive.totals).length" class="metrics"><div v-for="(total, action) in archive.totals" :key="action"><dt>{{ action || '未注明来源' }} · {{ archive.coverage[action] }} 条有数值记录</dt><dd>{{ total }}</dd></div></dl>
      <p v-if="archive.unknown_amounts" class="hint">另有 {{ archive.unknown_amounts }} 条未提供可统计数值，未计入合计。</p>
      <EmptyState v-if="!archive.rows.length" icon="search" title="没有符合条件的记录" text="换一个月份或关键词。" />
      <div v-else class="table-wrap">
        <table>
          <thead><tr><th>时间</th><th>来源 / 物品</th><th class="num">变动</th><th class="num">扣除</th><th>官方编号</th></tr></thead>
          <tbody><tr v-for="(row, index) in archive.rows" :key="row.id || index"><td>{{ row.time || '未提供' }}</td><td>{{ row.action }} {{ row.item }}</td><td class="num">{{ row.change || '未提供' }}</td><td class="num">{{ row.deduct || '—' }}</td><td class="mono">{{ row.id || '无编号（本轮快照）' }}</td></tr></tbody>
        </table>
      </div>
      <div v-if="archive.next_offset !== null" class="actions"><button type="button" :disabled="busy" @click="run(() => load(archive!.next_offset!))">更多记录</button></div>
      <div class="actions"><label class="check"><input v-model="confirmRemove" type="checkbox" :disabled="busy">清空所选类别与方向的本地档案</label><button class="destructive" type="button" :disabled="busy || !confirmRemove" @click="removeArchive">确认清空</button></div>
    </div>
  </section>
</template>
