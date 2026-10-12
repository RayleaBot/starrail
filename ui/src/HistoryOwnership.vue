<script setup lang="ts">
import { computed, ref } from 'vue'
import type { BotAccount } from '@rayleabot/plugin-ui'
import { accountPath, historicalRecords, type HistoryEntry, type HistoryRecord } from './bot-history'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'

const shell = useShell()
const operation = useOperation('历史数据读取失败，请重试。')
const entries = ref<HistoryEntry[]>([]), total = ref(0), loaded = ref(false)
const bots = ref<BotAccount[]>([]), cursor = ref<string | undefined>(), target = ref('')
const candidate = ref<HistoryEntry | null>(null), mode = ref<'remove' | 'assign'>('remove')
const detail = ref(''), detailKey = ref('')
const counts = computed(() => ({ missing: entries.value.filter(item => item.state === 'missing').length, unresolved: entries.value.filter(item => item.state !== 'missing').length }))
const kinds: Record<string, string> = { groups: '群设置与排行', tasks: '提醒与任务', sync: '每日抽卡同步', push: '群推送', interaction: '原图记录', monthly: '月报档案', panels: '面板历史', billing: '资产档案', exchange: '交换档案', simulation: '聊天模拟档案' }
const states = { unassigned: '待选择机器人账号', missing: '账号不在目录中', unknown: '档案未保存账号身份' }
function key(item: HistoryRecord) { return `${item.kind}:${item.ref}` }
function assignable(item: HistoryEntry) { return item.state === 'unassigned' && item.owner.source_protocol === 'onebot11' }
function payload(item: HistoryRecord) { return { kind: item.kind, ref: item.ref, version: item.version } }
async function load() {
  const result = await shell.invoke<{ items: HistoryRecord[]; total: number }>('history.list')
  const visible = await historicalRecords(result.items, shell.apiRequest)
  entries.value = visible; total.value = result.total; loaded.value = true
  candidate.value = null; detail.value = ''; detailKey.value = ''
}
async function loadBots(more = false) {
  const result = await shell.apiRequest<{ items: BotAccount[]; next_cursor?: string }>('GET', `/api/bot-accounts?source_protocol=onebot11&limit=100${more && cursor.value ? `&cursor=${encodeURIComponent(cursor.value)}` : ''}`)
  bots.value = more ? [...bots.value, ...result.items] : result.items
  cursor.value = result.next_cursor
}
async function inspect(item: HistoryEntry) {
  await operation.run(async () => {
    const result = await shell.invoke<{ record: unknown }>('history.get', payload(item))
    detail.value = JSON.stringify(result.record, null, 2); detailKey.value = key(item)
  })
}
async function ask(item: HistoryEntry, action: 'remove' | 'assign') {
  await operation.run(async () => {
    if (action === 'assign') await loadBots()
    target.value = ''; candidate.value = item; mode.value = action
  })
}
async function apply() {
  const item = candidate.value
  if (!item) return
  await operation.run(async () => {
    if (mode.value === 'remove') await shell.invoke('history.remove', { ...payload(item), confirm: true })
    else {
      const bot = await shell.apiRequest<BotAccount>('GET', accountPath('onebot11', target.value))
      await shell.assignHistory({ source_protocol: bot.source_protocol, bot_id: bot.bot_id }, { ...payload(item), confirm: true })
    }
    await load()
  }, mode.value === 'remove' ? '记录已删除。' : '归属已保存；提醒与同步任务保持暂停。')
}
void operation.run(load)
</script>

<template>
  <section class="section" aria-labelledby="history-title">
    <div class="section-heading"><div class="titles"><h2 id="history-title">历史数据与归属</h2><p>查看和清理未归属记录、目录外账号的数据。仅缺少账号的 OneBot11 记录可重新归属。</p></div><button :disabled="operation.busy.value" @click="operation.run(load)">刷新</button></div>
    <p v-if="loaded" class="hint">已检查 {{ total }} 份记录 · 未确定归属 {{ counts.unresolved }} 份 · 目录外 {{ counts.missing }} 份</p>
    <p class="hint">无法确认账号的档案可查看或删除。QQ 官方记录与完整账号记录不能改挂；删除任务后，原委托到期失效。</p>
    <StatusLine kind="error" :text="operation.error.value" />
    <StatusLine kind="success" :text="operation.notice.value" />
    <ul v-if="entries.length" class="rows">
      <li v-for="item in entries" :key="key(item)">
        <div class="row-main"><strong>{{ kinds[item.kind] }}<template v-if="item.label"> · {{ item.label }}</template></strong><div class="row-meta"><span class="chip warning">{{ states[item.state] }}</span><span v-if="item.owner.source_protocol">{{ item.owner.source_protocol }} {{ item.owner.bot_id }}</span><span>{{ item.count }} 条</span></div></div>
        <div class="row-actions"><button class="compact" :disabled="operation.busy.value" @click="inspect(item)">查看</button><button v-if="assignable(item)" class="compact" :disabled="operation.busy.value" @click="ask(item, 'assign')">选择归属</button><button class="compact danger" :disabled="operation.busy.value" @click="ask(item, 'remove')">删除</button></div>
        <pre v-if="detailKey === key(item)" class="history-data">{{ detail }}</pre>
        <div v-if="candidate && key(candidate) === key(item)" class="confirm">
          <template v-if="mode === 'assign'">
            <label class="field"><span>归入 OneBot11 账号</span><select v-model="target"><option value="" disabled>选择账号</option><option v-for="bot in bots" :key="bot.bot_id" :value="bot.bot_id">{{ bot.nickname || bot.bot_id }} · {{ bot.bot_id }}</option></select></label>
            <button v-if="cursor" class="compact" :disabled="operation.busy.value" @click="operation.run(() => loadBots(true))">更多账号</button>
            <p>提醒与同步任务归属后暂停，请重新开启。</p>
          </template>
          <p v-else>确认删除这份记录及其本地数据？</p>
          <div class="actions"><button class="compact" :disabled="operation.busy.value" @click="candidate = null">取消</button><button :class="['compact', mode === 'remove' ? 'destructive' : 'primary']" :disabled="operation.busy.value || (mode === 'assign' && !target)" @click="apply">{{ mode === 'remove' ? '确认删除' : '确认归属' }}</button></div>
        </div>
      </li>
    </ul>
    <EmptyState v-else-if="loaded" icon="info" title="没有待处理的历史数据" text="账号目录变化后可刷新检查。" />
  </section>
</template>

<style scoped>
.history-data { max-height: 24rem; overflow: auto; width: 100%; white-space: pre-wrap; overflow-wrap: anywhere; }
</style>
