<script setup lang="ts">
import Icon from './Icon.vue'
// EmptyState says what is missing and what fills it; the slot holds the action.
const props = defineProps<{ icon?: string; title: string; text?: string }>()
</script>
<template>
  <div class="empty">
    <Icon :name="props.icon ?? 'inbox'" />
    <strong>{{ props.title }}</strong>
    <p v-if="props.text">{{ props.text }}</p>
    <div v-if="$slots.default" class="actions"><slot /></div>
  </div>
</template>
