<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'
import { useReader } from './reader'

// 挑战排行: one group's self-submitted challenge results, by partition, paged
// 20 at a time, read as soon as the group opens and as the filters change.
// A late response never replaces a newer selection.
export interface Scope { source_protocol: string; bot_id: string; group_id: string }
interface Metric { key: string; label: string; lower: boolean }
interface Kind { id: string; label: string; metrics: Metric[] }
interface Entry { actor_id: string; nickname: string; uid: string; metrics: Record<string, number>; updated_ms: number }
interface Partition { partition: string; items: Entry[]; total: number; next_offset: number | null }
const props = defineProps<{ scope: Scope }>()
const shell = useShell()
const operation = useOperation('群榜操作未完成，请重试。')
const { busy, error, notice, run } = operation
const kinds = ref<Kind[]>([]), kind = ref(''), season = ref(''), dimension = ref(''), offset = ref(0), groups = ref<Partition[]>([]), revision = ref(0), confirming = ref(false), loaded = ref(false)
const selected = computed(() => kinds.value.find(item => item.id === kind.value))
const hasNext = computed(() => groups.value.some(item => item.next_offset !== null))
let target = 0
const reader = useReader(operation, async stale => {
  if (!kinds.value.length) {
    const schema = await shell.invoke<{ kinds: Kind[] }>('challenge.schema')
    kinds.value = schema.kinds; kind.value = schema.kinds[0]?.id ?? ''
  }
  if (!kind.value) return
  const start = target
  const result = await shell.invoke<{ groups: Partition[]; revision: number }>('challenge.list', { scope: props.scope, kind: kind.value, season: season.value, dimension: dimension.value, offset: start })
  if (stale()) return
  groups.value = result.groups; revision.value = result.revision; offset.value = start; loaded.value = true
})
function load(start = 0, delay = 0) { target = start; confirming.value = false; reader.request(delay) }
watch(kind, () => { if (!selected.value?.metrics.some(metric => metric.key === dimension.value)) dimension.value = '' })
async function clear() {
  if (!confirming.value) return
  await run(async () => {
    await shell.invoke('challenge.clear', { scope: props.scope, kind: kind.value, season: season.value, revision: revision.value, confirmed: true })
    groups.value = []; loaded.value = false; confirming.value = false
  }, '已清空所选玩法及期次的成绩。')
}
load()
</script>

<template>
  <section class="subsection" aria-labelledby="challenge-title">
    <h3 id="challenge-title">群 {{ props.scope.group_id }} 的挑战榜</h3>
    <p class="hint">只收录本人在此群主动提交的官方成绩，按区服及官方期次分开。选择玩法、期次或排序后自动按当前规则读取并排列已保存成绩；完整提交方式见本游戏帮助。</p>
    <form class="field-row" @submit.prevent="load()">
      <fieldset :disabled="busy && !reader.loading.value">
        <label class="field"><span>挑战玩法</span><select v-model="kind" @change="load(0, 250)"><option v-for="item in kinds" :key="item.id" :value="item.id">{{ item.label }}</option></select></label>
        <label class="field"><span>官方期次</span><input v-model="season" maxlength="128" placeholder="留空显示已收录期次" @input="load(0, 500)"></label>
        <label class="field"><span>排序指标</span><select v-model="dimension" @change="load(0, 250)"><option value="">综合顺序</option><option v-for="metric in selected?.metrics" :key="metric.key" :value="metric.key">{{ metric.label }}（{{ metric.lower ? '越低越前' : '越高越前' }}）</option></select></label>
        <button type="submit" :disabled="!kind">{{ reader.loading.value ? '正在读取…' : '重新读取' }}</button>
      </fieldset>
    </form>
    <StatusLine v-if="reader.loading.value" kind="info" text="正在读取群榜…" />
    <StatusLine kind="error" :text="error"><template v-if="reader.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="reader.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <EmptyState v-if="loaded && !groups.length" icon="trophy" title="当前条件没有主动提交的成绩" text="换一个玩法或期次，或等群员在聊天中提交成绩。" />
    <template v-for="group in groups" :key="group.partition">
      <h4>{{ group.partition }} · {{ group.total }} 人</h4>
      <div class="table-wrap">
        <table>
          <thead><tr><th class="num">名次</th><th>用户 / UID</th><th v-for="metric in selected?.metrics" :key="metric.key" class="num">{{ metric.label }}</th><th>更新时间</th></tr></thead>
          <tbody>
            <tr v-for="(row, index) in group.items" :key="row.actor_id"><td class="num">{{ offset + index + 1 }}</td><td>{{ row.nickname }}<br><span class="mono">{{ row.uid }}</span></td><td v-for="metric in selected?.metrics" :key="metric.key" class="num">{{ row.metrics[metric.key] ?? '未提供' }}</td><td>{{ formatTime(row.updated_ms) }}</td></tr>
          </tbody>
        </table>
      </div>
    </template>
    <div class="actions"><button type="button" :disabled="busy || offset === 0" @click="load(Math.max(0, offset - 20))">上一页成绩</button><button type="button" :disabled="busy || !hasNext" @click="load(offset + 20)">下一页成绩</button><button type="button" class="danger" :disabled="busy || !loaded || !groups.length" @click="confirming = true">清空所选玩法及期次</button></div>
    <StatusLine v-if="confirming" kind="attention" :text="`将删除此群的 ${selected?.label ?? ''} ${season || '全部期次'} 成绩，需用户重新提交才能恢复。`"><template #actions><button type="button" class="compact" :disabled="busy" @click="confirming = false">保留成绩</button><button type="button" class="compact destructive" :disabled="busy" @click="clear">确认清空挑战成绩</button></template></StatusLine>
  </section>
</template>
