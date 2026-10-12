<script setup lang="ts">
import { ref } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import CloudQueries from '../CloudQueries.vue'
import CloudExchange from '../CloudExchange.vue'
import StatusLine from '../shell/StatusLine.vue'

const sections: Section[] = [
  { id: 'queries', label: '云查询', icon: 'cloud' },
  { id: 'exchange', label: '交换档案', icon: 'swap' },
]
// One switch for the configured ark token: every cloud request of either
// section sends it, so both sections show and edit the same value.
const authenticated = ref(false)
</script>

<template>
  <Workspace title="ark 云端" description="ark-plugin 提供的云端排名、面板、交换与截图识别；交换档案与识别试算使用上方选择的角色。" :sections="sections" roles>
    <template #default="{ active }">
      <StatusLine v-if="active === 'queries'" kind="info" text="公开云查询使用全局资料；交换档案与识别试算使用当前账号的角色授权。" />
      <CloudQueries v-if="active === 'queries'" v-model:authenticated="authenticated" />
      <CloudExchange v-else-if="active === 'exchange'" v-model:authenticated="authenticated" />
    </template>
  </Workspace>
</template>
