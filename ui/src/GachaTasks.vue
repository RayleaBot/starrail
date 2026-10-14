<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { overseas, useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'

// 后台与每日同步: the plugin's own sync tasks, polled every 5 s while this
// view is open; new tasks are created for the selected overseas role.
interface Progress { pages: number; fetched: number; result?: { added: number; total: number } }
interface Task {
  ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string }
  hour: number; full: boolean; notify: boolean
  state: string; last_code: string; next_check_ms: number; expires_at_ms: number; progress: Progress
}
interface Sync {
  ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string }
  full: boolean; notify: boolean; state: string; last_code: string; started_ms: number; finished_ms?: number; progress: Progress
}
type Action = 'gacha.task.start' | 'gacha.task.create' | 'gacha.task.run' | 'gacha.task.remove' | 'gacha.task.cancel'
const shell = useShell()
const { busy, error, notice, run } = useOperation('任务操作未完成。')
const kind = ref<'once' | 'daily'>('once'), days = ref(30), hour = ref(8), full = ref(false), notify = ref(false)
const tasks = ref<Task[]>([]), syncs = ref<Sync[]>([]), loading = ref(false), loaded = ref(false), loadError = ref('')
let disposed = false, revision = 0
const available = computed(() => !!shell.selectedRole.value && overseas(shell.selectedRole.value.role.region))
const states: Record<string, string> = { creating: '创建中', waiting: '等待调度', running: '同步中', completed: '已完成', paused: '已暂停', expired: '已到期', failed: '未完成', canceled: '已取消' }
const chips: Record<string, string> = { completed: 'success', failed: 'danger', expired: 'danger', paused: 'warning', canceled: 'warning' }
const reasons: Record<string, string> = {
  queued: '已排队', sync_running: '正在读取官方记录', sync_completed: '本轮同步完成', sync_canceled: '已取消，原档案保持不变', sync_timeout: '后台期限内未读完，原档案保持不变', expired: '授权已到期，请移除后重新创建', archive_removed: '档案已移除，同步已停止',
  'sync_completed.notification_failed': '记录已保存，结果通知未能发送', 'plugin.game_sync_conflict': '档案发生变化，请确认后重新运行', 'plugin.game_sync_invalid': '官方分页或记录异常，原档案保留',
  'plugin.account_delegation_denied': '委托已撤销或失效，请重新创建', 'plugin.account_caller_denied': '本游戏的账号授权已撤销', 'plugin.account_not_found': '账号已移除', 'plugin.account_role_denied': '角色授权已变更',
  'plugin.upstream_auth_invalid': '账号登录已失效', 'plugin.upstream_device_required': '请先在官方应用完成设备验证', 'plugin.upstream_challenge_required': '请先在官方应用完成验证',
  'plugin.vault_locked': '账号库已锁定，稍后重试', 'plugin.service_unavailable': '账号插件不可用，稍后重试', 'plugin.game_sync_busy': '其他同步正在占用名额，稍后重试', 'plugin.account_delegation_throttled': '分页请求过于频繁，稍后重试',
}
function date(ms: number) { return ms ? formatTime(ms) : '等待下一次调度' }
async function load() {
  const request = ++revision; loading.value = true
  try {
    const result = await shell.invoke<{ items: Task[]; syncs: Sync[] }>('gacha.task.list')
    if (!disposed && request === revision) { tasks.value = result.items; syncs.value = result.syncs; loadError.value = '' }
  } catch (cause) { if (!disposed && request === revision) loadError.value = cause instanceof Error ? cause.message : '后台任务读取失败。' }
  finally { if (!disposed && request === revision) { loading.value = false; loaded.value = true } }
}
function payload(action: Action, ref?: string): Record<string, unknown> | undefined {
  if (action === 'gacha.task.run') return { ref, confirm: true }
  if (ref) return { ref }
  if (!available.value) return undefined
  const chosen = { ...shell.chosen(), full: full.value, notify: notify.value, confirm: true }
  return action === 'gacha.task.create' ? { ...chosen, days: Number(days.value), hour: Number(hour.value) } : chosen
}
async function change(action: Action, ref?: string) {
  const input = payload(action, ref)
  if (!input) return
  await run(async () => {
    revision++; loading.value = false
    const result = await shell.invoke<{ delegation_revoked?: boolean }>(action, input)
    if (disposed) return
    notice.value = {
      'gacha.task.start': '后台同步已开始，关闭页面仍会继续。', 'gacha.task.create': '每日同步已开启，关闭页面仍会继续。', 'gacha.task.run': '任务已排队重新运行。', 'gacha.task.cancel': '后台同步已取消，原档案保持不变。',
      'gacha.task.remove': result.delegation_revoked ? '每日同步与委托已停止。' : '每日同步已停止；账号服务暂未确认撤销，原委托等待到期。',
    }[action]
    await load()
  })
}
void load()
const timer = setInterval(() => { if (!busy.value && !loading.value) void load() }, 5000)
// Closing this view only stops polling; syncs and daily tasks run on in the plugin.
onUnmounted(() => { disposed = true; revision++; clearInterval(timer) })
</script>

