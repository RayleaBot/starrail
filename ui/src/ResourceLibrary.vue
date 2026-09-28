<script setup lang="ts">
import { ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'

// 材料与卡池: the bundled reference snapshot of materials and banners, each
// list paged on its own offset.
interface Material { id: string; name: string; family: string; kind: string; rarity: number; sources: string[]; description: string }
interface Pool { estimated_start?: boolean; version: string; half: string; from: string; to: string; kind: string; characters5: string[]; characters4: string[]; weapons5: string[]; weapons4: string[] }
interface Appearance { name: string; kind: string; rarity: number; count: number; last_end: string; days_since_end: number | null }
const shell = useShell()
const { busy, error, run } = useOperation('资料读取未完成，请重试。')
const sourceVersion = ref('')
const query = ref(''), materials = ref<Material[]>([]), materialTotal = ref(0), materialOffset = ref(0), materialNext = ref<number | null>(null)
const version = ref(''), date = ref(''), bannerName = ref(''), bannerKind = ref('all'), pools = ref<Pool[]>([]), poolTotal = ref(0), poolOffset = ref(0), poolNext = ref<number | null>(null), appearances = ref<Appearance[]>([])
const kinds: Record<string, string> = { normal: '通用材料', talent: '天赋材料', weapon: '武器材料', weekly: '周本材料', gem: '突破宝石', boss: '首领材料', specialty: '地区特产' }
watch(query, () => { materials.value = []; materialTotal.value = 0; materialOffset.value = 0; materialNext.value = null })
watch([version, date, bannerName, bannerKind], () => { pools.value = []; poolTotal.value = 0; poolOffset.value = 0; poolNext.value = null; appearances.value = [] })
async function loadMaterials(next = 0) {
  const result = await shell.invoke<{ materials: Material[]; total: number; version: string; next_offset?: number | null }>('materials.query', { query: query.value, offset: next })
  materialOffset.value = next; materialNext.value = result.next_offset ?? null; materials.value = result.materials; materialTotal.value = result.total; sourceVersion.value = result.version
}
async function loadPools(next = 0) {
  const result = await shell.invoke<{ pools: Pool[]; total: number; version: string; next_offset?: number | null; appearances?: Appearance[] }>('banners.query', { version: version.value, date: date.value, query: bannerName.value, kind: bannerKind.value, offset: next })
  poolOffset.value = next; poolNext.value = result.next_offset ?? null; appearances.value = result.appearances ?? []; pools.value = result.pools; poolTotal.value = result.total; sourceVersion.value = result.version
}
void run(async () => { await loadMaterials(); await loadPools() })
</script>

<template>
  <section class="section" aria-labelledby="resources-title">
    <div class="section-heading"><div class="titles"><h2 id="resources-title">材料与卡池</h2><p>资料来自固定参考快照 {{ sourceVersion || '—' }}，可离线查询，不需要 CK。卡池与日期未作在线校准，不代表当前官方活动；日期沿用国服 UTC+8。</p></div></div>
    <StatusLine kind="error" :text="error" />
    <div class="subsection">
      <h3>材料查询</h3>
      <form class="field-row" @submit.prevent="run(() => loadMaterials())">
        <fieldset :disabled="busy">
          <label class="field"><span>材料名称或系列</span><input v-model="query" maxlength="128" type="search" placeholder="留空列出全部材料"></label>
          <button class="primary" type="submit">{{ busy ? '正在查询…' : '查询材料' }}</button>
        </fieldset>
      </form>
      <p class="hint">匹配 {{ materialTotal }} 项，当前页 {{ materials.length }} 项；可用关键词缩小范围。</p>
      <ul v-if="materials.length" class="rows">
        <li v-for="item in materials" :key="`${item.kind}:${item.id}:${item.name}`">
          <div class="row-main">
            <strong>{{ item.name }} · {{ item.rarity }} 星</strong>
            <div class="row-meta"><span>{{ kinds[item.kind] || '培养材料' }}</span><span>{{ item.family }}</span><span v-if="item.sources.length">来源：{{ item.sources.join('；') }}</span></div>
            <p v-if="item.description" class="hint">{{ item.description }}</p>
          </div>
        </li>
      </ul>
      <EmptyState v-else-if="!busy" icon="cube" title="没有匹配的材料" text="换一个关键词，或清空后查看全部材料。" />
      <div class="actions"><button type="button" :disabled="busy || materialOffset === 0" @click="run(() => loadMaterials(Math.max(0, materialOffset - 100)))">上一页材料</button><button type="button" :disabled="busy || materialNext === null" @click="run(() => loadMaterials(materialNext!))">下一页材料</button></div>
    </div>
    <div class="subsection">
      <h3>版本卡池</h3>
      <form class="field-row" @submit.prevent="run(() => loadPools())">
        <fieldset :disabled="busy">
          <label class="field"><span>游戏版本</span><input v-model="version" maxlength="32" placeholder="如 1.0；留空不限"></label>
          <label class="field"><span>角色或装备名称</span><input v-model="bannerName" maxlength="128" type="search"></label>
          <label class="field"><span>卡池类别</span><select v-model="bannerKind"><option value="all">全部</option><option value="character">角色</option><option value="weapon">装备</option></select></label>
          <label class="field"><span>卡池日期</span><input v-model="date" type="date"></label>
          <button class="primary" type="submit">{{ busy ? '正在查询…' : '查询卡池' }}</button>
        </fieldset>
      </form>
      <p class="hint">匹配 {{ poolTotal }} 期，当前页 {{ pools.length }} 期；日期留空时不限日期。</p>
      <details v-if="appearances.length">
        <summary>出现与复刻记录统计</summary>
        <p class="hint">按收录记录统计次数；距离最近收录结束的天数，不证明此后没有复刻。</p>
        <ul class="rows">
          <li v-for="item in appearances" :key="item.kind + item.name">
            <div class="row-main"><strong>{{ item.name }}</strong><div class="row-meta"><span>收录 {{ item.count }} 次</span><span>最近结束 {{ item.last_end }}</span><span v-if="item.days_since_end !== null">距今 {{ item.days_since_end }} 天</span></div></div>
          </li>
        </ul>
      </details>
      <ul v-if="pools.length" class="rows">
        <li v-for="(item, index) in pools" :key="`${item.version}:${item.from}:${index}`">
          <div class="row-main">
            <strong>{{ item.version }} · {{ item.half }}</strong>
            <div class="row-meta"><span>{{ item.from || '未注明开始时间' }} — {{ item.to || '未注明结束时间' }}</span><span v-if="item.estimated_start">开始时间按固定参考规则估算</span></div>
            <p v-if="item.characters5.length">五星角色：{{ item.characters5.join('、') }}</p>
            <p v-if="item.characters4.length">四星角色：{{ item.characters4.join('、') }}</p>
            <p v-if="item.weapons5.length">五星装备：{{ item.weapons5.join('、') }}</p>
            <p v-if="item.weapons4.length">四星装备：{{ item.weapons4.join('、') }}</p>
          </div>
        </li>
      </ul>
      <EmptyState v-else-if="!busy" icon="cube" title="没有匹配的卡池" text="放宽版本、名称或日期条件后再查询。" />
      <div class="actions"><button type="button" :disabled="busy || poolOffset === 0" @click="run(() => loadPools(Math.max(0, poolOffset - 50)))">上一页卡池</button><button type="button" :disabled="busy || poolNext === null" @click="run(() => loadPools(poolNext!))">下一页卡池</button></div>
    </div>
  </section>
</template>
