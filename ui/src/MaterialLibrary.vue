<script setup lang="ts">
import { ref } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { useReader } from './reader'

// 材料: the bundled material list, read as the search text changes.
interface Material { id: string; name: string; family: string; kind: string; rarity: number; sources: string[]; description: string }
const shell = useShell()
const operation = useOperation('材料读取未完成，请重试。')
const { busy, error } = operation
const sourceVersion = ref(''), query = ref(''), materials = ref<Material[]>([]), total = ref(0), offset = ref(0), next = ref<number | null>(null)
const kinds: Record<string, string> = { char: '角色晋阶材料', normal: '周本行迹材料', exp: '经验材料', material: '其他材料' }
let target = 0
const reader = useReader(operation, async stale => {
  const at = target
  const result = await shell.invoke<{ materials: Material[]; total: number; version: string; next_offset?: number | null }>('materials.query', { query: query.value, offset: at })
  if (stale()) return
  offset.value = at; next.value = result.next_offset ?? null; materials.value = result.materials; total.value = result.total; sourceVersion.value = result.version
})
function load(at = 0, delay = 0) { target = at; reader.request(delay) }
load()
</script>

<template>
  <section class="section" aria-labelledby="materials-title">
    <div class="section-heading"><div class="titles"><h2 id="materials-title">材料</h2><p>培养材料的稀有度、分类与获取途径，输入名称即筛选。资料来自固定参考快照 {{ sourceVersion || '—' }}，可离线查询，不需要 CK。</p></div></div>
    <form class="field-row" @submit.prevent="load()">
      <label class="field"><span>材料名称</span><input v-model="query" maxlength="128" type="search" placeholder="留空列出全部材料" @input="load(0, 400)"></label>
    </form>
    <StatusLine v-if="reader.loading.value" kind="info" text="正在查询材料…" />
    <StatusLine kind="error" :text="error"><template v-if="reader.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="reader.request()">重试</button></template></StatusLine>
    <p class="hint">匹配 {{ total }} 项，当前显示 {{ materials.length }} 项。</p>
    <ul v-if="materials.length" class="rows">
      <li v-for="item in materials" :key="`${item.kind}:${item.id}:${item.name}`">
        <div class="row-main">
          <strong>{{ item.name }} · {{ item.rarity }} 星</strong>
          <div class="row-meta"><span>{{ kinds[item.kind] || '培养材料' }}</span><span v-if="item.family && item.family !== item.name">{{ item.family }}</span><span v-if="item.sources.length">来源：{{ item.sources.join('；') }}</span></div>
          <p v-if="item.description" class="hint">{{ item.description }}</p>
        </div>
      </li>
    </ul>
    <EmptyState v-else-if="!reader.loading.value" icon="cube" title="没有匹配的材料" text="换一个关键词，或清空后查看全部材料。" />
    <div v-if="offset > 0 || next !== null" class="actions"><button type="button" :disabled="busy || offset === 0" @click="load(Math.max(0, offset - 100))">上一页</button><button type="button" :disabled="busy || next === null" @click="load(next!)">下一页</button></div>
  </section>
</template>
