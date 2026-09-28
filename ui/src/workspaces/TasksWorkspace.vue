<script setup lang="ts">
import { computed, ref } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import StatusLine from '../shell/StatusLine.vue'
import EmptyState from '../shell/EmptyState.vue'
import { useShell } from '../shell/context'
import { formatTime, useOperation } from '../shell/status'
import { useReader } from '../reader'
import Reminders, { reminderLabels, type ReminderTask } from '../Reminders.vue'
import ChallengeReminders, { challengeLabels, weekdays, type ChallengeTask } from '../ChallengeReminders.vue'
import MonthlyCollection, { monthlyLabels, type MonthlyTask } from '../MonthlyCollection.vue'
import AccountTasks, { communityLabels, type CommunityTask } from '../AccountTasks.vue'

const shell = useShell()
const sections: Section[] = [
  { id: 'all', label: '全部任务', icon: 'tasks' },
  { id: 'reminder', label: '体力提醒', icon: 'bell' },
  { id: 'challenge', label: '挑战提醒', icon: 'trophy' },
  { id: 'monthly', label: '月报收集', icon: 'calendar' },
  { id: 'signin', label: '每日签到', icon: 'checkin' },
  { id: 'community', label: '米游社任务', icon: 'chat' },
]

// 全部任务: every kind's list read through one adapter each, so the rows share
// a shape. The plugin's field names are kept when reading; only the wording is common.
type Kind = 'reminder' | 'challenge' | 'monthly' | 'signin' | 'community'
interface SigninTask { ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string }; enabled: boolean; hour?: number; expires_at_ms: number; next_check_ms: number; last_code: string; notify?: boolean }
interface TaskRow { key: string; kind: Kind; ref: string; remove: string; subject: string; schedule: string; owner: string; status: string; tone: string; next_check_ms: number; expires_at_ms: number }
const kindLabels: Record<Kind, string> = { reminder: '体力提醒', challenge: '挑战提醒', monthly: '月报收集', signin: '每日签到', community: '米游社任务' }
const signinLabels: Record<string, string> = { signed: '签到成功', already_signed: '今日已签到', sign_attempted: '已尝试，结果待确认', expired: '委托已到期', 'plugin.upstream_device_required': '需要官方设备验证', 'plugin.upstream_challenge_required': '需要官方验证', 'plugin.upstream_first_sign_required': '需要在官方应用首次签到', 'plugin.account_delegation_denied': '委托失效', 'plugin.account_caller_denied': '游戏授权已撤销', 'plugin.vault_locked': '账号库锁定，稍后检查' }
const list = useOperation('任务列表读取未完成，请重试。')
const reminders = ref<ReminderTask[]>([]), challenges = ref<ChallengeTask[]>([]), monthlies = ref<MonthlyTask[]>([]), signins = ref<SigninTask[]>([]), communities = ref<CommunityTask[]>([]), challengeKinds = ref<{ id: string; label: string }[]>([]), removal = ref('')
const loaders: Record<Kind, () => Promise<void>> = {
  reminder: async () => { reminders.value = (await shell.invoke<{ items: ReminderTask[] }>('reminder.list')).items },
  challenge: async () => { challenges.value = (await shell.invoke<{ items: ChallengeTask[] }>('challenge.reminder.list')).items },
  monthly: async () => { monthlies.value = (await shell.invoke<{ items: MonthlyTask[] }>('monthly.task.list')).items },
  signin: async () => { signins.value = (await shell.invoke<{ items: SigninTask[] }>('signin.task.list')).items },
  community: async () => { communities.value = (await shell.invoke<{ items: CommunityTask[] }>('community.task.list')).items },
}
async function loadChallengeKinds() { challengeKinds.value = (await shell.invoke<{ kinds: { id: string; label: string }[] }>('challenge.schema')).kinds }
const lister = useReader(list, async () => { await Promise.all([...Object.values(loaders).map(load => load()), loadChallengeKinds()]) })
function loadAll() { lister.request() }
function refresh(kind: Kind) { void list.run(loaders[kind]) }
const finished = new Set(['checked', 'notified', 'target_met', 'collected', 'signed', 'already_signed', 'step_completed', 'completed', 'cloud_completed'])
function tone(task: { enabled: boolean; last_code: string }): string {
  if (!task.enabled) return 'warning'
  const code = task.last_code
  if (code.startsWith('plugin.') || code === 'expired' || code === 'notification_failed' || code === 'step_outcome_unknown') return 'danger'
  if (code === 'completed_with_remaining' || code === 'metric_unknown') return 'warning'
  return finished.has(code) ? 'success' : ''
}
function clock(hour: number, minute: number): string { return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}` }
function row(kind: Kind, task: { ref: string; enabled: boolean; last_code: string; next_check_ms: number; expires_at_ms: number }, labels: Record<string, string>, waiting: string, text: { remove: string; subject: string; schedule: string; owner: string }): TaskRow {
  return { key: `${kind}:${task.ref}`, kind, ref: task.ref, ...text, status: task.enabled ? labels[task.last_code] || task.last_code || waiting : '已暂停', tone: tone(task), next_check_ms: task.next_check_ms, expires_at_ms: task.expires_at_ms }
}
const rows = computed<TaskRow[]>(() => [
  ...reminders.value.map(task => row('reminder', task, reminderLabels, '等待首次检查', { remove: 'reminder.remove', subject: `${task.role.nickname} · ${task.role.uid}`, schedule: `每十分钟检查，体力达到 ${task.threshold}% 时提醒${task.enabled ? (task.armed ? '，等待达到阈值' : '，等待体力低于阈值') : ''}`, owner: `私聊 ${task.owner.actor_id}（${task.owner.source_adapter}）` })),
  ...challenges.value.map(task => row('challenge', task, challengeLabels, '等待首次检查', { remove: 'challenge.reminder.remove', subject: `${task.role.nickname} · ${task.role.uid} · ${challengeKinds.value.find(item => item.id === task.challenge_kind)?.label ?? task.challenge_kind}`, schedule: `${weekdays[task.weekday]} ${clock(task.hour, task.minute)} 检查，${task.metric} 目标 ${task.threshold}`, owner: `私聊 ${task.owner.actor_id}` })),
  ...monthlies.value.map(task => row('monthly', task, monthlyLabels, '等待首次执行', { remove: 'monthly.task.remove', subject: `${task.role.nickname} · ${task.role.uid}`, schedule: `每日 ${clock(task.hour, task.minute ?? 0)} 保存官方默认月份`, owner: `所属用户 ${task.owner.actor_id}` })),
  ...signins.value.map(task => row('signin', task, signinLabels, '等待首次检查', { remove: 'signin.task.remove', subject: `${task.role.nickname} · ${task.role.uid}`, schedule: `每日北京时间 ${task.hour ?? 0}:00 后首次检查时签到`, owner: task.notify ? `私聊通知 ${task.owner.actor_id}` : '不发送结果通知' })),
  ...communities.value.map(task => row('community', task, communityLabels, '等待执行', { remove: `${task.kind}.task.remove`, subject: `${task.role.nickname} · 社区任务`, schedule: `${task.once ? '执行一次' : `每日 ${clock(task.hour, task.minute || 0)}（北京时间）`} · 进度 ${task.community?.cursor || 0} / ${task.community?.steps?.length || 0}`, owner: `所属用户 ${task.owner.actor_id}` })),
])
async function remove(target: TaskRow) {
  await list.run(async () => {
    const result = await shell.invoke<{ delegation_revoked?: boolean }>(target.remove, { ref: target.ref })
    removal.value = ''
    await loaders[target.kind]()
    list.notice.value = result.delegation_revoked === undefined ? '任务已移除。' : result.delegation_revoked ? '任务与账号委托已停止。' : '本地任务已停止；账号服务暂未确认撤销，原委托到期后失效，此任务不会再执行。'
  })
}

// 每日签到: the scheduled sign-in of the selected role, kept here beside the merged list.
const signinTask = useOperation('自动签到设置未完成，请重试。')
const hour = ref(8), days = ref(30), notify = ref(false), signinRemoval = ref('')
async function createSignin() {
  await signinTask.run(async () => {
    await shell.invoke('signin.task.create', { ...shell.chosen(), hour: Number(hour.value), days: Number(days.value), notify: notify.value, confirm: true })
    await loaders.signin()
  }, '每日自动签到已开启。')
}
async function removeSignin(ref: string) {
  await signinTask.run(async () => {
    const result = await shell.invoke<{ delegation_revoked?: boolean }>('signin.task.remove', { ref })
    signinRemoval.value = ''
    await loaders.signin()
    signinTask.notice.value = result.delegation_revoked ? '自动签到与委托已停止。' : '本地自动签到已停止；账号服务暂未确认撤销，此任务不会再执行。'
  })
}
loadAll()
</script>

<template>
  <Workspace title="定时任务" description="后台按时执行的提醒、收集与签到。每类任务在开启时为当前角色创建有期限的委托，到期或撤销授权后自动暂停。" :sections="sections" roles>
    <template #default="{ active }">
      <section v-if="active === 'all'" class="section" aria-labelledby="all-title">
        <div class="section-heading"><div class="titles"><h2 id="all-title">全部任务</h2><p>所有已开启的定时任务及其最近状态；新建任务请到对应分区。</p></div><button type="button" :disabled="list.busy.value" @click="loadAll">{{ lister.loading.value ? '正在刷新…' : '刷新' }}</button></div>
        <StatusLine v-if="lister.loading.value && !rows.length" kind="info" text="正在读取任务列表…" />
        <StatusLine kind="error" :text="list.error.value"><template v-if="lister.failed.value" #actions><button class="compact" type="button" :disabled="list.busy.value" @click="loadAll">重试</button></template></StatusLine>
        <StatusLine kind="success" :text="list.notice.value" />
        <ul v-if="rows.length" class="rows">
          <li v-for="item in rows" :key="item.key">
            <div class="row-main">
              <strong><span class="chip plain">{{ kindLabels[item.kind] }}</span> {{ item.subject }}</strong>
              <div class="row-meta"><span class="chip" :class="item.tone">{{ item.status }}</span><span>{{ item.schedule }}</span><span>{{ item.owner }}</span></div>
              <div class="row-meta"><span>下次检查 {{ formatTime(item.next_check_ms) }}</span><span>到期 {{ formatTime(item.expires_at_ms) }}</span></div>
            </div>
            <div class="row-actions"><button class="compact danger" type="button" :disabled="list.busy.value" @click="removal = item.key">移除</button></div>
            <div v-if="removal === item.key" class="confirm"><p>移除这条{{ kindLabels[item.kind] }}？开启时创建的委托会一并停止。</p><div class="actions"><button class="compact" type="button" :disabled="list.busy.value" @click="removal = ''">保留</button><button class="compact destructive" type="button" :disabled="list.busy.value" @click="remove(item)">确认移除</button></div></div>
          </li>
        </ul>
        <EmptyState v-else-if="!list.busy.value" icon="tasks" title="还没有定时任务" text="在左侧分区为当前角色开启体力提醒、挑战提醒、月报收集、每日签到或米游社任务。" />
      </section>

      <Reminders v-else-if="active === 'reminder'" @changed="refresh('reminder')" />
      <ChallengeReminders v-else-if="active === 'challenge'" @changed="refresh('challenge')" />
      <MonthlyCollection v-else-if="active === 'monthly'" @changed="refresh('monthly')" />

      <section v-else-if="active === 'signin'" class="section" aria-labelledby="signin-task-title">
        <div class="section-heading"><div class="titles"><h2 id="signin-task-title">每日签到</h2><p>显式开启后，在期限内每天北京时间所选时刻后的首次检查执行，每十分钟检查一次。默认不发送结果通知。未确认的提交不在当天自动重试，可在“账号操作”页手动查询签到状态。</p></div><button type="button" :disabled="signinTask.busy.value" @click="signinTask.run(loaders.signin)">刷新</button></div>
        <form class="field-grid" @submit.prevent="createSignin">
          <fieldset :disabled="signinTask.busy.value || !shell.selectedRole.value">
            <label class="field"><span>北京时间（小时）</span><input v-model.number="hour" type="number" min="0" max="23" step="1" required></label>
            <label class="field"><span>授权天数</span><input v-model.number="days" type="number" min="1" max="90" step="1" required></label>
            <label class="check wide"><input v-model="notify" type="checkbox">私聊账号所属用户，通知每日结果</label>
            <div class="actions"><button class="primary" type="submit">{{ signinTask.busy.value ? '正在开启…' : '开启每日自动签到' }}</button></div>
          </fieldset>
        </form>
        <StatusLine kind="error" :text="signinTask.error.value" />
        <StatusLine kind="success" :text="signinTask.notice.value" />
        <StatusLine v-if="lister.loading.value && !signins.length" kind="info" text="正在读取…" />
        <ul v-if="signins.length" class="rows">
          <li v-for="task in signins" :key="task.ref">
            <div class="row-main">
              <strong>{{ task.role.nickname }} · {{ task.role.uid }}</strong>
              <div class="row-meta"><span class="chip" :class="{ warning: !task.enabled }">{{ task.enabled ? '已开启' : '已暂停' }}</span><span>每日北京时间 {{ task.hour ?? 0 }}:00</span><span>{{ task.notify ? '私聊通知 ' + task.owner.actor_id : '不发送结果通知' }}</span></div>
              <div class="row-meta"><span>{{ signinLabels[task.last_code] || task.last_code || '等待首次检查' }}</span><span>下次可检查 {{ formatTime(task.next_check_ms) }}</span><span>到期 {{ formatTime(task.expires_at_ms) }}</span></div>
            </div>
            <div class="row-actions"><button class="compact danger" type="button" :disabled="signinTask.busy.value" @click="signinRemoval = task.ref">停止</button></div>
            <div v-if="signinRemoval === task.ref" class="confirm"><p>停止这条自动签到？开启时创建的委托会一并停止。</p><div class="actions"><button class="compact" type="button" :disabled="signinTask.busy.value" @click="signinRemoval = ''">保留</button><button class="compact destructive" type="button" :disabled="signinTask.busy.value" @click="removeSignin(task.ref)">确认停止</button></div></div>
          </li>
        </ul>
        <EmptyState v-else-if="!signinTask.busy.value && !list.busy.value" icon="checkin" title="尚未开启自动签到" text="设置北京时间小时与授权天数后为当前角色开启。" />
      </section>

      <AccountTasks v-else-if="active === 'community'" @changed="refresh('community')" />
    </template>
  </Workspace>
</template>
