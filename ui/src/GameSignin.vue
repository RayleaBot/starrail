<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { useReader } from './reader'

// 签到: the sign-in status and the month's rewards of the selected role are
// read as soon as it is chosen; the sign-in itself stays one explicit action.
const shell = useShell()
const operation = useOperation('签到操作未完成，请重试。')
const { busy, error, run } = operation
const status = ref<View | null>(null), rewards = ref<View | null>(null), result = ref<View | null>(null)
const reader = useReader(operation, async stale => {
  if (!shell.selectedRole.value) return
  const chosen = shell.chosen()
  const current = (await shell.invoke<{ view: View }>('signin.status', chosen)).view
  if (stale()) return
  status.value = current
  const month = (await shell.invoke<{ view: View }>('signin.rewards', chosen)).view
  if (!stale()) rewards.value = month
})
watch(shell.selection, () => { status.value = null; rewards.value = null; result.value = null; reader.request() })
if (shell.selectedRole.value) reader.request()
async function signIn() {
  const done = await run(async () => {
    result.value = null
    result.value = (await shell.invoke<{ view: View }>('signin.run', { ...shell.chosen(), confirm: true })).view
  })
  if (done) reader.request()
}
</script>

<template>
  <section class="section" aria-labelledby="signin-title">
    <div class="section-heading"><div class="titles"><h2 id="signin-title">签到</h2><p>通过米游社官方游戏签到为当前角色领取当日奖励。选择角色后自动读取签到状态与当月奖励；已签到不会重复提交，首次签到或验证要求需在官方应用处理。每日自动签到在“定时任务”页开启。</p></div><button type="button" :disabled="busy || !shell.selectedRole.value" @click="reader.request()">{{ reader.loading.value ? '正在读取…' : '刷新' }}</button></div>
    <div class="actions"><button class="primary" type="button" :disabled="busy || !shell.selectedRole.value" @click="signIn">{{ busy && !reader.loading.value ? '正在签到…' : '立即签到' }}</button></div>
    <StatusLine v-if="reader.loading.value" kind="info" text="正在读取签到状态与当月奖励…" />
    <StatusLine kind="error" :text="error"><template v-if="reader.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="reader.request()">重试</button></template></StatusLine>
    <div v-if="result" class="subsection"><h3>本次签到</h3><BusinessView :value="result" /></div>
    <div v-if="status" class="subsection"><h3>签到状态</h3><BusinessView :value="status" /></div>
    <div v-if="rewards" class="subsection"><h3>本月奖励</h3><BusinessView :value="rewards" /></div>
  </section>
</template>
