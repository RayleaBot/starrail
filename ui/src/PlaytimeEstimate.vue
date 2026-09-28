<script setup lang="ts">
import { ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'

// 在线时长: play sessions inferred from the stamina recovery rhythm in the saved asset records.
interface Day { date: string; estimated_minutes: number; periods: [number, number][] }
const shell = useShell()
const { busy, error, run } = useOperation('估算未完成，请重试。')
const actions = ref<string[]>([]), action = ref(''), days = ref<Day[]>([]), note = ref(''), calculating = ref(false)
watch(shell.selection, () => { actions.value = []; action.value = ''; days.value = []; note.value = '' })
function time(minutes: number): string { return `${String(Math.floor(minutes / 60)).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}` }
function periods(day: Day): string { return day.periods.map(period => `${time(period[0])}–${time(period[1])}`).join('、') }
async function load(calculate = false) {
  calculating.value = calculate
  await run(async () => {
    const result = await shell.invoke<{ actions: string[]; days: Day[]; note?: string }>('billing.playtime', { ...shell.chosen(), recovery_action: calculate ? action.value : '' })
    actions.value = result.actions; days.value = result.days; note.value = result.note ?? ''
  })
}
</script>

<template>
  <section class="section" aria-labelledby="playtime-title">
    <div class="section-heading"><div class="titles"><h2 id="playtime-title">在线时长</h2><p>先在“资产记录”分区收集“开拓力 · 全部变动”的官方记录，再选择自然恢复的记录分类。此功能根据恢复节奏推算，无法证明实际在线或离线。</p></div></div>
    <form class="field-row" @submit.prevent="load()">
      <button class="primary" type="submit" :disabled="busy || !shell.selectedRole.value">{{ busy && !calculating ? '正在读取…' : '读取开拓力记录分类' }}</button>
    </form>
    <form v-if="actions.length" class="field-row" @submit.prevent="load(true)">
      <fieldset :disabled="busy">
        <label class="field"><span>自然恢复分类</span><select v-model="action"><option value="">选择自然恢复分类</option><option v-for="name in actions" :key="name" :value="name">{{ name }}</option></select></label>
        <button class="primary" type="submit" :disabled="!action">{{ busy && calculating ? '正在估算…' : '估算时长' }}</button>
      </fieldset>
    </form>
    <StatusLine kind="error" :text="error" />
    <StatusLine kind="info" :text="note" />
    <ul v-if="days.length" class="rows">
      <li v-for="day in days" :key="day.date">
        <div class="row-main"><strong>{{ day.date }} · 约 {{ (day.estimated_minutes / 60).toFixed(1) }} 小时</strong><div class="row-meta"><span>{{ periods(day) }}</span></div></div>
      </li>
    </ul>
  </section>
</template>
