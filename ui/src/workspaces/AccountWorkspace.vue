<script setup lang="ts">
import Workspace, { type Section } from '../shell/Workspace.vue'
import GameSignin from '../GameSignin.vue'
import CommunityPanel from '../CommunityPanel.vue'
import CodesPanel from '../CodesPanel.vue'
import BillingRecords from '../BillingRecords.vue'
import PlaytimeEstimate from '../PlaytimeEstimate.vue'
import StatusLine from '../shell/StatusLine.vue'

// 账号操作: actions run right now for the selected role; each section owns its state.
const sections: Section[] = [
  { id: 'signin', label: '签到', icon: 'checkin' },
  { id: 'community', label: '米游社任务', icon: 'chat' },
  { id: 'codes', label: '兑换码', icon: 'ticket' },
  { id: 'billing', label: '资产记录', icon: 'wallet' },
  { id: 'playtime', label: '在线时长', icon: 'clock' },
]
</script>

<template>
  <Workspace title="账号操作" description="用当前角色立即执行的操作：签到、米游社任务、兑换码、资产记录与在线时长估算；结果直接显示在下方。" :sections="sections" roles>
    <template #default="{ active }">
      <StatusLine v-if="active === 'codes'" kind="info" text="兑换码列表为全局资料；兑换操作使用当前账号的角色授权。" />
      <GameSignin v-if="active === 'signin'" />
      <CommunityPanel v-else-if="active === 'community'" />
      <CodesPanel v-else-if="active === 'codes'" />
      <BillingRecords v-else-if="active === 'billing'" />
      <PlaytimeEstimate v-else-if="active === 'playtime'" />
    </template>
  </Workspace>
</template>
