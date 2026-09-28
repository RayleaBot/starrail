<script lang="ts">
// The list item shape and status labels are shared with the merged task list.
export interface MonthlyTask { ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string }; hour: number; minute?: number; enabled: boolean; last_code: string; next_check_ms: number; expires_at_ms: number }
export const monthlyLabels: Record<string, string> = { collected: '月报已保存' }
</script>
<script setup lang="ts">
import { ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'
import { useReader } from './reader'

// 月报收集: the official default month saved daily for the selected role.
const emit = defineEmits<{ changed: [] }>()
const shell = useShell()
const operation = useOperation('月报收集操作未完成，请重试。')
const { busy, error, notice, run } = operation
const items = ref<MonthlyTask[]>([]), hour = ref(23), minute = ref(50), days = ref(30), notify = ref(false), confirmed = ref(false), removal = ref('')
watch([shell.selection, hour, minute, days, notify], () => { confirmed.value = false })
async function load() { items.value = (await shell.invoke<{ items: MonthlyTask[] }>('monthly.task.list')).items }
async function create() {
  if (!confirmed.value) return
  await run(async () => {
    await shell.invoke('monthly.task.create', { ...shell.chosen(), hour: Number(hour.value), minute: Number(minute.value), days: Number(days.value), notify: notify.value, confirm: true })
    confirmed.value = false; await load(); emit('changed')
  }, '每日月报收集已开启。')
}
async function remove(ref: string) { await run(async () => { await shell.invoke('monthly.task.remove', { ref }); removal.value = ''; await load(); emit('changed') }, '月报收集已停止。') }
function clock(item: MonthlyTask): string { return `${String(item.hour).padStart(2, '0')}:${String(item.minute ?? 0).padStart(2, '0')}` }
const lister = useReader(operation, load)
lister.request()
</script>

<template>
  <section class="section" aria-labelledby="monthly-title">
    <div class="section-heading"><div class="titles"><h2 id="monthly-title">月报收集</h2><p>默认关闭。按北京时间每天保存官方默认月份，累计档案在“查询”页的月报入口查看，历史可用月份也可在那里手动补存。</p></div><button type="button" :disabled="busy" @click="lister.request()">{{ lister.loading.value ? '正在读取…' : '刷新' }}</button></div>
    <form class="field-grid" @submit.prevent="create">
      <fieldset :disabled="busy">
        <label class="field"><span>每天几时</span><input v-model="hour" type="number" min="0" max="23" required></label>
        <label class="field"><span>几分</span><input v-model="minute" type="number" min="0" max="59" required></label>
        <label class="field"><span>收集授权天数</span><input v-model="days" type="number" min="1" max="90" required></label>
        <div class="checks wide">
          <label class="check"><input v-model="notify" type="checkbox">保存成功后私聊通知账号所属用户</label>
          <label class="check"><input v-model="confirmed" type="checkbox">允许按以上设置定时读取并保存月报</label>
        </div>
        <div class="actions"><button class="primary" type="submit" :disabled="!shell.selectedRole.value || !confirmed">{{ busy ? '正在开启…' : '开启每日月报收集' }}</button></div>
      </fieldset>
    </form>
    <StatusLine v-if="lister.loading.value && !items.length" kind="info" text="正在读取…" />
    <StatusLine kind="error" :text="error"><template v-if="lister.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="lister.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <ul v-if="items.length" class="rows">
      <li v-for="item in items" :key="item.ref">
        <div class="row-main">
          <strong>{{ item.role.nickname }} · {{ item.role.uid }}</strong>
          <div class="row-meta"><span class="chip" :class="{ warning: !item.enabled }">{{ item.enabled ? '已开启' : '已暂停' }}</span><span>每日 {{ clock(item) }}</span><span>所属用户 {{ item.owner.actor_id }}</span></div>
          <div class="row-meta"><span>{{ monthlyLabels[item.last_code] || item.last_code || '等待首次执行' }}</span><span>下次 {{ formatTime(item.next_check_ms) }}</span><span>到期 {{ formatTime(item.expires_at_ms) }}</span></div>
        </div>
        <div class="row-actions"><button class="compact danger" type="button" :disabled="busy" @click="removal = item.ref">停止</button></div>
        <div v-if="removal === item.ref" class="confirm"><p>停止这条月报收集？已保存的月报保留，开启时创建的委托会一并停止。</p><div class="actions"><button class="compact" type="button" :disabled="busy" @click="removal = ''">保留</button><button class="compact destructive" type="button" :disabled="busy" @click="remove(item.ref)">确认停止</button></div></div>
      </li>
    </ul>
    <EmptyState v-else-if="!lister.loading.value" icon="calendar" title="没有月报收集任务" text="设置每天的保存时间与授权天数后为当前角色开启。" />
  </section>
</template>
