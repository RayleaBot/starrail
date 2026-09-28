<script setup lang="ts">
import { computed, useId, watch } from 'vue'
import { useShell } from './context'

// CharacterSelect offers characters to pick instead of a name to type: the
// selected role's own characters (source role) or the bundled catalog
// (source catalog). The value is the character ID the actions take.
const props = withDefaults(defineProps<{
  modelValue: string
  source: 'role' | 'catalog'
  label?: string
  // none labels an empty choice; without it the first character is chosen
  // when autoselect is set, otherwise the field waits for a choice.
  none?: string
  autoselect?: boolean
  exclude?: string[]
  kind?: 'character' | 'weapon'
  disabled?: boolean
}>(), { label: '角色', kind: 'character', exclude: () => [], autoselect: false, disabled: false })
const emit = defineEmits<{ 'update:modelValue': [id: string]; change: [id: string] }>()
const shell = useShell()
const id = useId()
const state = computed(() => props.source === 'role' ? shell.charactersState.value : shell.catalogState.value)
const options = computed(() => {
  const excluded = new Set(props.exclude)
  if (props.source === 'role') return shell.characters.value.filter(item => !excluded.has(item.id)).map(item => ({ id: item.id, label: `${item.name} · ${item.level} 级` }))
  return shell.catalog.value
    .filter(entry => entry.kind === props.kind && !excluded.has(entry.id))
    .sort((a, b) => b.rarity - a.rarity || a.name.localeCompare(b.name, 'zh-Hans-CN'))
    .map(entry => ({ id: entry.id, label: entry.rarity ? `${entry.name} · ${entry.rarity} 星` : entry.name }))
})
const waiting = computed(() => props.source === 'role' && !shell.selectedRole.value)
const placeholder = computed(() => {
  if (waiting.value) return '先在上方选择游戏角色'
  if (state.value === 'loading') return '正在读取角色列表…'
  if (state.value === 'failed') return '角色列表读取失败'
  if (state.value === 'ready' && !options.value.length) return '没有可选的角色'
  return '选择角色'
})
function load(force = false) { if (props.source === 'role') void shell.loadCharacters(force); else void shell.loadCatalog() }
watch(() => [props.source, shell.selection.value], () => load(), { immediate: true })
// A loaded list drops a value it no longer offers and, when asked, picks the
// first entry instead, so the section reads without a further click, also
// after a role change.
watch([options, state], ([list, current]) => {
  if (current !== 'ready' || (props.modelValue && list.some(item => item.id === props.modelValue))) return
  const next = props.autoselect && props.none === undefined ? list[0]?.id ?? '' : ''
  if (next !== props.modelValue) { emit('update:modelValue', next); emit('change', next) }
}, { immediate: true })
function pick(event: Event) { const value = (event.target as HTMLSelectElement).value; emit('update:modelValue', value); emit('change', value) }
</script>

<template>
  <div class="field">
    <label :for="id">{{ props.label }}</label>
    <select :id="id" :value="props.modelValue" :disabled="props.disabled || waiting || state === 'loading'" :required="props.none === undefined" @change="pick">
      <option v-if="props.none !== undefined" value="">{{ props.none }}</option>
      <option v-else-if="!props.modelValue" value="" disabled>{{ placeholder }}</option>
      <option v-for="option in options" :key="option.id" :value="option.id">{{ option.label }}</option>
    </select>
    <span v-if="state === 'failed'" class="hint danger-text">{{ props.source === 'role' ? shell.charactersIssue.value : '图鉴清单读取失败。' }} <button type="button" class="compact ghost" @click="load(true)">重试</button></span>
    <span v-else-if="props.source === 'role' && state === 'ready' && !options.length" class="hint">官方角色列表里没有可选的角色。</span>
  </div>
</template>
