<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'

// 米游社任务 (manual): the coin missions of the selected role's account, one explicit step at a time.
interface Mission { mission_id?: number; happened_times?: number; mission_cnt?: number; is_get_award?: boolean }
interface Post { post_id: string; title: string }
const shell = useShell()
const { busy, error, run } = useOperation('米游社操作未完成，请重试。')
const view = ref<View | null>(null), missions = ref<Mission[]>([]), posts = ref<Post[]>([]), post = ref(''), step = ref('read'), confirmed = ref(false), running = ref('')
const account = computed(() => shell.selectedRole.value?.account)
const stepNames: Record<string, string> = { read: '浏览并计入任务', like: '点赞', unlike: '取消点赞', share: '分享任务' }
watch(shell.selection, () => { view.value = null; missions.value = []; posts.value = []; post.value = ''; confirmed.value = false })
watch([step, post], () => { confirmed.value = false })
async function perform(action: string, payload: Record<string, unknown> = {}) {
  const current = account.value
  if (!current) return
  running.value = action + (typeof payload.step === 'string' ? `:${payload.step}` : '')
  await run(async () => {
    const result = await shell.invoke<{ view: View; result: { data: { states?: Mission[]; posts?: Post[] } } }>(action, { account_ref: current.ref, ...payload })
    view.value = result.view
    if (action === 'community.status') missions.value = result.result.data.states ?? []
    if (action === 'community.posts') { posts.value = result.result.data.posts ?? []; post.value = posts.value[0]?.post_id ?? '' }
  })
  confirmed.value = false
}
function label(key: string, idle: string, working: string): string { return busy.value && running.value === key ? working : idle }
</script>

<template>
  <section class="section" aria-labelledby="community-title">
    <div class="section-heading"><div class="titles"><h2 id="community-title">米游社任务</h2><p>米游币任务按米游社账号统计，三游戏共用同一登录。这里查询当前角色所属账号的进度并明确执行单次操作；定时执行在“定时任务”页开启。</p></div></div>
    <p v-if="account" class="hint">当前账号：{{ account.account_label }} · 用户 {{ account.owner.actor_id }}</p>
    <div class="actions">
      <button type="button" :disabled="busy || !account" @click="perform('community.status')">{{ label('community.status', '查询米游币与任务', '正在查询…') }}</button>
      <button type="button" :disabled="busy || !account" @click="perform('community.posts')">{{ label('community.posts', '读取本游戏讨论区帖子', '正在读取…') }}</button>
      <button type="button" :disabled="busy || !account" @click="perform('community.run', { step: 'sign', confirm: true })">{{ label('community.run:sign', '社区签到', '正在签到…') }}</button>
    </div>
    <StatusLine kind="error" :text="error" />
    <div v-if="missions.length" class="table-wrap">
      <table>
        <thead><tr><th>任务编号</th><th class="num">已发生次数</th><th class="num">要求次数</th><th>奖励状态</th></tr></thead>
        <tbody><tr v-for="(mission, index) in missions" :key="index"><td>{{ mission.mission_id ?? '未提供' }}</td><td class="num">{{ mission.happened_times ?? '未提供' }}</td><td class="num">{{ mission.mission_cnt ?? '未提供' }}</td><td>{{ mission.is_get_award === true ? '已领取' : mission.is_get_award === false ? '未领取' : '未提供' }}</td></tr></tbody>
      </table>
    </div>
    <form v-if="posts.length" class="field-grid" @submit.prevent="confirmed && perform('community.run', { step, post_id: post, confirm: true })">
      <fieldset :disabled="busy">
        <label class="field"><span>讨论区帖子</span><select v-model="post"><option v-for="item in posts" :key="item.post_id" :value="item.post_id">{{ item.title }} · {{ item.post_id }}</option></select></label>
        <label class="field"><span>社区操作</span><select v-model="step"><option v-for="(name, id) in stepNames" :key="id" :value="id">{{ name }}</option></select></label>
        <label class="check wide"><input v-model="confirmed" type="checkbox">确认以当前账号对该帖子执行{{ stepNames[step] }}</label>
        <div class="actions"><button class="primary" type="submit" :disabled="!confirmed || !post">{{ busy ? '正在执行…' : '执行所选社区操作' }}</button></div>
      </fieldset>
    </form>
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
