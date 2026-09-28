<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { download, formatTime, useOperation } from './shell/status'
import { useReader } from './reader'

// 面板历史: saved snapshots of the selected role's characters, compared over
// time. Once a character's list is read, its two newest snapshots are
// compared by themselves; another pair compares on selection.
interface Snapshot { ref: string; saved_at_ms: number; version: string; name: string; level: number; weapon: string }
const shell = useShell()
const operation = useOperation('历史操作未完成，请重试。')
const { busy, error, notice, run } = operation
// The automatic comparison has its own lock so it neither blocks the form
// nor clears the notice of a save or removal that queued it.
const compare = useOperation('对比未完成，请重试。')
const character = ref(''), items = ref<Snapshot[]>([]), before = ref(''), after = ref(''), view = ref<View | null>(null), removal = ref('')
let resolvedID = ''
function payload() { if (!resolvedID) throw new Error('请先读取角色的历史。'); return { ...shell.chosen(), character_id: resolvedID } }
const comparer = useReader(compare, async stale => {
  if (!resolvedID || !before.value || !after.value || before.value === after.value) return
  const result = (await shell.invoke<{ view: View }>('panel.history.compare', { ...payload(), before_ref: before.value, after_ref: after.value })).view
  if (!stale()) view.value = result
})
watch([shell.selection, character], () => { comparer.cancel(); items.value = []; before.value = ''; after.value = ''; resolvedID = ''; view.value = null; removal.value = '' })
async function resolve() {
  let id = character.value.trim()
  if (!/^\d+$/.test(id)) {
    const result = await shell.invoke<{ entries: { id: string }[] }>('catalog.search', { query: id, kind: 'character' })
    if (result.entries.length !== 1) throw new Error('角色名未唯一匹配，请使用角色 ID。')
    id = result.entries[0]!.id
  }
  resolvedID = id
}
async function load() {
  items.value = (await shell.invoke<{ items: Snapshot[] }>('panel.history.list', payload())).items
  if (!items.value.some(item => item.ref === before.value)) before.value = items.value[1]?.ref ?? ''
  if (!items.value.some(item => item.ref === after.value)) after.value = items.value[0]?.ref ?? ''
  if (items.value.length >= 2) comparer.request()
}
async function list() { await run(async () => { await resolve(); await load(); view.value = null }) }
async function save() { await run(async () => { await resolve(); await shell.invoke('panel.history.save', payload()); await load() }, '已重新读取官方面板并保存快照。') }
async function get(ref: string) { await run(async () => { view.value = (await shell.invoke<{ view: View }>('panel.history.get', { ...payload(), ref })).view }) }
async function remove(ref: string) { await run(async () => { await shell.invoke('panel.history.remove', { ...payload(), ref }); removal.value = ''; view.value = null; await load() }, '历史快照已移除。') }
async function exportSnapshot(ref: string) { await run(async () => { const result = await shell.invoke<{ export: unknown }>('panel.history.export', { ...payload(), ref }); download(`panel-${resolvedID}-${ref}.json`, JSON.stringify(result.export, null, 2)) }) }
</script>

<template>
  <section class="section" aria-labelledby="history-title">
    <div class="section-heading"><div class="titles"><h2 id="history-title">面板历史</h2><p>主动保存官方面板，比较不同时间的属性、装备与技能；读取历史后自动对比最近两份快照。每个角色最多保留 20 份，读取仍需当前账号授权，不会写回游戏。</p></div></div>
    <form class="field-row" @submit.prevent="list">
      <fieldset :disabled="busy">
        <label class="field"><span>角色名称或 ID</span><input v-model="character" required placeholder="角色全名或 ID"></label>
        <div class="actions"><button type="submit" :disabled="!shell.selectedRole.value || !character.trim()">读取历史</button><button class="primary" type="button" :disabled="!shell.selectedRole.value || !character.trim()" @click="save">读取官方面板并保存</button></div>
      </fieldset>
    </form>
    <StatusLine v-if="comparer.loading.value" kind="info" text="正在对比面板…" />
    <StatusLine kind="error" :text="error || compare.error.value"><template v-if="comparer.failed.value" #actions><button class="compact" type="button" :disabled="busy || compare.busy.value" @click="comparer.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <form v-if="items.length >= 2" class="field-row" @submit.prevent="comparer.request()">
      <fieldset :disabled="busy">
        <label class="field"><span>对比前</span><select v-model="before" @change="comparer.request(250)"><option v-for="item in items" :key="item.ref" :value="item.ref">{{ formatTime(item.saved_at_ms) }} · {{ item.weapon }}</option></select></label>
        <label class="field"><span>对比后</span><select v-model="after" @change="comparer.request(250)"><option v-for="item in items" :key="item.ref" :value="item.ref">{{ formatTime(item.saved_at_ms) }} · {{ item.weapon }}</option></select></label>
        <button type="submit" :disabled="!before || !after || before === after">{{ comparer.loading.value ? '正在对比…' : '重新对比' }}</button>
      </fieldset>
    </form>
    <ul v-if="items.length" class="rows">
      <li v-for="item in items" :key="item.ref">
        <div class="row-main"><strong>{{ formatTime(item.saved_at_ms) }} · {{ item.name }}</strong><div class="row-meta"><span>等级 {{ item.level }}</span><span>{{ item.weapon }}</span><span>{{ item.version }}</span></div></div>
        <div class="row-actions"><button class="compact" :disabled="busy" @click="get(item.ref)">查看</button><button class="compact" :disabled="busy" @click="exportSnapshot(item.ref)">导出</button><button class="compact danger" :disabled="busy" @click="removal = item.ref">移除</button></div>
        <div v-if="removal === item.ref" class="confirm"><p>确认移除这份本地快照？</p><div class="actions"><button class="compact" :disabled="busy" @click="removal = ''">保留</button><button class="compact destructive" :disabled="busy" @click="remove(item.ref)">确认移除</button></div></div>
      </li>
    </ul>
    <EmptyState v-else-if="!busy" icon="history" title="还没有读取历史快照" text="输入角色后读取历史，或读取官方面板并保存第一份快照。" />
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
