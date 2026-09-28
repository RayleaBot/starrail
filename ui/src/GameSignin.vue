<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'

// 签到: the official game sign-in of the selected role, run once by hand.
const shell = useShell()
const { busy, error, run, clear } = useOperation('签到操作未完成，请重试。')
const view = ref<View | null>(null), running = ref('')
watch(shell.selection, () => { view.value = null; clear() })
async function perform(action: string) {
  running.value = action
  await run(async () => {
    view.value = null
    view.value = (await shell.invoke<{ view: View }>(action, { ...shell.chosen(), ...(action === 'signin.run' ? { confirm: true } : {}) })).view
  })
}
function label(action: string, idle: string, working: string): string { return busy.value && running.value === action ? working : idle }
</script>

<template>
  <section class="section" aria-labelledby="signin-title">
    <div class="section-heading"><div class="titles"><h2 id="signin-title">签到</h2><p>通过米游社官方游戏签到为当前角色领取当日奖励。已签到不会重复提交；首次签到或验证要求需在官方应用处理，完成后可先查询签到状态，再执行当日签到。每日自动签到在“定时任务”页开启。</p></div></div>
    <div class="actions">
      <button type="button" :disabled="busy || !shell.selectedRole.value" @click="perform('signin.status')">{{ label('signin.status', '查询签到状态', '正在查询…') }}</button>
      <button type="button" :disabled="busy || !shell.selectedRole.value" @click="perform('signin.rewards')">{{ label('signin.rewards', '查看当月奖励', '正在读取…') }}</button>
      <button class="primary" type="button" :disabled="busy || !shell.selectedRole.value" @click="perform('signin.run')">{{ label('signin.run', '立即签到', '正在签到…') }}</button>
    </div>
    <StatusLine kind="error" :text="error" />
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
