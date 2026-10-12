<script setup lang="ts">
import { ref } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { useReader } from './reader'
import type { Scope } from './ChallengeRank.vue'

// 群设置: per-bot, per-group overrides of the game's settings and aliases,
// listed a page at a time, read when the section opens and on each turn. `rank` asks the workspace to open a group's
// challenge ranking; `listed` tells it which groups were seen.
interface Group { scope: Scope; revision: number; config: { enabled?: boolean; image_replies?: boolean; aliases?: Record<string, string> }; rank_entries: number; challenge_entries?: number }
const emit = defineEmits<{ rank: [scope: Scope]; listed: [scopes: Scope[]] }>()
const shell = useShell()
const operation = useOperation('群设置操作未完成，请重试。')
const { busy, error, notice, run } = operation
const groups = ref<Group[]>([]), page = ref(0), next = ref<number | null>(null)
const editing = ref<Group | null>(null), groupID = ref(''), enabled = ref('default'), images = ref('default'), aliases = ref<{ name: string; id: string }[]>([]), clearing = ref<Group | null>(null)
function scopeKey(scope: Scope) { return JSON.stringify([scope.source_protocol, scope.bot_id, scope.group_id]) }
function edit(item: Group) {
  editing.value = item
  enabled.value = item.config.enabled === undefined ? 'default' : String(item.config.enabled)
  images.value = item.config.image_replies === undefined ? 'default' : String(item.config.image_replies)
  aliases.value = Object.entries(item.config.aliases ?? {}).map(([name, id]) => ({ name, id }))
  clearing.value = null
}
async function load() {
  const result = await shell.invoke<{ items: Group[]; next_page: number | null }>('groups.list', { page: page.value })
  groups.value = result.items; next.value = result.next_page
  emit('listed', result.items.map(item => item.scope))
}
function create() {
  const chosen = shell.bot.value
  if (!chosen || !groupID.value.trim()) return
  edit({ scope: { source_protocol: chosen.source_protocol, bot_id: chosen.bot_id, group_id: groupID.value.trim() }, revision: 0, config: {}, rank_entries: 0 })
}
async function save() {
  if (!editing.value) return
  await run(async () => {
    const map: Record<string, string> = {}
    for (const row of aliases.value) {
      const key = row.name.trim().toLowerCase()
      if (!key || !row.id.trim() || Object.hasOwn(map, key)) throw new Error('别名与资料 ID 不可为空，别名不能重复。')
      map[key] = row.id.trim()
    }
    const config = { aliases: map, ...(enabled.value === 'default' ? {} : { enabled: enabled.value === 'true' }), ...(images.value === 'default' ? {} : { image_replies: images.value === 'true' }) }
    await shell.invoke('groups.set', { group_id: editing.value!.scope.group_id, revision: editing.value!.revision, config })
    editing.value = null
    await load()
  }, '群内游戏设置已保存。')
}
async function clear() {
  if (!clearing.value) return
  await run(async () => {
    await shell.invoke('groups.clear', { group_id: clearing.value!.scope.group_id, revision: clearing.value!.revision })
    clearing.value = null; editing.value = null
    await load()
  }, '已恢复默认设置并清空此群的本地参评记录。')
}
const lister = useReader(operation, load)
function turn(value: number) { page.value = value; lister.request() }
lister.request()
</script>

<template>
  <section class="section" aria-labelledby="groups-title">
    <div class="section-heading"><div class="titles"><h2 id="groups-title">群设置</h2><p>群内设置优先于本游戏的全局设置，按机器人和群分别保存。排行只收录用户主动提交的装备评分与挑战成绩；云端排名不在此管理。</p></div><button type="button" :disabled="busy" @click="lister.request()">{{ lister.loading.value ? '正在读取…' : '刷新列表' }}</button></div>
    <form class="field-row" @submit.prevent="create">
      <fieldset :disabled="busy && !lister.loading.value">
        <label class="field"><span>群 ID</span><input v-model="groupID" maxlength="256" required></label>
        <button type="submit" :disabled="!shell.bot.value || !groupID.trim()">编辑此群</button>
      </fieldset>
    </form>
    <StatusLine v-if="lister.loading.value && !groups.length" kind="info" text="正在读取群列表…" />
    <StatusLine kind="error" :text="error"><template v-if="lister.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="lister.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <form v-if="editing" class="subsection" @submit.prevent="save">
      <fieldset :disabled="busy">
        <h3>群 {{ editing.scope.group_id }}</h3>
        <div class="field-grid">
          <label class="field"><span>本群游戏功能</span><select v-model="enabled"><option value="default">默认开启</option><option value="true">开启</option><option value="false">关闭</option></select></label>
          <label class="field"><span>本群回复方式</span><select v-model="images"><option value="default">跟随全局</option><option value="true">图片优先</option><option value="false">文本</option></select></label>
        </div>
        <div v-for="(row, index) in aliases" :key="index" class="field-row">
          <label class="field"><span>群别名</span><input v-model="row.name" maxlength="64" required></label>
          <label class="field"><span>资料 ID</span><input v-model="row.id" required></label>
          <button type="button" @click="aliases.splice(index, 1)">移除此别名</button>
        </div>
        <div class="actions"><button type="button" :disabled="aliases.length >= 64" @click="aliases.push({ name: '', id: '' })">添加群别名</button><button class="primary" type="submit">{{ busy ? '正在保存…' : '保存群设置' }}</button><button type="button" @click="editing = null">取消编辑</button></div>
      </fieldset>
    </form>
    <div class="subsection">
      <h3>已保存的群</h3>
      <ul v-if="groups.length" class="rows">
        <li v-for="item in groups" :key="scopeKey(item.scope)">
          <div class="row-main">
            <strong>{{ item.scope.group_id }}</strong>
            <div class="row-meta"><span class="chip" :class="item.config.enabled === false ? 'danger' : 'success'">{{ item.config.enabled === false ? '游戏功能关闭' : '游戏功能开启' }}</span><span>{{ item.rank_entries }} 条面板排名</span><span>{{ item.challenge_entries ?? 0 }} 条挑战成绩</span><span>{{ Object.keys(item.config.aliases ?? {}).length }} 个群别名</span></div>
          </div>
          <div class="row-actions"><button type="button" class="compact" :disabled="busy" @click="edit(item)">编辑群设置</button><button type="button" class="compact" :disabled="busy" @click="emit('rank', item.scope)">查看挑战榜</button><button type="button" class="compact danger" :disabled="busy" @click="clearing = item">恢复默认并清空排行</button></div>
          <div v-if="clearing && scopeKey(clearing.scope) === scopeKey(item.scope)" class="confirm"><p>将恢复群 {{ item.scope.group_id }} 的默认设置，并删除全部主动提交的本地评分和挑战成绩。</p><div class="actions"><button type="button" class="compact" :disabled="busy" @click="clearing = null">保留群记录</button><button type="button" class="compact destructive" :disabled="busy" @click="clear">确认清空群记录</button></div></div>
        </li>
      </ul>
      <EmptyState v-else-if="!lister.loading.value" icon="people" title="当前页没有群设置" text="输入群 ID 添加，或翻到其他页。" />
      <div class="actions"><button type="button" :disabled="busy || page === 0" @click="turn(page - 1)">上一页</button><span class="muted">第 {{ page + 1 }} 页</span><button type="button" :disabled="busy || next === null" @click="turn(next!)">下一页</button></div>
    </div>
  </section>
</template>
