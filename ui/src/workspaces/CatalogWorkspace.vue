<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import StatusLine from '../shell/StatusLine.vue'
import EmptyState from '../shell/EmptyState.vue'
import { useShell, type AliasConflict, type CatalogSummary } from '../shell/context'
import { useOperation } from '../shell/status'
import { useReader } from '../reader'
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

// 图鉴: the whole bundled catalog at once, narrowed by kind, element, path,
// rarity and a typed name; the first match opens by itself and the entry
// shown on the right is also the target of a new alias.
interface Talent { name: string; type?: string; tag?: string; level?: number; description: string[] }
interface Entry extends CatalogSummary { description?: string; stats?: Record<string, number>; talents?: Talent[]; cons?: Talent[] }
type Kind = 'character' | 'weapon'
const statLabels: [string, string][] = [['hp', '生命值'], ['atk', '攻击力'], ['def', '防御力'], ['speed', '速度'], ['cpct', '暴击率'], ['cdmg', '暴击伤害'], ['aggro', '嘲讽值'], ['sp', '能量上限']]
const catalog = useOperation('图鉴读取未完成，请重试。')
const kind = ref<Kind>('character'), search = ref(''), element = ref(''), path = ref(''), rarity = ref('')
const selectedID = ref(''), entry = ref<Entry | null>(null)
const catalogSection = ref<HTMLElement | null>(null)
watch(catalogSection, section => { if (section) void shell.loadCatalog() }, { immediate: true })
function distinct(values: (string | undefined)[]): string[] { return [...new Set(values.filter((value): value is string => !!value))] }
const ofKind = computed(() => shell.catalog.value.filter(item => item.kind === kind.value))
const elements = computed(() => distinct(ofKind.value.map(item => item.element)))
const paths = computed(() => distinct(ofKind.value.map(item => item.weapon)))
const rarities = computed(() => distinct(ofKind.value.map(item => item.rarity ? String(item.rarity) : undefined)).sort((a, b) => Number(b) - Number(a)))
function matches(item: CatalogSummary, query: string): boolean {
  if (!query) return true
  return [item.id, item.name, item.abbr ?? '', ...(item.aliases ?? [])].some(name => name.toLowerCase().includes(query))
}
const entries = computed(() => {
  const query = search.value.trim().toLowerCase()
  return ofKind.value
    .filter(item => (!element.value || item.element === element.value) && (!path.value || item.weapon === path.value) && (!rarity.value || String(item.rarity) === rarity.value) && matches(item, query))
    .sort((a, b) => b.rarity - a.rarity || a.name.localeCompare(b.name, 'zh-Hans-CN'))
})
watch(kind, () => { element.value = ''; path.value = ''; rarity.value = '' })
const reader = useReader(catalog, async stale => {
  if (!selectedID.value) return
  const result = await shell.invoke<{ entry: Entry }>('catalog.get', { id: selectedID.value })
  if (!stale()) entry.value = result.entry
})
function show(id: string) { if (id !== selectedID.value) { selectedID.value = id; entry.value = null; reader.request() } }
// The first match opens as the list narrows, so the page never waits for a click.
watch(entries, list => {
  if (!list.length) { selectedID.value = ''; entry.value = null; reader.cancel(); return }
  if (!list.some(item => item.id === selectedID.value)) show(list[0]!.id)
}, { immediate: true })
function statValue(key: string, value: number): string { return key === 'cpct' || key === 'cdmg' ? `${value}%` : String(value) }
const statRows = computed(() => statLabels.flatMap(([key, label]) => typeof entry.value?.stats?.[key] === 'number' ? [{ label, value: statValue(key, entry.value.stats[key]!) }] : []))
const statsLevel = computed(() => entry.value?.stats?.level ?? 0)
const skills = computed(() => (entry.value?.talents ?? []).filter(talent => talent.type !== '光锥技能'))
const coneSkill = computed(() => (entry.value?.talents ?? []).find(talent => talent.type === '光锥技能'))
function levelNote(talent: Talent): string { return talent.level ? `${talent.level} 级数值` : '' }

// 自定义别名: saved through the shell, which validates them and reports the
// built-in names they shadow.
const aliasState = useOperation('别名未保存，请重试。')
const aliasName = ref(''), conflicts = ref<AliasConflict[]>([])
const aliases = computed(() => shell.settings.value.custom_aliases)
const target = computed(() => entry.value?.name ?? entries.value.find(item => item.id === selectedID.value)?.name ?? '')
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
function aliasTarget(id: string): string { return shell.catalog.value.find(item => item.id === id)?.name ?? id }
</script>

