<script lang="ts">
// The list item shape, weekday names and status labels are shared with the merged task list.
export interface ChallengeTask { ref: string; challenge_kind: string; metric: string; threshold: number; hour: number; minute: number; weekday: number; role: { uid: string; nickname: string }; owner: { actor_id: string }; enabled: boolean; last_code: string; next_check_ms: number; expires_at_ms: number }
export const weekdays = ['每日', '每周一', '每周二', '每周三', '每周四', '每周五', '每周六', '每周日']
export const challengeLabels: Record<string, string> = { notified: '已发送提醒', target_met: '已达到目标', metric_unknown: '官方未提供所选指标', check_attempted: '已尝试检查', notification_failed: '发送结果未确认', expired: '委托到期', 'plugin.upstream_device_required': '需要官方设备验证', 'plugin.upstream_challenge_required': '需要官方验证', 'plugin.account_delegation_denied': '委托失效', 'plugin.account_caller_denied': '插件授权已撤销', 'plugin.game_challenge_unavailable': '官方成绩暂不可用' }
</script>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'
import { useReader } from './reader'

// 挑战提醒: a per-period target checked at a fixed Beijing time for the selected role.
interface Kind { id: string; label: string; metrics: { key: string; label: string; lower: boolean }[] }
interface Check { known: boolean; met: boolean; value: number; state?: string }
const emit = defineEmits<{ changed: [] }>()
const shell = useShell()
const operation = useOperation('挑战提醒操作未完成，请重试。')
const { busy, error, notice, run } = operation
const kinds = ref<Kind[]>([]), items = ref<ChallengeTask[]>([]), kind = ref(''), metric = ref(''), threshold = ref(1), hour = ref(20), minute = ref(0), weekday = ref(0), days = ref(30), confirmed = ref(false), checking = ref<Check | null>(null), removal = ref('')
const chosenKind = computed(() => kinds.value.find(item => item.id === kind.value)), chosenMetric = computed(() => chosenKind.value?.metrics.find(item => item.key === metric.value))
const checkText = computed(() => !checking.value ? '' : checking.value.state === 'not_started' ? '本期尚无完成记录' : !checking.value.known ? '官方未提供此指标' : `${chosenMetric.value?.label}：${checking.value.value} · ${checking.value.met ? '已达到目标' : '尚未达到目标'}`)
watch(kind, () => { metric.value = chosenKind.value?.metrics[0]?.key ?? '' })
watch([shell.selection, kind, metric, threshold, hour, minute, weekday, days], () => { confirmed.value = false; checking.value = null })
async function load() { items.value = (await shell.invoke<{ items: ChallengeTask[] }>('challenge.reminder.list')).items }
function payload() { return { ...shell.chosen(), kind: kind.value, metric: metric.value, threshold: Number(threshold.value), hour: Number(hour.value), minute: Number(minute.value), weekday: Number(weekday.value), days: Number(days.value), confirm: confirmed.value } }
async function create() { if (!confirmed.value) return; await run(async () => { await shell.invoke('challenge.reminder.create', payload()); confirmed.value = false; await load(); emit('changed') }, '挑战提醒已开启。') }
async function check() { await run(async () => { checking.value = await shell.invoke<Check>('challenge.reminder.check', payload()) }) }
async function remove(ref: string) { await run(async () => { await shell.invoke('challenge.reminder.remove', { ref }); removal.value = ''; await load(); emit('changed') }, '挑战提醒已关闭。') }
function kindLabel(id: string): string { return kinds.value.find(item => item.id === id)?.label ?? id }
function clock(item: ChallengeTask): string { return `${String(item.hour).padStart(2, '0')}:${String(item.minute).padStart(2, '0')}` }
const lister = useReader(operation, async () => {
  if (!kinds.value.length) { kinds.value = (await shell.invoke<{ kinds: Kind[] }>('challenge.schema')).kinds; kind.value = kinds.value[0]?.id ?? '' }
  await load()
})
lister.request()
</script>

