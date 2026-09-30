<script setup lang="ts">
import { computed, ref } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { useReader } from './reader'

// 卡池: the bundled banner history, newest first, narrowed by version, a
// featured name, kind and date as soon as one of them changes.
interface Pool { estimated_start?: boolean; version: string; half: string; from: string; to: string; kind: string; characters5: string[]; characters4: string[]; weapons5: string[]; weapons4: string[] }
interface Appearance { name: string; kind: string; rarity: number; count: number; last_end: string; days_since_end: number | null }
const shell = useShell()
const operation = useOperation('卡池读取未完成，请重试。')
const { busy, error } = operation
const sourceVersion = ref(''), versions = ref<string[]>([]), names = ref<string[]>([])
const version = ref(''), date = ref(''), name = ref(''), kind = ref('all')
const pools = ref<Pool[]>([]), total = ref(0), offset = ref(0), next = ref<number | null>(null), appearances = ref<Appearance[]>([])
let target = 0
const reader = useReader(operation, async stale => {
  const at = target
  const result = await shell.invoke<{ pools: Pool[]; total: number; version: string; next_offset?: number | null; appearances?: Appearance[]; versions?: string[]; names?: string[] }>('banners.query', { version: version.value, date: date.value, query: name.value, kind: kind.value, offset: at })
  if (stale()) return
  offset.value = at; next.value = result.next_offset ?? null; appearances.value = result.appearances ?? []; pools.value = result.pools; total.value = result.total; sourceVersion.value = result.version
  if (result.versions?.length) versions.value = result.versions
  if (result.names?.length) names.value = result.names
})
function load(at = 0, delay = 0) { target = at; reader.request(delay) }
const filtered = computed(() => !!(version.value || date.value || name.value || kind.value !== 'all'))
function reset() { version.value = ''; date.value = ''; name.value = ''; kind.value = 'all'; load() }
load()
</script>

<template>
  <section class="section" aria-labelledby="banners-title">
    <div class="section-heading"><div class="titles"><h2 id="banners-title">卡池</h2><p>历代跃迁卡池，最新的在前；选择版本、角色或光锥、卡池类别与日期后自动筛选。资料来自固定参考快照 {{ sourceVersion || '—' }}，未作在线校准，不代表当前官方活动；日期沿用国服 UTC+8。</p></div><button v-if="filtered" type="button" @click="reset">清除筛选</button></div>
    <form class="field-row" @submit.prevent="load()">
      <label class="field"><span>游戏版本</span><select v-model="version" @change="load()"><option value="">全部版本</option><option v-for="item in versions" :key="item" :value="item">{{ item }}</option></select></label>
      <label class="field"><span>角色或光锥</span><input v-model="name" maxlength="128" type="search" list="banner-names" placeholder="输入或从候选中选择" @input="load(0, 400)"><datalist id="banner-names"><option v-for="item in names" :key="item" :value="item" /></datalist></label>
      <label class="field"><span>卡池类别</span><select v-model="kind" @change="load()"><option value="all">全部</option><option value="character">角色</option><option value="weapon">光锥</option></select></label>
      <label class="field"><span>卡池日期</span><input v-model="date" type="date" @change="load()"></label>
    </form>
    <StatusLine v-if="reader.loading.value" kind="info" text="正在查询卡池…" />
    <StatusLine kind="error" :text="error"><template v-if="reader.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="reader.request()">重试</button></template></StatusLine>
    <p class="hint">匹配 {{ total }} 期，当前显示 {{ pools.length }} 期。</p>
    <ul v-if="pools.length" class="rows">
      <li v-for="(item, index) in pools" :key="`${item.version}:${item.from}:${index}`">
        <div class="row-main">
          <strong>{{ item.version }} · {{ item.half }}</strong>
          <div class="row-meta"><span>{{ item.from || '未注明开始时间' }} — {{ item.to || '未注明结束时间' }}</span><span v-if="item.estimated_start">开始时间按固定参考规则估算</span></div>
          <p v-if="item.characters5.length">五星角色：{{ item.characters5.join('、') }}</p>
          <p v-if="item.characters4.length">四星角色：{{ item.characters4.join('、') }}</p>
          <p v-if="item.weapons5.length">五星光锥：{{ item.weapons5.join('、') }}</p>
          <p v-if="item.weapons4.length">四星光锥：{{ item.weapons4.join('、') }}</p>
        </div>
      </li>
    </ul>
    <EmptyState v-else-if="!reader.loading.value" icon="cube" title="没有匹配的卡池" text="放宽版本、名称或日期条件，或清除筛选。" />
    <div v-if="offset > 0 || next !== null" class="actions"><button type="button" :disabled="busy || offset === 0" @click="load(Math.max(0, offset - 50))">较新的卡池</button><button type="button" :disabled="busy || next === null" @click="load(next!)">更早的卡池</button></div>
    <details v-if="appearances.length">
      <summary>出现与复刻记录统计</summary>
      <p class="hint">按收录记录统计次数；距离最近收录结束的天数，不证明此后没有复刻。</p>
      <ul class="rows">
        <li v-for="item in appearances" :key="item.kind + item.name">
          <div class="row-main"><strong>{{ item.name }}</strong><div class="row-meta"><span>{{ item.rarity }} 星{{ item.kind === 'weapon' ? '光锥' : '角色' }}</span><span>收录 {{ item.count }} 次</span><span>最近结束 {{ item.last_end }}</span><span v-if="item.days_since_end !== null">距今 {{ item.days_since_end }} 天</span></div></div>
        </li>
      </ul>
    </details>
  </section>
</template>
