<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { cloudTask } from './cloud'
import { taskOperation, useReader } from './reader'

// 攻略: posts of the reference miyoushe collections, read as a content task
// that is polled until it completes. The collection is read as soon as the
// source and collection are chosen; a keyword is searched on Enter.
interface Source { id: string; name: string; collections: string[] }
interface Post { id: string; title: string; url: string; parts?: string[]; images?: string[] }
const shell = useShell()
const { busy, error, notice, run } = useOperation('攻略设置操作未完成，请重试。')
const task = cloudTask(shell.invoke, 'content')
const sources = ref<Source[]>([]), source = ref(''), revision = ref(0), collection = ref(0), query = ref(''), active = ref(''), part = ref(0)
const selected = computed(() => sources.value.find(item => item.id === source.value))
const posts = computed(() => (task.job.value?.result?.items ?? []) as Post[])
const post = computed(() => posts.value.find(item => item.id === active.value))
const running = computed(() => task.job.value?.state === 'running')
watch(source, () => { collection.value = 0 })
async function load() {
  const loaded = await run(async () => {
    const result = await shell.invoke<{ sources: Source[]; settings: { default_source: string; revision: number } }>('guides.schema')
    sources.value = result.sources; source.value = result.settings.default_source; revision.value = result.settings.revision
  })
  if (loaded) search()
}
async function save() {
  await run(async () => {
    const result = await shell.invoke<{ settings: { revision: number } }>('guides.configure', { default_source: source.value, revision: revision.value })
    revision.value = result.settings.revision
  }, '默认攻略来源已保存。')
}
let nextPage = false
const reader = useReader(taskOperation(task), async () => {
  const next = nextPage
  nextPage = false
  if (!source.value) return
  const offset = next ? task.job.value?.result?.next_offset ?? 0 : 0
  active.value = ''; part.value = 0
  await task.start({ kind: 'guides', source: source.value, collection_offset: collection.value, query: query.value.trim(), offset })
})
function search(next = false, delay = 0) { nextPage = next; reader.request(delay) }
function open(item: Post) { active.value = item.id; part.value = 0 }
void load()
onUnmounted(task.dispose)
</script>

<template>
  <section class="section" aria-labelledby="guides-title">
    <div class="section-heading"><div class="titles"><h2 id="guides-title">攻略</h2><p>按固定参考的米游社合集查找攻略；选择来源与合集后自动读取，关键词按回车查询。攻略保留作者观点与原文，图片在官方原图链接中查看。</p></div><button type="button" :disabled="busy" @click="load">刷新设置</button></div>
    <form class="field-row" @submit.prevent="search()">
      <fieldset :disabled="busy">
        <label class="field"><span>来源</span><select v-model="source" @change="search(false, 250)"><option v-for="item in sources" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
        <label class="field"><span>合集</span><select v-model="collection" @change="search(false, 250)"><option v-for="(id, index) in selected?.collections" :key="id" :value="index">{{ index + 1 }} · {{ id }}</option></select></label>
        <label class="field"><span>角色名或标题关键词</span><input v-model="query" maxlength="100" placeholder="留空浏览本合集"></label>
        <div class="actions"><button type="submit" :disabled="task.busy.value || running">{{ task.busy.value || running ? '正在读取…' : '查询' }}</button><button type="button" :disabled="!source" @click="save">设为默认来源</button></div>
      </fieldset>
    </form>
    <StatusLine kind="error" :text="error || task.error.value || task.job.value?.message || ''"><template v-if="task.error.value || task.job.value?.state === 'failed'" #actions><button class="compact" type="button" :disabled="task.busy.value || running" @click="search()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <StatusLine v-if="reader.pending.value || task.busy.value || running" kind="info" text="正在读取攻略合集…"><template v-if="running" #actions><button type="button" class="compact" :disabled="task.busy.value" @click="task.cancel">取消</button><button type="button" class="compact" @click="task.poll">刷新进度</button></template></StatusLine>
    <template v-if="task.job.value?.state === 'completed'">
      <p class="hint">本合集匹配 {{ task.job.value.result?.total }} 篇。</p>
      <EmptyState v-if="!posts.length" icon="compass" title="没有匹配的攻略" text="换一个合集或关键词再查询。" />
      <ul v-else class="rows">
        <li v-for="item in posts" :key="item.id">
          <div class="row-main"><strong>{{ item.title }}</strong></div>
          <div class="row-actions"><button type="button" class="compact" :aria-current="item.id === active ? 'true' : undefined" @click="open(item)">查看正文</button><a :href="item.url" target="_blank" rel="noreferrer">官方原文</a></div>
        </li>
      </ul>
      <div v-if="Number(task.job.value.result?.next_offset) >= 0" class="actions"><button type="button" :disabled="task.busy.value" @click="search(true)">下一页攻略</button></div>
      <article v-if="post" class="panel">
        <header><h3>{{ post.title }}</h3></header>
        <p class="prose">{{ post.parts?.[part] || '以图片为主，请查看官方原图。' }}</p>
        <div v-if="(post.parts?.length ?? 0) > 1" class="actions"><button type="button" class="compact" :disabled="part <= 0" @click="part--">上一段</button><span class="muted">{{ part + 1 }} / {{ post.parts?.length }}</span><button type="button" class="compact" :disabled="part + 1 >= (post.parts?.length ?? 0)" @click="part++">下一段</button></div>
        <div v-if="post.images?.length" class="actions"><a v-for="(url, index) in post.images" :key="url" :href="url" target="_blank" rel="noreferrer">官方攻略原图 {{ index + 1 }}</a></div>
      </article>
    </template>
    <EmptyState v-else-if="!task.job.value && !task.busy.value && !reader.pending.value" icon="compass" title="选择来源与合集后自动读取" text="匹配的攻略会列在这里，点“查看正文”阅读原文。" />
  </section>
</template>