<template>
  <section class="section" aria-labelledby="challenge-title">
    <div class="section-heading"><div class="titles"><h2 id="challenge-title">挑战提醒</h2><p>在指定的北京时间检查本期官方成绩，未达目标时私聊账号所属用户。默认关闭；每次授权最多 90 天。每个角色每种玩法可设置一个目标，调整前先关闭旧任务。</p></div><button type="button" :disabled="busy" @click="lister.request()">{{ lister.loading.value ? '正在读取…' : '刷新' }}</button></div>
    <form class="field-grid" @submit.prevent="create">
      <fieldset :disabled="busy">
        <label class="field"><span>提醒玩法</span><select v-model="kind"><option v-for="item in kinds" :key="item.id" :value="item.id">{{ item.label }}</option></select></label>
        <label class="field"><span>完成指标</span><select v-model="metric"><option v-for="item in chosenKind?.metrics" :key="item.key" :value="item.key">{{ item.label }} · {{ item.key }}</option></select></label>
        <label class="field"><span>目标{{ chosenMetric?.lower ? '不超过' : '至少达到' }}</span><input v-model="threshold" type="number" min="0" max="100000000000000" step="any" required></label>
        <label class="field"><span>重复周期</span><select v-model="weekday"><option v-for="(label, index) in weekdays" :key="index" :value="index">{{ label }}</option></select></label>
        <label class="field"><span>时</span><input v-model="hour" type="number" min="0" max="23" required></label>
        <label class="field"><span>分</span><input v-model="minute" type="number" min="0" max="59" required></label>
        <label class="field"><span>委托天数</span><input v-model="days" type="number" min="1" max="90" required></label>
        <label class="check wide"><input v-model="confirmed" type="checkbox">允许定时读取所选玩法并私聊通知账号所属用户</label>
        <div class="actions"><button class="primary" type="submit" :disabled="!shell.selectedRole.value || !confirmed">{{ busy ? '正在开启…' : '开启挑战提醒' }}</button><button type="button" :disabled="!shell.selectedRole.value" @click="check">立即检查挑战目标</button></div>
      </fieldset>
    </form>
    <StatusLine v-if="lister.loading.value && !items.length" kind="info" text="正在读取…" />
    <StatusLine kind="error" :text="error"><template v-if="lister.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="lister.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <StatusLine kind="info" :text="checkText" />
    <ul v-if="items.length" class="rows">
      <li v-for="item in items" :key="item.ref">
        <div class="row-main">
          <strong>{{ item.role.nickname }} · {{ item.role.uid }} · {{ kindLabel(item.challenge_kind) }}</strong>
          <div class="row-meta"><span class="chip" :class="{ warning: !item.enabled }">{{ item.enabled ? '已开启' : '已暂停' }}</span><span>{{ weekdays[item.weekday] }} {{ clock(item) }}</span><span>{{ item.metric }} 目标 {{ item.threshold }}</span><span>私聊 {{ item.owner.actor_id }}</span></div>
          <div class="row-meta"><span>{{ challengeLabels[item.last_code] || item.last_code || '等待首次检查' }}</span><span>下次 {{ formatTime(item.next_check_ms) }}</span><span>到期 {{ formatTime(item.expires_at_ms) }}</span></div>
        </div>
        <div class="row-actions"><button class="compact danger" type="button" :disabled="busy" @click="removal = item.ref">关闭</button></div>
        <div v-if="removal === item.ref" class="confirm"><p>关闭这条挑战提醒？开启时创建的委托会一并停止。</p><div class="actions"><button class="compact" type="button" :disabled="busy" @click="removal = ''">保留</button><button class="compact destructive" type="button" :disabled="busy" @click="remove(item.ref)">确认关闭</button></div></div>
      </li>
    </ul>
    <EmptyState v-else-if="!lister.loading.value" icon="trophy" title="没有已开启的挑战提醒" text="选择玩法、指标与检查时间后为当前角色开启。" />
  </section>
</template>
