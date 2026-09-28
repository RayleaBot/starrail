<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { overseas, useShell } from './shell/context'
import { useOperation } from './shell/status'
import { cloudTask } from './cloud'

// 兑换码: the official preview codes, and web redemption for an overseas role.
interface CodeRow { code: string; reward: string; expires_at?: string; expiry_estimated: boolean }
const shell = useShell()
const task = cloudTask(shell.invoke, 'content')
const { busy, error, run } = useOperation('兑换未完成，请重试。')
const code = ref(''), confirmed = ref(false), view = ref<View | null>(null)
const rows = computed(() => (task.job.value?.result?.items ?? []) as CodeRow[])
const abroad = computed(() => !!shell.selectedRole.value && overseas(shell.selectedRole.value.role.region))
watch(shell.selection, () => { confirmed.value = false; view.value = null })
function fill(value: string) { code.value = value; confirmed.value = false }
function edited() { confirmed.value = false; view.value = null }
async function redeem() {
  if (!shell.selectedRole.value || !confirmed.value) return
  await run(async () => {
    view.value = null
    view.value = (await shell.invoke<{ view: View }>('redeem.run', { ...shell.chosen(), code: code.value.trim(), confirm: true })).view
    confirmed.value = false
  })
}
onUnmounted(task.dispose)
</script>

<template>
  <section class="section" aria-labelledby="codes-title">
    <div class="section-heading"><div class="titles"><h2 id="codes-title">兑换码</h2><p>读取本游戏官方前瞻入口公布的兑换码；是否可兑换以官方结果为准。国服请在游戏内兑换，海外角色可在下方直接提交。</p></div><button type="button" :disabled="task.busy.value || task.job.value?.state === 'running'" @click="task.start({ kind: 'codes' })">{{ task.busy.value ? '正在查询…' : '查询公开兑换码' }}</button></div>
    <StatusLine kind="error" :text="task.error.value || task.job.value?.message || ''" />
    <StatusLine v-if="task.job.value?.state === 'running'" kind="info" text="正在读取官方入口…"><template #actions><button class="compact" type="button" @click="task.cancel">取消</button><button class="compact" type="button" @click="task.poll">刷新进度</button></template></StatusLine>
    <template v-if="task.job.value?.state === 'completed'">
      <h3 v-if="task.job.value?.result?.title">{{ task.job.value?.result?.title }}</h3>
      <ul v-if="rows.length" class="rows">
        <li v-for="row in rows" :key="row.code">
          <div class="row-main"><strong class="mono">{{ row.code }}</strong><div class="row-meta"><span>{{ row.reward }}</span><span v-if="row.expires_at">{{ row.expiry_estimated ? '参考估计期限' : '有效期' }}：{{ row.expires_at }}</span></div></div>
          <div class="row-actions"><button class="compact" type="button" @click="fill(row.code)">填入兑换</button></div>
        </li>
      </ul>
      <EmptyState v-else icon="ticket" title="当前官方入口没有可读取的兑换码" />
    </template>
    <div class="subsection">
      <h3>海外角色兑换</h3>
      <form class="field-row" @submit.prevent="redeem">
        <fieldset :disabled="busy">
          <label class="field"><span>兑换码</span><input v-model="code" maxlength="64" @input="edited"></label>
          <label class="check"><input v-model="confirmed" type="checkbox" :disabled="!abroad">确认使用此代码为当前角色兑换</label>
          <button class="primary" type="submit" :disabled="!abroad || !confirmed || !code.trim()">{{ busy ? '正在兑换…' : '提交兑换' }}</button>
        </fieldset>
      </form>
      <StatusLine v-if="shell.selectedRole.value && !abroad" kind="info" text="网页兑换仅支持海外区服，请为此国服角色在游戏内兑换。" />
      <StatusLine kind="error" :text="error" />
      <BusinessView v-if="view" :value="view" />
    </div>
  </section>
</template>
