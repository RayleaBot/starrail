<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'

// 资产记录: the official asset change log of the selected role, collected page by page and kept locally.
interface Row { id: string; time: string; action: string; change: string; deduct?: string; item?: string }
interface Archive { rows: Row[]; total: number; next_offset: number | null; totals: Record<string, string>; coverage: Record<string, number>; unknown_amounts: number; revision: number; saved_ms: number; note: string }
interface Job { ref: string; state: string; sequence: number; pages: number; count: number }
const shell = useShell()
const { busy, error, notice, run, clear } = useOperation('资产记录操作未完成，请重试。')
const categories = ref<{ id: string; label: string }[]>([]), category = ref(''), direction = ref('all'), month = ref(''), query = ref(''), archive = ref<Archive | null>(null), job = ref<Job | null>(null), confirmRemove = ref(false), stopping = ref(false)
let disposed = false
function choice() { return { ...shell.chosen(), category: category.value, direction: direction.value } }
async function load(offset = 0) {
  const result = await shell.invoke<Archive>('billing.archive.get', { ...choice(), offset, month: month.value, query: query.value.trim() })
  archive.value = offset && archive.value ? { ...result, rows: [...archive.value.rows, ...result.rows] } : result
  confirmRemove.value = false
}
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
watch([shell.selection, category, direction], () => { archive.value = null; confirmRemove.value = false; clear() })
void run(async () => { categories.value = (await shell.invoke<{ categories: { id: string; label: string }[] }>('billing.schema')).categories; category.value = categories.value[0]?.id ?? '' })
onUnmounted(() => { disposed = true; stopping.value = true })
</script>

<template>
  <section class="section" aria-labelledby="billing-title">
    <div class="section-heading"><div class="titles"><h2 id="billing-title">资产记录</h2><p>读取官方当前可提供的资产变动记录，成功读完才合并保存到本地档案。这里的数值是游戏资产变化，不是实际支付金额。</p></div></div>
    <form class="field-row" @submit.prevent="run(() => load())">
      <fieldset :disabled="busy">
        <label class="field"><span>资产类别</span><select v-model="category"><option v-for="item in categories" :key="item.id" :value="item.id">{{ item.label }}</option></select></label>
        <label class="field"><span>方向</span><select v-model="direction"><option value="all">全部变动</option><option value="produce">获得</option><option value="consume">消耗</option></select></label>
        <div class="actions"><button type="submit" :disabled="!shell.selectedRole.value">查看已保存记录</button><button class="primary" type="button" :disabled="!shell.selectedRole.value" @click="collect">{{ job ? '正在收集…' : '收集官方记录' }}</button></div>
      </fieldset>
      <button v-if="job" type="button" :disabled="stopping" @click="stopping = true">停止收集</button>
    </form>
    <StatusLine v-if="job" kind="info" :text="`已读取 ${job.pages} 页、${job.count} 条；停止后保留原档案。`" />
    <StatusLine kind="error" :text="error" />
    <StatusLine kind="success" :text="notice" />
    <div v-if="archive" class="subsection">
      <h3>已保存档案</h3>
      <form class="field-row" @submit.prevent="run(() => load())">
        <fieldset :disabled="busy">
          <label class="field"><span>月份</span><input v-model="month" type="month"></label>
          <label class="field"><span>来源或物品</span><input v-model="query" maxlength="128"></label>
          <button type="submit">筛选档案</button>
        </fieldset>
      </form>
      <p class="hint">符合条件 {{ archive.total }} 条；保存时间：{{ archive.saved_ms ? formatTime(archive.saved_ms) : '尚未保存' }}。{{ archive.note }}</p>
      <dl v-if="Object.keys(archive.totals).length" class="metrics"><div v-for="(total, action) in archive.totals" :key="action"><dt>{{ action || '未注明来源' }} · {{ archive.coverage[action] }} 条有数值记录</dt><dd>{{ total }}</dd></div></dl>
      <p v-if="archive.unknown_amounts" class="hint">另有 {{ archive.unknown_amounts }} 条未提供可统计数值，未计入合计。</p>
      <div class="table-wrap">
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
