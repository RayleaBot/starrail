<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { useReader } from './reader'

// 米游社任务 (manual): the coin missions and the discussion posts of the
// selected role's account are read as soon as that account is chosen; each
// community step stays one explicit, confirmed action.
interface Mission { mission_id?: number; happened_times?: number; mission_cnt?: number; is_get_award?: boolean }
interface Post { post_id: string; title: string }
interface Result { view: View; result: { data: { states?: Mission[]; posts?: Post[] } } }
const shell = useShell()
const operation = useOperation('米游社操作未完成，请重试。')
const { busy, error, run } = operation
const statusView = ref<View | null>(null), postsView = ref<View | null>(null), view = ref<View | null>(null)
const missions = ref<Mission[]>([]), posts = ref<Post[]>([]), post = ref(''), step = ref('read'), confirmed = ref(false), running = ref('')
const account = computed(() => shell.selectedRole.value?.account)
const accountRef = computed(() => account.value?.ref ?? '')
const stepNames: Record<string, string> = { read: '浏览并计入任务', like: '点赞', unlike: '取消点赞', share: '分享任务' }
// The posts are read with the first status and on refresh; after a step only
// the status is read again, since the posts do not change with it.
let withPosts = true
const reader = useReader(operation, async stale => {
  const current = account.value, includePosts = withPosts
  withPosts = false
  if (!current) return
  const status = await shell.invoke<Result>('community.status', { account_ref: current.ref })
  if (stale()) return
  statusView.value = status.view; missions.value = status.result.data.states ?? []
  if (!includePosts) return
  const listed = await shell.invoke<Result>('community.posts', { account_ref: current.ref })
  if (stale()) return
  postsView.value = listed.view; posts.value = listed.result.data.posts ?? []
  if (!posts.value.some(item => item.post_id === post.value)) post.value = posts.value[0]?.post_id ?? ''
})
function load(includePosts: boolean) { withPosts = withPosts || includePosts; reader.request() }
watch(accountRef, () => { statusView.value = null; postsView.value = null; view.value = null; missions.value = []; posts.value = []; post.value = ''; confirmed.value = false; load(true) })
watch([step, post], () => { confirmed.value = false })
if (account.value) load(true)
async function perform(payload: Record<string, unknown>) {
  const current = account.value
  if (!current) return
  running.value = String(payload.step)
  const done = await run(async () => { view.value = (await shell.invoke<Result>('community.run', { account_ref: current.ref, ...payload })).view })
  confirmed.value = false
  if (done) load(false)
}
</script>

<template>
  <section class="section" aria-labelledby="community-title">
    <div class="section-heading"><div class="titles"><h2 id="community-title">米游社任务</h2><p>米游币任务按米游社账号统计，三游戏共用同一登录。选择角色后自动读取所属账号的任务进度与本游戏讨论区帖子；单次操作需明确执行，定时执行在“定时任务”页开启。</p></div><button type="button" :disabled="busy || !account" @click="load(true)">{{ reader.loading.value ? '正在读取…' : '刷新' }}</button></div>
    <p v-if="account" class="hint">当前账号：{{ account.account_label }} · 用户 {{ account.owner.actor_id }}</p>
    <div class="actions"><button type="button" :disabled="busy || !account" @click="perform({ step: 'sign', confirm: true })">{{ busy && !reader.loading.value && running === 'sign' ? '正在签到…' : '社区签到' }}</button></div>
    <StatusLine v-if="reader.loading.value" kind="info" text="正在读取米游币任务与讨论区帖子…" />
    <StatusLine kind="error" :text="error"><template v-if="reader.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="load(true)">重试</button></template></StatusLine>
    <BusinessView v-if="view" :value="view" />
    <div v-if="statusView || missions.length" class="subsection">
      <h3>米游币与任务</h3>
      <BusinessView v-if="statusView" :value="statusView" />
      <div v-if="missions.length" class="table-wrap">
        <table>
          <thead><tr><th>任务编号</th><th class="num">已发生次数</th><th class="num">要求次数</th><th>奖励状态</th></tr></thead>
          <tbody><tr v-for="(mission, index) in missions" :key="index"><td>{{ mission.mission_id ?? '未提供' }}</td><td class="num">{{ mission.happened_times ?? '未提供' }}</td><td class="num">{{ mission.mission_cnt ?? '未提供' }}</td><td>{{ mission.is_get_award === true ? '已领取' : mission.is_get_award === false ? '未领取' : '未提供' }}</td></tr></tbody>
        </table>
      </div>
    </div>
    <div v-if="posts.length" class="subsection">
      <h3>讨论区帖子</h3>
      <form class="field-grid" @submit.prevent="confirmed && perform({ step, post_id: post, confirm: true })">
        <fieldset :disabled="busy">
          <label class="field"><span>讨论区帖子</span><select v-model="post"><option v-for="item in posts" :key="item.post_id" :value="item.post_id">{{ item.title }} · {{ item.post_id }}</option></select></label>
          <label class="field"><span>社区操作</span><select v-model="step"><option v-for="(name, id) in stepNames" :key="id" :value="id">{{ name }}</option></select></label>
          <label class="check wide"><input v-model="confirmed" type="checkbox">确认以当前账号对该帖子执行{{ stepNames[step] }}</label>
          <div class="actions"><button class="primary" type="submit" :disabled="!confirmed || !post">{{ busy && !reader.loading.value && running !== 'sign' ? '正在执行…' : '执行所选社区操作' }}</button></div>
        </fieldset>
      </form>
      <BusinessView v-if="postsView" :value="postsView" />
    </div>
  </section>
</template>
