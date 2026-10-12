<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Icon from './Icon.vue'
import RolePicker from './RolePicker.vue'
import { useShell } from './context'
import { sectionStorageKey } from './section-storage'

export interface Section { id: string; label: string; icon: string; count?: number }

// Workspace is one host page: a title row, the shared role picker when the
// page works on a bound role, a rail of sections and one active section.
// The active section is remembered for the browser session per page.
const props = defineProps<{ title: string; description?: string; sections: Section[]; roles?: boolean; roleLabel?: string }>()
// section reports the active section, first on setup, so a workspace that
// keeps a section's state outside the slot can read only while it is shown.
const emit = defineEmits<{ section: [id: string] }>()
const shell = useShell()
const storageKey = sectionStorageKey(shell.page, shell.bot.value)
function remembered(): string {
  try { return sessionStorage.getItem(storageKey) ?? '' } catch { return '' }
}
const active = ref(props.sections.some(section => section.id === remembered()) ? remembered() : props.sections[0]?.id ?? '')
watch(active, value => { try { sessionStorage.setItem(storageKey, value) } catch { /* per-session convenience only */ } })
watch(active, value => emit('section', value), { immediate: true })
const current = computed(() => props.sections.find(section => section.id === active.value))
const single = computed(() => props.sections.length <= 1)
function select(id: string) {
  active.value = id
  document.getElementById(`rail-${id}`)?.scrollIntoView({ inline: 'nearest', block: 'nearest' })
}
function move(offset: number) {
  const index = props.sections.findIndex(section => section.id === active.value)
  const next = props.sections[(index + offset + props.sections.length) % props.sections.length]
  if (next) { active.value = next.id; (document.getElementById(`rail-${next.id}`) as HTMLButtonElement | null)?.focus() }
}
</script>
<template>
  <header class="workspace-header">
    <div class="titles">
      <h1>{{ props.title }}</h1>
      <p v-if="props.description">{{ props.description }}</p>
      <p v-if="shell.bot.value" class="hint">当前机器人：{{ shell.bot.value.nickname || shell.bot.value.bot_id }}<template v-if="shell.bot.value.nickname">（{{ shell.bot.value.bot_id }}）</template><template v-if="shell.bot.value.state !== 'online'"> · {{ shell.bot.value.state === 'no_connection' ? '无连接' : '离线' }}，仍可编辑已保存数据</template></p>
    </div>
    <RolePicker v-if="props.roles" :label="props.roleLabel" />
  </header>
  <div class="workspace-body" :class="{ single }">
    <nav v-if="!single" :aria-label="`${props.title}的分区`">
      <ul class="rail" @keydown.down.prevent="move(1)" @keydown.up.prevent="move(-1)" @keydown.right.prevent="move(1)" @keydown.left.prevent="move(-1)">
        <li v-for="section in props.sections" :key="section.id">
          <button :id="`rail-${section.id}`" type="button" :aria-current="section.id === active ? 'true' : undefined" :tabindex="section.id === active ? undefined : -1" @click="select(section.id)">
            <Icon :name="section.icon" />
            <span>{{ section.label }}</span>
            <small v-if="section.count !== undefined">{{ section.count }}</small>
          </button>
        </li>
      </ul>
    </nav>
    <div :key="active" class="workspace-content" :aria-label="current?.label">
      <slot :active="active" :select="select" />
    </div>
  </div>
</template>
<style scoped>
.workspace-body.single { grid-template-columns: minmax(0, 1fr); }
</style>
