<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import StatusLine from '../shell/StatusLine.vue'
import EmptyState from '../shell/EmptyState.vue'
import { useShell } from '../shell/context'
import { useOperation } from '../shell/status'
import GroupSettings from '../GroupSettings.vue'
import ChallengeRank, { type Scope } from '../ChallengeRank.vue'

const shell = useShell()
const sections: Section[] = [
  { id: 'settings', label: '群设置', icon: 'people' },
  { id: 'rank', label: '挑战排行', icon: 'trophy' },
  { id: 'push', label: '公告推送', icon: 'megaphone' },
]

// 挑战排行 works on one group: the one opened from 群设置 with “查看挑战榜”,
// or any group listed there since the page opened.
const rankScope = ref<Scope | null>(null)
const known = ref<Scope[]>([])
function scopeKey(scope: Scope) { return JSON.stringify([scope.source_protocol, scope.bot_id, scope.group_id]) }
function remember(scopes: Scope[]) {
  const seen = new Map(known.value.map(scope => [scopeKey(scope), scope]))
  for (const scope of scopes) seen.set(scopeKey(scope), scope)
  known.value = [...seen.values()]
}
function openRank(scope: Scope, select: (id: string) => void) { remember([scope]); rankScope.value = scope; select('rank') }
const rankKey = computed({
  get: () => rankScope.value ? scopeKey(rankScope.value) : '',
  set: value => { rankScope.value = known.value.find(scope => scopeKey(scope) === value) ?? null },
})

// 公告推送: the groups that turned on news pushes in chat, reloaded whenever
// the section opens.
interface Subscription { ref: string; owner: { source_protocol: string; bot_id: string }; group_id: string; kinds: string[]; warned_on?: string; last_code?: string }
const push = useOperation('推送操作未完成，请重试。')
const subscriptions = ref<Subscription[]>([]), removeRef = ref('')
const kindNames: Record<string, string> = { announce: '公告', info: '资讯', expiry: '到期活动' }
const prefix = computed(() => shell.game.value?.prefix ?? '')
const pushSection = ref<HTMLElement | null>(null)
watch(pushSection, element => { if (element) void loadSubscriptions() })
async function listSubscriptions() { subscriptions.value = (await shell.invoke<{ items: Subscription[] }>('content.subscription.list')).items }
async function loadSubscriptions() { await push.run(listSubscriptions) }
async function removeSubscription() {
  await push.run(async () => {
    await shell.invoke('content.subscription.remove', { ref: removeRef.value, confirm: true })
    removeRef.value = ''
    await listSubscriptions()
  }, '已停止该群的推送。')
}
</script>

<template>
  <Workspace title="群管理" description="按机器人和群保存功能开关、回复方式与群别名，查看各群的挑战排行，并管理公告推送。" :sections="sections">
    <template #default="{ active, select }">
      <GroupSettings v-if="active === 'settings'" @rank="scope => openRank(scope, select)" @listed="remember" />

      <section v-else-if="active === 'rank'" class="section" aria-labelledby="rank-title">
        <div class="section-heading"><div class="titles"><h2 id="rank-title">挑战排行</h2><p>各群本地保存的挑战成绩。先在群设置中对某个群点“查看挑战榜”，或在下方选择本次已列出的群。</p></div></div>
        <div v-if="known.length" class="field-row">
          <label class="field"><span>群</span><select v-model="rankKey"><option value="" disabled>选择群</option><option v-for="scope in known" :key="scopeKey(scope)" :value="scopeKey(scope)">{{ scope.group_id }}</option></select></label>
        </div>
        <ChallengeRank v-if="rankScope" :key="scopeKey(rankScope)" :scope="rankScope" />
        <EmptyState v-else icon="trophy" title="还没有选择群" text="在群设置里对某个群点“查看挑战榜”后，这里会显示该群的挑战成绩。"><button type="button" @click="select('settings')">前往群设置</button></EmptyState>
      </section>

      <section v-else-if="active === 'push'" ref="pushSection" class="section" aria-labelledby="push-title">
        <div class="section-heading"><div class="titles"><h2 id="push-title">公告推送</h2><p>群管理员在群里发送“{{ prefix }}开启公告推送”“{{ prefix }}开启资讯推送”或“{{ prefix }}开启到期活动推送”开启，把“开启”换成“关闭”即停止。每五分钟推送两小时内的一条新帖，每天十点后推送一天内结束的活动；超级管理员发送“{{ prefix }}推送公告”可立即检查。</p></div><button type="button" :disabled="push.busy.value" @click="loadSubscriptions">刷新</button></div>
        <StatusLine v-if="push.busy.value && !subscriptions.length" kind="info" text="正在读取推送列表…" />
        <StatusLine kind="error" :text="push.error.value"><template v-if="!subscriptions.length" #actions><button class="compact" type="button" :disabled="push.busy.value" @click="loadSubscriptions">重试</button></template></StatusLine>
        <StatusLine kind="success" :text="push.notice.value" />
        <ul v-if="subscriptions.length" class="rows">
          <li v-for="item in subscriptions" :key="item.ref">
            <div class="row-main"><strong>群 {{ item.group_id }}</strong><div class="row-meta"><span>{{ item.kinds.map(kind => kindNames[kind] ?? kind).join('、') }}</span><span>{{ item.last_code || '等待首次检查' }}</span><span v-if="item.warned_on">到期活动已于 {{ item.warned_on }} 推送</span></div></div>
            <div class="row-actions"><button type="button" class="compact danger" :disabled="push.busy.value" @click="removeRef = item.ref">移除此群的推送</button></div>
            <div v-if="removeRef === item.ref" class="confirm"><p>确认停止群 {{ item.group_id }} 的全部推送？</p><div class="actions"><button type="button" class="compact" :disabled="push.busy.value" @click="removeRef = ''">保留</button><button type="button" class="compact destructive" :disabled="push.busy.value" @click="removeSubscription">确认移除</button></div></div>
          </li>
        </ul>
        <EmptyState v-else-if="!push.busy.value" icon="megaphone" title="没有开启推送的群" text="群管理员在群里发送开启命令后，这里会列出该群。" />
      </section>
    </template>
  </Workspace>
</template>
