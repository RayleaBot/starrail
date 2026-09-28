<script setup lang="ts">
import { ref } from 'vue'
import { useShell } from './context'
import StatusLine from './StatusLine.vue'
// RolePicker is the one role selection of an account-bound workspace. Every
// section of the workspace reads shell.selection instead of asking again.
const shell = useShell()
const props = defineProps<{ disabled?: boolean; label?: string }>()
const busy = ref(false)
async function reload(page = 0) {
  if (busy.value) return
  busy.value = true
  try { await shell.loadAccounts(page) } finally { busy.value = false }
}
</script>
<template>
  <div class="role-picker">
    <label class="field">
      <span>{{ props.label ?? '游戏角色' }}</span>
      <select v-model="shell.selection.value" :disabled="props.disabled || busy || !shell.roles.value.length">
        <option v-if="!shell.roles.value.length" value="">{{ shell.accountsLoaded.value ? '没有已授权角色' : '正在读取账号…' }}</option>
        <option v-for="item in shell.roles.value" :key="item.key" :value="item.key">{{ item.label }}</option>
      </select>
    </label>
    <button type="button" :disabled="props.disabled || busy" @click="reload()">刷新账号</button>
    <button v-if="shell.nextAccountPage.value !== null" type="button" :disabled="props.disabled || busy" @click="reload(shell.nextAccountPage.value!)">更多账号</button>
    <StatusLine v-if="shell.accountIssue.value" kind="attention" :text="shell.accountIssue.value + ' 可在米游社账号插件的管理页扫码或调整授权。'" />
    <StatusLine v-else-if="shell.accountsLoaded.value && !shell.roles.value.length" kind="info" :text="`没有已授权角色。请先向机器人发送“扫码登录”，并授权${shell.game.value?.name ?? '此游戏'}。`" />
  </div>
</template>
