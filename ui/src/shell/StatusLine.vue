<script setup lang="ts">
import Icon from './Icon.vue'
// StatusLine is an inline status: icon, text and an optional action, never a
// banner across the workspace. error and attention announce assertively.
const props = defineProps<{ kind: 'error' | 'attention' | 'success' | 'info'; text: string }>()
const icons = { error: 'error', attention: 'alert', success: 'check', info: 'info' } as const
</script>
<template>
  <div v-if="props.text" class="status" :class="props.kind" :role="props.kind === 'error' || props.kind === 'attention' ? 'alert' : 'status'">
    <Icon :name="icons[props.kind]" />
    <p>{{ props.text }}<slot /></p>
    <slot name="actions" />
  </div>
</template>