<template>
  <Workspace title="资料" description="本地图鉴、材料与卡池、攻略、官方公告与活动，以及出图素材与照片图库；这些内容不需要选择角色。" :sections="sections">
    <template #default="{ active }">
      <section v-if="active === 'catalog'" ref="catalogSection" class="section" aria-labelledby="catalog-title">
        <div class="section-heading"><div class="titles"><h2 id="catalog-title">图鉴</h2><p>本地资料 · {{ shell.catalogVersion.value }}。全部角色与光锥都在列表里，按属性、命途、星级筛选或输入名称即可缩小范围；第一项会自动打开，选中的条目也可以添加常用别名。</p></div></div>
        <form class="field-row" @submit.prevent>
          <label class="field"><span>类型</span><select v-model="kind"><option value="character">角色</option><option value="weapon">光锥</option></select></label>
          <label class="field"><span>名称或别名</span><input v-model="search" placeholder="输入即筛选，留空列出全部"></label>
          <label v-if="elements.length" class="field"><span>属性</span><select v-model="element"><option value="">全部属性</option><option v-for="item in elements" :key="item" :value="item">{{ item }}</option></select></label>
          <label v-if="paths.length" class="field"><span>命途</span><select v-model="path"><option value="">全部命途</option><option v-for="item in paths" :key="item" :value="item">{{ item }}</option></select></label>
          <label v-if="rarities.length" class="field"><span>星级</span><select v-model="rarity"><option value="">全部星级</option><option v-for="item in rarities" :key="item" :value="item">{{ item }} 星</option></select></label>
        </form>
        <StatusLine v-if="shell.catalogState.value === 'loading'" kind="info" text="正在读取图鉴…" />
        <StatusLine kind="error" :text="shell.catalogState.value === 'failed' ? '图鉴清单读取失败。' : catalog.error.value"><template v-if="shell.catalogState.value === 'failed' || reader.failed.value" #actions><button class="compact" type="button" :disabled="catalog.busy.value" @click="shell.catalogState.value === 'failed' ? shell.loadCatalog() : reader.request()">重试</button></template></StatusLine>
        <div class="split even">
          <div class="catalog-side">
            <p class="hint" role="status">{{ entries.length }} 项</p>
            <div v-if="entries.length" class="tiles" role="list">
              <button v-for="item in entries" :key="item.id" type="button" class="tile" role="listitem" :aria-current="item.id === selectedID ? 'true' : undefined" @click="show(item.id)">
                <strong>{{ item.name }}</strong>
                <small><span :class="{ rare: item.rarity >= 5 }">{{ item.rarity }} 星</span>{{ item.element ? ` · ${item.element}` : '' }}{{ item.weapon ? ` · ${item.weapon}` : '' }}</small>
              </button>
            </div>
            <EmptyState v-else-if="shell.catalogState.value === 'ready'" icon="search" title="没有匹配的条目" text="换一个名称，或放宽属性、命途与星级的筛选。" />
          </div>
          <article v-if="entry" class="panel entry">
            <header class="entry-header">
              <h2>{{ entry.name }}</h2>
              <div class="chips">
                <span v-if="entry.rarity" class="chip plain gold">{{ entry.rarity }} 星</span>
                <span v-if="entry.element" class="chip plain">{{ entry.element }}</span>
                <span v-if="entry.weapon" class="chip plain">{{ entry.weapon }}</span>
                <span class="chip plain">{{ entry.kind === 'character' ? '角色' : '光锥' }}</span>
              </div>
              <p v-if="entry.aliases?.length" class="hint">别名：{{ entry.aliases.join('、') }}</p>
            </header>
            <p v-if="entry.description" class="prose">{{ entry.description }}</p>
            <section v-if="statRows.length" class="result-section">
              <h3>基础属性{{ statsLevel ? `（${statsLevel} 级）` : '' }}</h3>
              <dl class="facts"><div v-for="row in statRows" :key="row.label"><dt>{{ row.label }}</dt><dd>{{ row.value }}</dd></div></dl>
            </section>
            <section v-if="coneSkill" class="result-section">
              <h3>光锥技能 · {{ coneSkill.name }}</h3>
              <p class="hint">数值为叠影 1 到 5 的区间。</p>
              <p class="prose">{{ coneSkill.description.join('\n') }}</p>
            </section>
            <section v-if="skills.length" class="result-section">
              <h3>技能</h3>
              <ul class="skills">
                <li v-for="talent in skills" :key="talent.name">
                  <div class="skill-head"><strong>{{ talent.name }}</strong><span v-if="talent.type" class="chip plain">{{ talent.type }}</span><span v-if="talent.tag" class="chip plain">{{ talent.tag }}</span><small v-if="levelNote(talent)">{{ levelNote(talent) }}</small></div>
                  <p class="prose">{{ talent.description.join('\n') }}</p>
                </li>
              </ul>
            </section>
            <section v-if="entry.cons?.length" class="result-section">
              <h3>星魂</h3>
              <ol class="skills">
                <li v-for="(con, index) in entry.cons" :key="con.name">
                  <div class="skill-head"><strong>{{ index + 1 }} · {{ con.name }}</strong></div>
                  <p class="prose">{{ con.description.join('\n') }}</p>
                </li>
              </ol>
            </section>
          </article>
          <div v-else-if="selectedID && (catalog.busy.value || reader.pending.value)" class="skeleton" aria-hidden="true"><span></span><span></span><span></span></div>
          <EmptyState v-else icon="book" title="选择一项查看资料" text="左侧列表中的角色或光锥资料会显示在这里。" />
        </div>
        <div class="subsection">
          <h3>自定义别名</h3>
          <p class="hint">{{ selectedID ? `为“${target}”添加常用称呼；` : '先在上方选择条目，再为它添加常用称呼；' }}自定义别名优先于内置名称，聊天与查询都能使用。</p>
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
              <div class="row-main"><strong>{{ name }}</strong><div class="row-meta"><span>指向 {{ aliasTarget(id) }}</span></div></div>
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
.catalog-side { display: grid; gap: 8px; align-content: start; }
</style>