<template>
  <section class="subsection" aria-labelledby="gacha-tasks-title">
    <div class="section-heading"><div class="titles"><h3 id="gacha-tasks-title">后台与每日同步</h3><p>关闭页面后仍会继续，每秒最多读取一页；全部卡池读取成功后合并。单次同步立即开始；每日同步在北京时间设定时刻后开始，每天一轮。插件重启会中止进行中的一轮，已有档案保留，每日同步由下一次调度重新读取。删除档案会取消对应的同步并暂停每日同步。</p></div><button type="button" :disabled="busy || loading" @click="load">刷新任务</button></div>
    <form class="field-grid" @submit.prevent="change(kind === 'daily' ? 'gacha.task.create' : 'gacha.task.start')">
        <fieldset :disabled="busy">
          <label class="field"><span>同步安排</span><select v-model="kind"><option value="once">单次后台同步</option><option value="daily">每日同步</option></select></label>
          <label v-if="kind === 'daily'" class="field"><span>授权天数</span><input v-model.number="days" type="number" min="1" max="90" step="1" required></label>
          <label v-if="kind === 'daily'" class="field"><span>北京时间（小时）</span><input v-model.number="hour" type="number" min="0" max="23" step="1" required></label>
          <div class="checks wide">
            <label class="check"><input v-model="full" type="checkbox">{{ kind === 'daily' ? '每轮全量读取官方仍保留的历史' : '全量读取官方仍保留的历史' }}（默认只读新增记录）</label>
            <label class="check"><input v-model="notify" type="checkbox">{{ kind === 'daily' ? '每轮完成后' : '完成后' }}私聊账号所属用户，通知结果</label>
          </div>
          <div class="actions">
            <button class="primary" type="submit" :disabled="!available">{{ busy ? '正在提交…' : kind === 'daily' ? '开启每日同步' : '开始单次后台同步' }}</button>
            <span v-if="!available" class="hint">{{ shell.selectedRole.value ? '当前角色属于国服，后台同步只对已接入 HoYoLab 的海外角色开放。' : '请先在上方选择已授权的海外角色。' }}</span>
          </div>
        </fieldset>
    </form>
    <StatusLine v-if="loading && !loaded" kind="info" text="正在读取后台任务…" />
    <StatusLine kind="error" :text="error || loadError"><template v-if="loadError" #actions><button class="compact" type="button" :disabled="busy || loading" @click="load">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <ul v-if="tasks.length || syncs.length" class="rows">
      <li v-for="task in tasks" :key="task.ref">
        <div class="row-main">
          <strong>{{ task.role.nickname }} · {{ task.role.uid }} <span class="chip" :class="chips[task.state]">{{ states[task.state] || '状态待确认' }}</span></strong>
          <div class="row-meta"><span>每日同步 · 北京时间 {{ task.hour }}:00</span><span>{{ task.full ? '全量' : '增量' }}</span><span>{{ task.notify ? `通知 ${task.owner.actor_id}` : '不发送结果通知' }}</span></div>
          <div class="row-meta"><span>本轮已读取 {{ task.progress.pages || 0 }} 页、{{ task.progress.fetched || 0 }} 条</span><span v-if="task.progress.result">最近完成：新增 {{ task.progress.result.added }} 条，共 {{ task.progress.result.total }} 条</span></div>
          <div class="row-meta"><span>{{ reasons[task.last_code] || (task.last_code ? '本次请求未完成，可查看插件诊断后重试。' : '等待首次运行') }}</span><span>授权到期 {{ date(task.expires_at_ms) }}（北京时间）</span><span v-if="['waiting', 'running'].includes(task.state)">下次可检查 {{ date(task.next_check_ms) }}</span></div>
        </div>
        <div class="row-actions"><button class="compact" type="button" :disabled="busy || ['creating', 'running', 'expired'].includes(task.state) || task.expires_at_ms <= Date.now()" @click="change('gacha.task.run', task.ref)">重新运行此同步</button><button class="compact danger" type="button" :disabled="busy" @click="change('gacha.task.remove', task.ref)">停止此同步任务</button></div>
      </li>
      <li v-for="sync in syncs" :key="sync.ref">
        <div class="row-main">
          <strong>{{ sync.role.nickname }} · {{ sync.role.uid }} <span class="chip" :class="chips[sync.state]">{{ states[sync.state] || '状态待确认' }}</span></strong>
          <div class="row-meta"><span>单次后台同步</span><span>{{ sync.full ? '全量' : '增量' }}</span><span>{{ sync.notify ? `通知 ${sync.owner.actor_id}` : '不发送结果通知' }}</span></div>
          <div class="row-meta"><span>已读取 {{ sync.progress.pages || 0 }} 页、{{ sync.progress.fetched || 0 }} 条</span><span v-if="sync.progress.result">新增 {{ sync.progress.result.added }} 条，共 {{ sync.progress.result.total }} 条</span></div>
          <div class="row-meta"><span>{{ reasons[sync.last_code] || '本次同步未完成，可查看插件诊断后重试。' }}</span><span>开始于 {{ date(sync.started_ms) }}<template v-if="sync.finished_ms">，结束于 {{ date(sync.finished_ms) }}</template>（北京时间）</span></div>
        </div>
        <div v-if="sync.state === 'running'" class="row-actions"><button class="compact danger" type="button" :disabled="busy" @click="change('gacha.task.cancel', sync.ref)">取消此同步</button></div>
      </li>
    </ul>
    <EmptyState v-else-if="loaded" icon="tasks" title="尚无后台同步任务" text="在上方创建单次或每日同步；每日同步可重新运行，或停止后调整设置。" />
  </section>
</template>
