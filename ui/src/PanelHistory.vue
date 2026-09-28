<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import CharacterSelect from './shell/CharacterSelect.vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { download, formatTime, useOperation } from './shell/status'
import { useReader } from './reader'

// 面板历史: saved snapshots of the selected role's characters, compared over
// time. A character's list is read as soon as it is picked and its two newest
// snapshots are compared by themselves; another pair compares on selection.
interface Snapshot { ref: string; saved_at_ms: number; version: string; name: string; level: number; weapon: string }
const shell = useShell()
const operation = useOperation('历史操作未完成，请重试。')
const { busy, error, notice, run } = operation
// The automatic comparison has its own lock so it neither blocks the form
// nor clears the notice of a save or removal that queued it.
const compare = useOperation('对比未完成，请重试。')
const character = ref(''), items = ref<Snapshot[]>([]), before = ref(''), after = ref(''), view = ref<View | null>(null), removal = ref('')
function payload() { if (!character.value) throw new Error('请先选择角色。'); return { ...shell.chosen(), character_id: character.value } }
const comparer = useReader(compare, async stale => {
  if (!character.value || !before.value || !after.value || before.value === after.value) return
  const result = (await shell.invoke<{ view: View }>('panel.history.compare', { ...payload(), before_ref: before.value, after_ref: after.value })).view
  if (!stale()) view.value = result
})
function accept(list: Snapshot[]) {
  items.value = list
  if (!items.value.some(item => item.ref === before.value)) before.value = items.value[1]?.ref ?? ''
  if (!items.value.some(item => item.ref === after.value)) after.value = items.value[0]?.ref ?? ''
  if (items.value.length >= 2) comparer.request()
}
async function load() { accept((await shell.invoke<{ items: Snapshot[] }>('panel.history.list', payload())).items) }
const reader = useReader(operation, async stale => {
  if (!shell.selectedRole.value || !character.value) return
  const list = (await shell.invoke<{ items: Snapshot[] }>('panel.history.list', payload())).items
  if (stale()) return
  view.value = null; accept(list)
})
// A character belongs to its role: a role change clears the choice, and the
// list of the new role picks its first character, whose history reads by itself.
watch(shell.selection, () => { character.value = '' })
watch(character, () => { comparer.cancel(); items.value = []; before.value = ''; after.value = ''; view.value = null; removal.value = ''; reader.request() })
async function save() { await run(async () => { await shell.invoke('panel.history.save', payload()); await load() }, '已重新读取官方面板并保存快照。') }
async function get(ref: string) { await run(async () => { view.value = (await shell.invoke<{ view: View }>('panel.history.get', { ...payload(), ref })).view }) }
async function remove(ref: string) { await run(async () => { await shell.invoke('panel.history.remove', { ...payload(), ref }); removal.value = ''; view.value = null; await load() }, '历史快照已移除。') }
async function exportSnapshot(ref: string) { await run(async () => { const result = await shell.invoke<{ export: unknown }>('panel.history.export', { ...payload(), ref }); download(`panel-${character.value}-${ref}.json`, JSON.stringify(result.export, null, 2)) }) }
</script>

<template>
  <section class="section" aria-labelledby="history-title">
    <div class="section-heading"><div class="titles"><h2 id="history-title">面板历史</h2><p>选择角色后自动列出它已保存的面板快照，并对比最近两份的属性、装备与技能；“读取官方面板并保存”新增一份快照。每个角色最多保留 20 份，读取仍需当前账号授权，不会写回游戏。</p></div></div>
    <form class="field-row" @submit.prevent="reader.request()">
      <fieldset :disabled="busy && !reader.loading.value">
        <CharacterSelect v-model="character" source="role" autoselect label="角色" />
        <div class="actions"><button type="submit" :disabled="busy || !character">{{ reader.loading.value ? '正在读取…' : '重新读取' }}</button><button class="primary" type="button" :disabled="busy || !character" @click="save">读取官方面板并保存</button></div>
      </fieldset>
    </form>
    <StatusLine v-if="reader.loading.value" kind="info" text="正在读取历史快照…" />
    <StatusLine v-else-if="comparer.loading.value" kind="info" text="正在对比面板…" />
    <StatusLine kind="error" :text="error || compare.error.value"><template v-if="reader.failed.value || comparer.failed.value" #actions><button class="compact" type="button" :disabled="busy || compare.busy.value" @click="reader.failed.value ? reader.request() : comparer.request()">重试</button></template></StatusLine>
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
    <EmptyState v-else-if="!busy" icon="history" title="还没有历史快照" text="选择角色后自动读取它已保存的快照；点击“读取官方面板并保存”保存第一份。" />
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
