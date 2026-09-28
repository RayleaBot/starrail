<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import StatusLine from '../shell/StatusLine.vue'
import EmptyState from '../shell/EmptyState.vue'
import { useShell, type AliasConflict } from '../shell/context'
import { useOperation } from '../shell/status'
import { useReader } from '../reader'
import BusinessView, { type View } from '../BusinessView.vue'
import ResourceLibrary from '../ResourceLibrary.vue'
import GuidesPanel from '../GuidesPanel.vue'
import PublicContent from '../PublicContent.vue'
import ArtworkPanel from '../ArtworkPanel.vue'
import MediaLibrary from '../MediaLibrary.vue'

const shell = useShell()
const sections: Section[] = [
  { id: 'catalog', label: '图鉴', icon: 'book' },
  { id: 'resources', label: '材料与卡池', icon: 'cube' },
  { id: 'guides', label: '攻略', icon: 'compass' },
  { id: 'content', label: '公告与活动', icon: 'megaphone' },
  { id: 'artwork', label: '素材源', icon: 'image' },
  { id: 'media', label: '图库', icon: 'images' },
]

// 图鉴: the bundled catalog, searched when the section first opens and again
// as the text or kind changes. The entry shown on the right is also the
// target of a new alias.
interface CatalogEntry { id: string; name: string; kind: string; rarity: number; element?: string; weapon?: string }
const catalog = useOperation('图鉴读取未完成，请重试。')
const search = ref(''), kind = ref(''), entries = ref<CatalogEntry[]>([]), selectedID = ref(''), view = ref<View | null>(null)
const catalogSection = ref<HTMLElement | null>(null)
let searched = false
const finder = useReader(catalog, async stale => {
  const result = await shell.invoke<{ entries: CatalogEntry[] }>('catalog.search', { query: search.value, kind: kind.value })
  if (!stale()) entries.value = result.entries
})
watch(catalogSection, element => { if (element && !searched) { searched = true; finder.request() } })
async function showEntry(entry: CatalogEntry) {
  await catalog.run(async () => { view.value = (await shell.invoke<{ view: View }>('catalog.get', { id: entry.id })).view; selectedID.value = entry.id })
}

// 自定义别名: saved through the shell, which validates them and reports the
// built-in names they shadow.
const aliasState = useOperation('别名未保存，请重试。')
const aliasName = ref(''), conflicts = ref<AliasConflict[]>([])
const aliases = computed(() => shell.settings.value.custom_aliases)
const target = computed(() => entries.value.find(entry => entry.id === selectedID.value)?.name ?? view.value?.title ?? selectedID.value)
async function saveAliases(next: Record<string, string>) {
  return await aliasState.run(async () => { conflicts.value = await shell.saveSettings({ custom_aliases: next }) }, '别名已保存。')
}
async function addAlias() {
  const name = aliasName.value.trim().toLowerCase()
  if (!name || !selectedID.value) return
  if (Object.hasOwn(aliases.value, name)) { aliasState.clear(); aliasState.error.value = '此别名已存在，请先移除再重新添加。'; return }
  if (await saveAliases({ ...aliases.value, [name]: selectedID.value })) aliasName.value = ''
}
async function removeAlias(name: string) { const copy = { ...aliases.value }; delete copy[name]; await saveAliases(copy) }
</script>

<template>
  <Workspace title="资料" description="本地图鉴、材料与卡池、攻略、官方公告与活动，以及出图素材与照片图库；这些内容不需要选择角色。" :sections="sections">
    <template #default="{ active }">
      <section v-if="active === 'catalog'" ref="catalogSection" class="section" aria-labelledby="catalog-title">
        <div class="section-heading"><div class="titles"><h2 id="catalog-title">图鉴</h2><p>本地资料 · {{ shell.catalogVersion.value }}。搜索角色或装备，输入或更改类型后自动搜索；选中后在右侧查看资料，并可为它添加常用别名。</p></div></div>
        <form class="field-row" @submit.prevent="finder.request()">
          <fieldset :disabled="catalog.busy.value && !finder.loading.value">
            <label class="field"><span>搜索角色、装备或 ID</span><input v-model="search" placeholder="名称或 ID，留空列出全部" @input="finder.request(500)"></label>
            <label class="field"><span>资料类型</span><select v-model="kind" @change="finder.request(250)"><option value="">全部类型</option><option value="character">角色</option><option value="weapon">装备</option></select></label>
            <button type="submit">{{ finder.loading.value ? '正在搜索…' : '重新搜索' }}</button>
          </fieldset>
        </form>
        <StatusLine v-if="finder.loading.value" kind="info" text="正在搜索图鉴…" />
        <StatusLine kind="error" :text="catalog.error.value"><template v-if="finder.failed.value" #actions><button class="compact" type="button" :disabled="catalog.busy.value" @click="finder.request()">重试</button></template></StatusLine>
        <div class="split">
          <div class="catalog-side">
            <ul v-if="entries.length" class="entry-list">
              <li v-for="entry in entries" :key="entry.id"><button type="button" :aria-current="entry.id === selectedID ? 'true' : undefined" :disabled="catalog.busy.value" @click="showEntry(entry)"><strong>{{ entry.name }}</strong><span>{{ entry.element || entry.weapon || entry.id }}</span></button></li>
            </ul>
            <EmptyState v-else-if="!finder.loading.value" icon="search" title="没有匹配资料" text="请尝试全名或 ID。" />
            <p v-if="entries.length === 50" class="hint">显示前 50 项，请使用关键词缩小范围。</p>
          </div>
          <BusinessView v-if="view" :value="view" />
          <EmptyState v-else icon="book" title="选择一项查看资料" text="左侧列表中的角色或装备资料会显示在这里。" />
        </div>
        <div class="subsection">
          <h3>自定义别名</h3>
          <p class="hint">{{ selectedID ? `为“${target}”添加常用称呼；` : '先在上方选择资料，再为它添加常用称呼；' }}自定义别名优先于内置名称。</p>
          <form class="field-row" @submit.prevent="addAlias">
            <fieldset :disabled="aliasState.busy.value">
              <label class="field"><span>别名</span><input v-model="aliasName" required placeholder="输入别名"></label>
              <button class="primary" type="submit" :disabled="!selectedID">{{ aliasState.busy.value ? '正在保存…' : '添加别名' }}</button>
            </fieldset>
          </form>
          <StatusLine kind="error" :text="aliasState.error.value" />
          <StatusLine kind="success" :text="aliasState.notice.value" />
          <StatusLine kind="attention" :text="conflicts.length ? `以下别名覆盖了内置名称：${conflicts.map(item => item.name).join('、')}。` : ''" />
          <ul v-if="Object.keys(aliases).length" class="rows">
            <li v-for="(id, name) in aliases" :key="name">
              <div class="row-main"><strong>{{ name }}</strong><div class="row-meta"><span>指向 {{ id }}</span></div></div>
              <div class="row-actions"><button type="button" class="compact danger" :disabled="aliasState.busy.value" :aria-label="`移除别名${name}`" @click="removeAlias(String(name))">移除</button></div>
            </li>
          </ul>
          <p v-else class="hint">还没有自定义别名。</p>
        </div>
      </section>
      <ResourceLibrary v-else-if="active === 'resources'" />
      <GuidesPanel v-else-if="active === 'guides'" />
      <PublicContent v-else-if="active === 'content'" />
      <ArtworkPanel v-else-if="active === 'artwork'" />
      <MediaLibrary v-else-if="active === 'media'" />
    </template>
  </Workspace>
</template>

<style scoped>
.catalog-side { display: grid; gap: 8px; }
</style>
