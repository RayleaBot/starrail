<script lang="ts">
// The list item shape and status labels are shared with the merged task list.
export interface ReminderTask { ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string; source_protocol: string; bot_id: string }; threshold: number; enabled: boolean; armed: boolean; expires_at_ms: number; last_checked_ms: number; last_attempt_ms: number; next_check_ms: number; last_code: string }
export const reminderLabels: Record<string, string> = { checked: '已检查', notified: '已发送提醒', notification_attempted: '已尝试发送', notification_failed: '发送未确认', expired: '委托到期', 'plugin.account_caller_denied': '游戏授权已撤销', 'plugin.account_delegation_denied': '委托失效', 'plugin.account_not_found': '账号已移除', 'plugin.account_role_denied': '角色授权失效', 'plugin.upstream_auth_invalid': '账号需要重新登录', 'plugin.upstream_device_required': '需要官方设备验证，已退避', 'plugin.upstream_challenge_required': '需要官方验证，已退避', 'plugin.vault_locked': '账号库锁定', 'plugin.game_note_invalid': '体力数据暂不兼容' }
</script>
<script setup lang="ts">
import { ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { botLabel, useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'
import { useReader } from './reader'

// 体力提醒: a stamina threshold checked every ten minutes for the selected role.
const emit = defineEmits<{ changed: [] }>()
const shell = useShell()
const operation = useOperation('提醒操作未完成，请重试。')
const { busy, error, notice, run } = operation
const items = ref<ReminderTask[]>([]), threshold = ref(80), days = ref(30), confirmed = ref(false), removal = ref('')
watch(shell.selection, () => { confirmed.value = false })
async function load() { items.value = (await shell.invoke<{ items: ReminderTask[] }>('reminder.list')).items }
async function create() {
  if (!confirmed.value) return
  await run(async () => {
    await shell.invoke('reminder.create', { ...shell.chosen(), threshold: Number(threshold.value), days: Number(days.value) })
    await load(); confirmed.value = false; emit('changed')
  }, '提醒已开启，下一次检查将在十分钟内执行。')
}
async function remove(ref: string) {
  await run(async () => {
    const result = await shell.invoke<{ delegation_revoked: boolean }>('reminder.remove', { ref })
    removal.value = ''; await load(); emit('changed')
    notice.value = result.delegation_revoked ? '提醒与账号委托已关闭。' : '本地提醒已关闭；账号服务暂未确认撤销，原委托到期后失效，此任务不会再使用它。'
  })
}
function status(item: ReminderTask): string { return item.enabled ? (item.armed ? '等待达到阈值' : '等待体力低于阈值') : '已暂停' }
const lister = useReader(operation, load)
lister.request()
</script>

<template>
  <section class="section" aria-labelledby="reminder-title">
    <div class="section-heading"><div class="titles"><h2 id="reminder-title">体力提醒</h2><p>每十分钟读取一次当前角色的体力，达到阈值后私聊账号所属用户。持续高于阈值只通知一次，低于阈值后重新待发。默认关闭，开启时创建有期限的只读委托。</p></div><button type="button" :disabled="busy" @click="lister.request()">{{ lister.loading.value ? '正在读取…' : '刷新' }}</button></div>
    <form class="field-grid" @submit.prevent="create">
      <fieldset :disabled="busy">
        <label class="field"><span>体力阈值（%）</span><input v-model="threshold" type="number" min="1" max="100" step="1" required></label>
        <label class="field"><span>授权天数</span><input v-model="days" type="number" min="1" max="90" step="1" required></label>
        <label class="check wide"><input v-model="confirmed" type="checkbox">允许定时读取当前角色体力并私聊通知账号所属用户</label>
        <div class="actions"><button class="primary" type="submit" :disabled="!shell.selectedRole.value || !confirmed">{{ busy ? '正在开启…' : '开启提醒' }}</button></div>
      </fieldset>
    </form>
    <StatusLine v-if="lister.loading.value && !items.length" kind="info" text="正在读取…" />
    <StatusLine kind="error" :text="error"><template v-if="lister.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="lister.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <ul v-if="items.length" class="rows">
      <li v-for="item in items" :key="item.ref">
        <div class="row-main">
          <strong>{{ item.role.nickname }} · {{ item.role.uid }}</strong>
          <div class="row-meta"><span class="chip" :class="{ warning: !item.enabled }">{{ status(item) }}</span><span>阈值 {{ item.threshold }}%</span><span>私聊 {{ item.owner.actor_id }} · 机器人 {{ botLabel(shell.bots.value, item.owner.source_protocol, item.owner.bot_id) }}</span></div>
          <div class="row-meta"><span>最近检查 {{ formatTime(item.last_checked_ms) }} · {{ reminderLabels[item.last_code] || item.last_code || '等待首次检查' }}</span><span>下一次可检查 {{ formatTime(item.next_check_ms) }}</span><span>到期 {{ formatTime(item.expires_at_ms) }}</span></div>
        </div>
        <div class="row-actions"><button class="compact danger" type="button" :disabled="busy" @click="removal = item.ref">关闭并移除</button></div>
        <div v-if="removal === item.ref" class="confirm"><p>关闭并移除这条体力提醒？开启时创建的账号委托会一并撤销。</p><div class="actions"><button class="compact" type="button" :disabled="busy" @click="removal = ''">保留</button><button class="compact destructive" type="button" :disabled="busy" @click="remove(item.ref)">确认移除</button></div></div>
      </li>
    </ul>
    <EmptyState v-else-if="!lister.loading.value" icon="bell" title="尚未开启体力提醒" text="填写阈值与授权天数后为当前角色开启，首次检查会在十分钟内执行。" />
    <p class="hint">发送结果不确定时不自动补发，以免重复通知。设备验证失败会暂停查询至少一天；撤销授权、凭据失效或到期后暂停，处理后可移除并重新开启。</p>
  </section>
</template>
