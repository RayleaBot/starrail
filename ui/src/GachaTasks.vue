<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
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
const props = defineProps<{ choices: { key: string; label: string; account: { ref: string }; role: { ref: string } }[]; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const selected = ref(''), kind = ref<'once' | 'daily'>('once'), days = ref(30), hour = ref(8), full = ref(false), notify = ref(false)
const tasks = ref<Task[]>([]), syncs = ref<Sync[]>([]), busy = ref(false), loading = ref(false), error = ref(''), loadError = ref(''), notice = ref('')
let disposed = false, revision = 0
const states: Record<string, string> = { creating: '创建中', waiting: '等待调度', running: '同步中', completed: '已完成', paused: '已暂停', expired: '已到期', failed: '未完成', canceled: '已取消' }
const reasons: Record<string, string> = {
  queued: '已排队', sync_running: '正在读取官方记录', sync_completed: '本轮同步完成', sync_canceled: '已取消，原档案保持不变', sync_timeout: '后台期限内未读完，原档案保持不变', expired: '授权已到期，请移除后重新创建', archive_removed: '档案已移除，同步已停止',
  'sync_completed.notification_failed': '记录已保存，结果通知未能发送', 'plugin.game_sync_conflict': '档案发生变化，请确认后重新运行', 'plugin.game_sync_invalid': '官方分页或记录异常，原档案保留',
  'plugin.account_delegation_denied': '委托已撤销或失效，请重新创建', 'plugin.account_caller_denied': '本游戏的账号授权已撤销', 'plugin.account_not_found': '账号已移除', 'plugin.account_role_denied': '角色授权已变更',
  'plugin.upstream_auth_invalid': '账号登录已失效', 'plugin.upstream_device_required': '请先在官方应用完成设备验证', 'plugin.upstream_challenge_required': '请先在官方应用完成验证',
  'plugin.vault_locked': '账号库已锁定，稍后重试', 'plugin.service_unavailable': '账号插件不可用，稍后重试', 'plugin.game_sync_busy': '其他同步正在占用名额，稍后重试', 'plugin.account_delegation_throttled': '分页请求过于频繁，稍后重试',
}
function date(ms: number) { return ms ? new Date(ms).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' }) : '等待下一次调度' }
async function load() {
  const request = ++revision; loading.value = true
  try {
    const result = await props.invoke<{ items: Task[]; syncs: Sync[] }>('gacha.task.list')
    if (!disposed && request === revision) { tasks.value = result.items; syncs.value = result.syncs; loadError.value = '' }
  } catch (cause) { if (!disposed && request === revision) loadError.value = cause instanceof Error ? cause.message : '后台任务读取失败。' }
  finally { if (!disposed && request === revision) loading.value = false }
}
function payload(action: Action, ref?: string): Record<string, unknown> | undefined {
  if (action === 'gacha.task.run') return { ref, confirm: true }
  if (ref) return { ref }
  const choice = props.choices.find(c => c.key === selected.value)
  if (!choice) return undefined
  const chosen = { account_ref: choice.account.ref, role_ref: choice.role.ref, full: full.value, notify: notify.value, confirm: true }
  return action === 'gacha.task.create' ? { ...chosen, days: Number(days.value), hour: Number(hour.value) } : chosen
}
async function change(action: Action, ref?: string) {
  const input = payload(action, ref)
  if (busy.value || !input) return
  busy.value = true; revision++; loading.value = false; error.value = ''; notice.value = ''
  try {
    const result = await props.invoke<{ delegation_revoked?: boolean }>(action, input)
    if (disposed) return
    notice.value = {
      'gacha.task.start': '后台同步已开始，关闭页面仍会继续。', 'gacha.task.create': '每日同步已开启，关闭页面仍会继续。', 'gacha.task.run': '任务已排队重新运行。', 'gacha.task.cancel': '后台同步已取消，原档案保持不变。',
      'gacha.task.remove': result.delegation_revoked ? '每日同步与委托已停止。' : '每日同步已停止；账号服务暂未确认撤销，原委托等待到期。',
    }[action]
    await load()
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '任务操作未完成。' }
  finally { if (!disposed) busy.value = false }
}
void load()
const timer = setInterval(() => { if (!busy.value && !loading.value) void load() }, 5000)
// Closing this view only stops polling; syncs and daily tasks run on in the plugin.
onUnmounted(() => { disposed = true; revision++; clearInterval(timer) })
</script>
<template>
  <section class="gacha-tasks separated">
    <div class="section-heading"><h3>后台与每日同步</h3><button :disabled="busy || loading" @click="load">刷新任务</button></div>
    <p class="hint">关闭页面后仍会继续，每秒最多读取一页；全部卡池读取成功后合并。单次同步立即开始；每日同步在北京时间设定时刻后开始，每天一轮。插件重启会中止进行中的一轮，已有档案保留，每日同步由下一次调度重新读取。删除档案会取消对应的同步并暂停每日同步。</p>
    <details><summary>创建后台同步任务</summary>
      <form @submit.prevent="change(kind === 'daily' ? 'gacha.task.create' : 'gacha.task.start')"><fieldset :disabled="busy"><legend class="sr-only">后台同步设置</legend>
        <label>后台同步角色<select v-model="selected" required><option value="" disabled>选择已授权角色</option><option v-for="c in choices" :key="c.key" :value="c.key">{{ c.label }}</option></select></label>
        <div class="task-fields"><label>同步安排<select v-model="kind"><option value="once">单次后台同步</option><option value="daily">每日同步</option></select></label><label v-if="kind === 'daily'">授权天数<input v-model.number="days" type="number" min="1" max="90" step="1" required></label><label v-if="kind === 'daily'">北京时间（小时）<input v-model.number="hour" type="number" min="0" max="23" step="1" required></label></div>
        <label class="check"><input v-model="full" type="checkbox">{{ kind === 'daily' ? '每轮全量读取官方仍保留的历史' : '全量读取官方仍保留的历史' }}（默认只读新增记录）</label>
        <label class="check"><input v-model="notify" type="checkbox">{{ kind === 'daily' ? '每轮完成后' : '完成后' }}私聊账号所属用户，通知结果</label>
        <button class="primary" type="submit" :disabled="!selected">{{ kind === 'daily' ? '开启每日同步' : '开始单次后台同步' }}</button>
      </fieldset></form>
    </details>
    <p v-if="error || loadError" role="alert" class="feedback danger">{{ error || loadError }}</p><p v-if="notice" role="status" class="feedback">{{ notice }}</p>
    <p v-if="!tasks.length && !syncs.length" class="hint">尚无后台同步任务。每日同步可重新运行，或停止后调整设置。</p>
    <ul class="task-list"><li v-for="task in tasks" :key="task.ref">
      <div class="task-heading"><strong>{{ task.role.nickname }} · {{ task.role.uid }}</strong><span>{{ states[task.state] || '状态待确认' }}</span></div>
      <p>每天北京时间 {{ task.hour }}:00 · {{ task.full ? '全量' : '增量' }} · {{ task.notify ? `通知 ${task.owner.actor_id}` : '不发送结果通知' }}</p>
      <p>本轮已读取 {{ task.progress.pages || 0 }} 页、{{ task.progress.fetched || 0 }} 条。<span v-if="task.progress.result">最近完成：新增 {{ task.progress.result.added }} 条，共 {{ task.progress.result.total }} 条。</span></p>
      <p class="hint">{{ reasons[task.last_code] || (task.last_code ? '本次请求未完成，可查看插件诊断后重试。' : '等待首次运行') }}<br>授权到期：{{ date(task.expires_at_ms) }}（北京时间）<template v-if="['waiting', 'running'].includes(task.state)"><br>下次可检查：{{ date(task.next_check_ms) }}</template></p>
      <div class="actions"><button :disabled="busy || ['creating', 'running', 'expired'].includes(task.state) || task.expires_at_ms <= Date.now()" @click="change('gacha.task.run', task.ref)">重新运行此同步</button><button :disabled="busy" @click="change('gacha.task.remove', task.ref)">停止此同步任务</button></div>
    </li><li v-for="sync in syncs" :key="sync.ref">
      <div class="task-heading"><strong>{{ sync.role.nickname }} · {{ sync.role.uid }}</strong><span>{{ states[sync.state] || '状态待确认' }}</span></div>
      <p>单次后台同步 · {{ sync.full ? '全量' : '增量' }} · {{ sync.notify ? `通知 ${sync.owner.actor_id}` : '不发送结果通知' }}</p>
      <p>已读取 {{ sync.progress.pages || 0 }} 页、{{ sync.progress.fetched || 0 }} 条。<span v-if="sync.progress.result">新增 {{ sync.progress.result.added }} 条，共 {{ sync.progress.result.total }} 条。</span></p>
      <p class="hint">{{ reasons[sync.last_code] || '本次同步未完成，可查看插件诊断后重试。' }}<br>开始于 {{ date(sync.started_ms) }}<template v-if="sync.finished_ms">，结束于 {{ date(sync.finished_ms) }}</template>（北京时间）</p>
      <div v-if="sync.state === 'running'" class="actions"><button :disabled="busy" @click="change('gacha.task.cancel', sync.ref)">取消此同步</button></div>
    </li></ul>
  </section>
</template>
<style scoped>
.gacha-tasks{display:grid;gap:20px}.gacha-tasks .section-heading{margin-bottom:0}.gacha-tasks fieldset{display:grid;gap:16px;padding:16px 0 0;border:0;min-width:0}.gacha-tasks label:not(.check){display:grid;gap:8px;min-width:0}.task-fields{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.task-list{list-style:none;padding:0;margin:0}.task-list li{display:grid;gap:12px;padding:20px 0;border-bottom:1px solid var(--raylea-color-border)}.task-heading{display:flex;justify-content:space-between;flex-wrap:wrap;gap:8px}.task-heading span{font-weight:600}@media(max-width:720px){.task-fields{grid-template-columns:1fr}}
</style>
