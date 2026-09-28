<script lang="ts">
// The list item shape and status labels are shared with the merged task list.
export interface CommunityTask { ref: string; kind: string; account_ref: string; owner: { actor_id: string }; role: { nickname: string }; enabled: boolean; once: boolean; hour: number; minute: number; last_code: string; next_check_ms: number; expires_at_ms: number; community?: { cursor: number; steps?: unknown[]; remaining?: string; state?: string } }
export const communityLabels: Record<string, string> = { step_attempted: '已记下尝试，等待结果', step_completed: '本步完成', completed: '本轮完成', completed_with_remaining: '本轮结束，仍有可获取米游币', step_outcome_unknown: '上一步结果未确认，本轮已停止', cloud_attempted: '已尝试云签到，结果待核对', cloud_completed: '云钱包查询已完成', expired: '委托已到期', 'plugin.upstream_community_unavailable': '社区授权需处理', 'plugin.account_cloud_unavailable': '尚未配置云授权', 'plugin.upstream_cloud_unavailable': '云授权需处理', 'plugin.game_community_posts_missing': '讨论区帖子不足', 'plugin.upstream_challenge_required': '需要官方验证', 'plugin.upstream_device_required': '需要设备验证' }
</script>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'
import { useReader } from './reader'

// 米游社任务 (scheduled): one-off or daily community steps for the selected role's account.
const kind = 'community'
const emit = defineEmits<{ changed: [] }>()
const shell = useShell()
const operation = useOperation('账号任务操作未完成，请重试。')
const { busy, error, notice, run } = operation
const items = ref<CommunityTask[]>([]), once = ref(true), hour = ref(8), minute = ref(0), days = ref(30), notify = ref(false), read = ref(false), like = ref(false), share = ref(false), unlike = ref(false), confirmed = ref(false), removal = ref('')
const account = computed(() => shell.selectedRole.value?.account)
watch([shell.selection, once, hour, minute, days, notify, read, like, share, unlike], () => { confirmed.value = false })
async function load() { items.value = (await shell.invoke<{ items: CommunityTask[] }>('community.task.list')).items }
async function create() {
  const current = account.value
  if (!confirmed.value || !current) return
  await run(async () => {
    await shell.invoke(`${kind}.task.create`, { account_ref: current.ref, once: once.value, hour: Number(hour.value), minute: Number(minute.value), days: once.value ? 1 : Number(days.value), notify: notify.value, read: read.value, like: like.value, share: share.value, unlike: like.value && unlike.value, confirm: true })
    confirmed.value = false; await load(); emit('changed')
  }, '任务已开启，每分钟最多处理一步。')
}
async function remove(task: CommunityTask) {
  await run(async () => {
    const result = await shell.invoke<{ delegation_revoked: boolean }>(`${task.kind}.task.remove`, { ref: task.ref })
    removal.value = ''; await load(); emit('changed')
    notice.value = result.delegation_revoked ? '任务与委托已停止。' : '本地任务已停止；账号服务暂未确认撤销，原委托将自动到期。'
  })
}
function schedule(task: CommunityTask): string { return task.once ? '执行一次' : `每日 ${String(task.hour).padStart(2, '0')}:${String(task.minute || 0).padStart(2, '0')}（北京时间）` }
const lister = useReader(operation, load)
lister.request()
</script>

<template>
  <section class="section" aria-labelledby="community-task-title">
    <div class="section-heading"><div class="titles"><h2 id="community-task-title">米游社任务</h2><p>默认关闭。以当前角色所属的米游社账号执行：先检查进度，再按选择进行签到、浏览、点赞和分享；每分钟处理一步，写入结果不确定时停止本轮。米游币进度由账号共用，可在常用游戏开启。</p></div><button type="button" :disabled="busy" @click="lister.request()">{{ lister.loading.value ? '正在读取…' : '刷新' }}</button></div>
    <p v-if="account" class="hint">任务账号：{{ account.account_label }} · 用户 {{ account.owner.actor_id }}</p>
    <form class="field-grid" @submit.prevent="create">
      <fieldset :disabled="busy">
        <label class="field"><span>执行方式</span><select v-model="once"><option :value="true">执行一次</option><option :value="false">每天执行</option></select></label>
        <template v-if="!once">
          <label class="field"><span>北京时间（时）</span><input v-model="hour" type="number" min="0" max="23" required></label>
          <label class="field"><span>北京时间（分）</span><input v-model="minute" type="number" min="0" max="59" required></label>
          <label class="field"><span>任务授权天数</span><input v-model="days" type="number" min="1" max="90" required></label>
        </template>
        <div class="checks wide">
          <span class="hint">始终检查是否需要讨论区签到；其余项目按需开启。</span>
          <label class="check"><input v-model="read" type="checkbox">执行浏览任务，最多三个帖子</label>
          <label class="check"><input v-model="like" type="checkbox">执行点赞任务，最多五个帖子</label>
          <label v-if="like" class="check"><input v-model="unlike" type="checkbox">点赞后取消该点赞</label>
          <label class="check"><input v-model="share" type="checkbox">执行一次分享任务</label>
          <label class="check"><input v-model="notify" type="checkbox">完成后私聊通知账号所属用户</label>
          <label class="check"><input v-model="confirmed" type="checkbox">确认以所选账号执行上述操作，并创建有期限的定时委托</label>
        </div>
        <div class="actions"><button class="primary" type="submit" :disabled="!confirmed || !account">{{ busy ? '正在开启…' : '开启账号任务' }}</button></div>
      </fieldset>
    </form>
    <StatusLine v-if="lister.loading.value && !items.length" kind="info" text="正在读取…" />
    <StatusLine kind="error" :text="error"><template v-if="lister.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="lister.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <ul v-if="items.length" class="rows">
      <li v-for="task in items" :key="task.ref">
        <div class="row-main">
          <strong>{{ task.role.nickname }} · 社区任务</strong>
          <div class="row-meta"><span class="chip" :class="{ warning: !task.enabled }">{{ task.enabled ? '已开启' : '已停止' }}</span><span>{{ schedule(task) }}</span><span>所属用户 {{ task.owner.actor_id }}</span></div>
          <div class="row-meta"><span>{{ communityLabels[task.last_code] || task.last_code || '等待执行' }}</span><span v-if="task.kind === 'community'">进度 {{ task.community?.cursor || 0 }} / {{ task.community?.steps?.length || 0 }}</span><span>到期 {{ formatTime(task.expires_at_ms) }}</span></div>
        </div>
        <div class="row-actions"><button class="compact danger" type="button" :disabled="busy" @click="removal = task.ref">停止并移除</button></div>
        <div v-if="removal === task.ref" class="confirm"><p>停止并移除这条账号任务？开启时创建的委托会一并停止。</p><div class="actions"><button class="compact" type="button" :disabled="busy" @click="removal = ''">保留</button><button class="compact destructive" type="button" :disabled="busy" @click="remove(task)">确认移除</button></div></div>
      </li>
    </ul>
    <EmptyState v-else-if="!lister.loading.value" icon="chat" title="尚未创建账号任务" text="选择执行方式与要执行的项目，确认后为当前账号开启。" />
  </section>
</template>
