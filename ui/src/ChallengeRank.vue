<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'

// 挑战排行: one group's self-submitted challenge results, by partition, paged
// 20 at a time. A late response never replaces a newer selection.
export interface Scope { source_protocol: string; source_adapter: string; bot_id: string; group_id: string }
interface Metric { key: string; label: string; lower: boolean }
interface Kind { id: string; label: string; metrics: Metric[] }
interface Entry { actor_id: string; nickname: string; uid: string; metrics: Record<string, number>; updated_ms: number }
interface Partition { partition: string; items: Entry[]; total: number; next_offset: number | null }
const props = defineProps<{ scope: Scope }>()
const shell = useShell()
const { busy, error, notice, run } = useOperation('群榜操作未完成，请重试。')
const kinds = ref<Kind[]>([]), kind = ref(''), season = ref(''), dimension = ref(''), offset = ref(0), groups = ref<Partition[]>([]), revision = ref(0), confirming = ref(false), loaded = ref(false)
const selected = computed(() => kinds.value.find(item => item.id === kind.value))
const hasNext = computed(() => groups.value.some(item => item.next_offset !== null))
let generation = 0
watch([() => props.scope, kind, season, dimension], () => { generation++; groups.value = []; offset.value = 0; confirming.value = false; loaded.value = false })
async function load(start = 0) {
  await run(async () => {
    const current = ++generation
    confirming.value = false
    try {
      const result = await shell.invoke<{ groups: Partition[]; revision: number }>('challenge.list', { scope: props.scope, kind: kind.value, season: season.value, dimension: dimension.value, offset: start })
      if (current !== generation) return
      groups.value = result.groups; revision.value = result.revision; offset.value = start; loaded.value = true
    } catch (cause) {
      if (current === generation) throw cause
    }
  })
}
async function clear() {
  if (!confirming.value) return
  await run(async () => {
    await shell.invoke('challenge.clear', { scope: props.scope, kind: kind.value, season: season.value, revision: revision.value, confirmed: true })
    groups.value = []; loaded.value = false; confirming.value = false
  }, '已清空所选玩法及期次的成绩。')
}
void run(async () => {
  const result = await shell.invoke<{ kinds: Kind[] }>('challenge.schema')
  kinds.value = result.kinds; kind.value = result.kinds[0]?.id ?? ''
})
onUnmounted(() => { generation++ })
</script>

<template>
  <section class="subsection" aria-labelledby="challenge-title">
    <h3 id="challenge-title">群 {{ props.scope.group_id }} 的挑战榜</h3>
    <p class="hint">只收录本人在此群主动提交的官方成绩，按区服及官方期次分开。重新读取会按当前规则排列已保存成绩；完整提交方式见本游戏帮助。</p>
    <form class="field-row" @submit.prevent="load()">
      <fieldset :disabled="busy">
        <label class="field"><span>挑战玩法</span><select v-model="kind"><option v-for="item in kinds" :key="item.id" :value="item.id">{{ item.label }}</option></select></label>
        <label class="field"><span>官方期次</span><input v-model="season" maxlength="128" placeholder="留空显示已收录期次"></label>
        <label class="field"><span>排序指标</span><select v-model="dimension"><option value="">综合顺序</option><option v-for="metric in selected?.metrics" :key="metric.key" :value="metric.key">{{ metric.label }}（{{ metric.lower ? '越低越前' : '越高越前' }}）</option></select></label>
        <button class="primary" type="submit" :disabled="!kind">{{ busy ? '正在读取…' : '读取或重排群榜' }}</button>
      </fieldset>
    </form>
    <StatusLine kind="error" :text="error" />
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
