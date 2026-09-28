<script setup lang="ts">
import { ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { useReader } from './reader'

// 在线时长: play sessions inferred from the stamina recovery rhythm in the
// saved asset records. The record categories are read with the role; the
// estimate follows the chosen recovery category.
interface Day { date: string; estimated_minutes: number; periods: [number, number][] }
const shell = useShell()
const operation = useOperation('估算未完成，请重试。')
const { busy, error } = operation
const actions = ref<string[]>([]), action = ref(''), days = ref<Day[]>([]), note = ref('')
const reader = useReader(operation, async stale => {
  if (!shell.selectedRole.value) return
  const result = await shell.invoke<{ actions: string[]; days: Day[]; note?: string }>('billing.playtime', { ...shell.chosen(), recovery_action: action.value })
  if (stale()) return
  actions.value = result.actions; days.value = result.days; note.value = result.note ?? ''
  if (action.value && !actions.value.includes(action.value)) action.value = ''
})
watch(shell.selection, () => { actions.value = []; action.value = ''; days.value = []; note.value = ''; reader.request() })
if (shell.selectedRole.value) reader.request()
function time(minutes: number): string { return `${String(Math.floor(minutes / 60)).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}` }
function periods(day: Day): string { return day.periods.map(period => `${time(period[0])}–${time(period[1])}`).join('、') }
</script>

<template>
  <section class="section" aria-labelledby="playtime-title">
    <div class="section-heading"><div class="titles"><h2 id="playtime-title">在线时长</h2><p>先在“资产记录”分区收集“开拓力 · 全部变动”的官方记录；选择角色后自动列出记录分类，选择自然恢复的分类即估算。此功能根据恢复节奏推算，无法证明实际在线或离线。</p></div><button type="button" :disabled="busy || !shell.selectedRole.value" @click="reader.request()">{{ reader.loading.value ? '正在读取…' : '刷新' }}</button></div>
    <form v-if="actions.length" class="field-row" @submit.prevent="reader.request()">
      <fieldset :disabled="busy && !reader.loading.value">
        <label class="field"><span>自然恢复分类</span><select v-model="action" @change="reader.request(250)"><option value="">选择自然恢复分类</option><option v-for="name in actions" :key="name" :value="name">{{ name }}</option></select></label>
        <button type="submit" :disabled="!action">重新估算</button>
      </fieldset>
    </form>
    <StatusLine v-if="reader.loading.value" kind="info" :text="action ? '正在估算在线时长…' : '正在读取开拓力记录分类…'" />
    <StatusLine kind="error" :text="error"><template v-if="reader.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="reader.request()">重试</button></template></StatusLine>
    <StatusLine kind="info" :text="note" />
    <EmptyState v-if="shell.selectedRole.value && !actions.length && !note && !error && !reader.loading.value" icon="clock" title="没有可用的开拓力记录分类" text="先在“资产记录”分区收集“开拓力 · 全部变动”的官方记录，再回到这里。" />
    <ul v-if="days.length" class="rows">
      <li v-for="day in days" :key="day.date">
        <div class="row-main"><strong>{{ day.date }} · 约 {{ (day.estimated_minutes / 60).toFixed(1) }} 小时</strong><div class="row-meta"><span>{{ periods(day) }}</span></div></div>
      </li>
    </ul>
  </section>
</template>
