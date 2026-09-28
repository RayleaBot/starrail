<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'
import { uploadMedia, readMedia, type MediaEntry } from './media'
import { useReader } from './reader'

// 图库: the operator's own character photos, uploaded and read in chunks;
// the list follows the search text, the linked character its own search.
// The preview is a data: URL because the host's CSP allows no blob: images.
const shell = useShell()
const operation = useOperation('图库操作未完成，请重试。')
const { busy, error, notice, run } = operation
const items = ref<MediaEntry[]>([]), total = ref(0), totalBytes = ref(0), next = ref<number | null>(null), query = ref('')
const catalogID = ref(''), source = ref(''), license = ref('本人有权使用'), rights = ref(false), files = ref<File[]>([]), progress = ref('')
const catalogQuery = ref(''), entries = ref<{ id: string; name: string }[]>([])
const preview = ref(''), selected = ref<MediaEntry | null>(null), editing = ref<MediaEntry | null>(null), removeRef = ref('')
const prefix = computed(() => shell.game.value?.prefix ?? '')
const fileName = computed(() => selected.value ? `${selected.value.title}.${selected.value.mime.split('/')[1]}` : '')
let controller: AbortController | undefined, disposed = false

async function operate(work: () => Promise<void>, success = '') {
  if (busy.value) return
  await run(work, success)
  progress.value = ''
}
async function list(offset = 0) {
  const result = await shell.invoke<{ items: MediaEntry[]; total: number; total_bytes: number; next_offset: number | null }>('media.list', { query: query.value, offset })
  items.value = offset ? [...items.value, ...result.items] : result.items
  total.value = result.total; totalBytes.value = result.total_bytes; next.value = result.next_offset
}
function choose(event: Event) {
  files.value = Array.from((event.target as HTMLInputElement).files ?? []).filter(file => /\.(png|jpe?g|gif|webp)$/i.test(file.name))
  rights.value = false
}
async function upload() {
  if (!rights.value || !files.value.length) return
  await operate(async () => {
    const aborter = new AbortController()
    controller = aborter
    let done = 0
    const pending = [...files.value]
    for (const file of pending) {
      if (disposed || aborter.signal.aborted) break
      progress.value = `第 ${done + 1} / ${pending.length} 张：${file.name}`
      await uploadMedia(shell.invoke, file, { title: file.name.replace(/\.[^.]+$/, ''), category: 'photo', catalog_id: catalogID.value, source: source.value, license: license.value }, aborter.signal, bytes => { progress.value = `第 ${done + 1} / ${pending.length} 张 · ${Math.round(bytes / file.size * 100)}%` })
      done++
      files.value = files.value.filter(item => item !== file)
    }
    if (!disposed) { notice.value = `已导入 ${done} 张。${files.value.length ? '剩余文件可以继续导入。' : ''}`; await list() }
  })
  controller = undefined
}
function toDataURL(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(reader.error ?? new Error('原图无法生成预览。'))
    reader.readAsDataURL(blob)
  })
}
function clearPreview() { preview.value = ''; selected.value = null; editing.value = null }
async function show(entry: MediaEntry) {
  await operate(async () => {
    clearPreview()
    const aborter = new AbortController()
    controller = aborter
    const result = await readMedia(shell.invoke, entry.ref, aborter.signal)
    if (disposed) return
    const url = await toDataURL(result.blob)
    if (disposed) return
    preview.value = url; selected.value = result.entry; editing.value = { ...result.entry }
  })
  controller = undefined
}
async function save() {
  if (!editing.value) return
  await operate(async () => {
    const result = await shell.invoke<{ entry: MediaEntry }>('media.update', editing.value as unknown as Record<string, unknown>)
    selected.value = result.entry; editing.value = { ...result.entry }
    await list()
  }, '素材信息已保存。')
}
async function remove(entry: MediaEntry) {
  await operate(async () => {
    await shell.invoke('media.remove', { ref: entry.ref, revision: entry.revision, confirm: true })
    if (selected.value?.ref === entry.ref) clearPreview()
    removeRef.value = ''
    await list()
  }, '素材已删除。')
}
const lister = useReader(operation, async stale => {
  const result = await shell.invoke<{ items: MediaEntry[]; total: number; total_bytes: number; next_offset: number | null }>('media.list', { query: query.value, offset: 0 })
  if (stale()) return
  items.value = result.items; total.value = result.total; totalBytes.value = result.total_bytes; next.value = result.next_offset
})
const finder = useReader(operation, async stale => {
  const result = await shell.invoke<{ entries: { id: string; name: string }[] }>('catalog.search', { query: catalogQuery.value, kind: 'character' })
  if (!stale()) entries.value = result.entries
})
lister.request()
onUnmounted(() => { disposed = true; controller?.abort(); clearPreview() })
</script>

<template>
  <section class="section" aria-labelledby="media-title">
    <div class="section-heading"><div class="titles"><h2 id="media-title">图库</h2><p>只保存你导入的角色照片，关联角色后供“{{ prefix }}角色名照片”与角色卡片使用。原图支持 PNG、JPEG、GIF、WebP，每张不超过 4 MiB、2400 万像素；全库最多 2000 项、512 MiB。</p></div></div>
    <dl class="metrics"><div><dt>匹配</dt><dd>{{ total }} 项</dd></div><div><dt>全库占用</dt><dd>{{ (totalBytes / 1024 / 1024).toFixed(1) }} MiB</dd></div></dl>
    <form class="field-row" @submit.prevent="lister.request()">
      <fieldset :disabled="busy && !lister.loading.value && !finder.loading.value">
        <label class="field"><span>搜索素材</span><input v-model="query" placeholder="标题或角色 ID，输入后自动查询" @input="lister.request(500)"></label>
        <button type="submit">{{ lister.loading.value ? '正在查询…' : '重新查询' }}</button>
      </fieldset>
    </form>
    <StatusLine v-if="lister.loading.value" kind="info" text="正在查询图库…" />
    <StatusLine kind="error" :text="error"><template v-if="lister.failed.value || finder.failed.value" #actions><button class="compact" type="button" :disabled="busy" @click="lister.failed.value ? lister.request() : finder.request()">重试</button></template></StatusLine>
    <StatusLine kind="success" :text="notice" />
    <StatusLine v-if="progress" kind="info" :text="progress"><template #actions><button type="button" class="compact" @click="controller?.abort()">停止上传</button></template></StatusLine>
    <ul v-if="items.length" class="rows">
      <li v-for="entry in items" :key="entry.ref">
        <div class="row-main"><strong>{{ entry.title }}</strong><div class="row-meta"><span>{{ entry.catalog_id || '未关联角色' }}</span><span>{{ entry.width }}×{{ entry.height }}</span><span>{{ (entry.bytes / 1024).toFixed(0) }} KiB</span></div></div>
        <div class="row-actions"><button type="button" class="compact" :aria-current="selected?.ref === entry.ref ? 'true' : undefined" :disabled="busy" @click="show(entry)">查看与编辑</button><button type="button" class="compact danger" :disabled="busy" @click="removeRef = entry.ref">删除</button></div>
        <div v-if="removeRef === entry.ref" class="confirm"><p>确认删除此素材原图？删除后无法恢复。</p><div class="actions"><button type="button" class="compact" :disabled="busy" @click="removeRef = ''">保留</button><button type="button" class="compact destructive" :disabled="busy" @click="remove(entry)">确认删除</button></div></div>
      </li>
    </ul>
    <EmptyState v-else-if="!lister.loading.value" icon="images" title="没有匹配的照片" text="导入图片后会列在这里；也可以换一个关键词查询。" />
    <div v-if="next !== null" class="actions"><button type="button" :disabled="busy" @click="operate(() => list(next!))">更多素材</button></div>
    <div class="subsection">
      <h3>导入图片或资源目录</h3>
      <form class="field-grid" @submit.prevent="upload">
        <fieldset :disabled="busy && !lister.loading.value && !finder.loading.value">
          <label class="field"><span>选择图片</span><input type="file" accept="image/png,image/jpeg,image/gif,image/webp" multiple @change="choose"></label>
          <label class="field"><span>选择目录</span><input type="file" webkitdirectory multiple @change="choose"></label>
          <label class="field"><span>来源或作者</span><input v-model="source" maxlength="512" placeholder="原作者或你自己的素材"></label>
          <label class="field"><span>使用许可</span><input v-model="license" maxlength="256" required></label>
          <div class="field-row wide">
            <label class="field"><span>查找关联角色</span><input v-model="catalogQuery" placeholder="角色名或 ID，输入后自动查找" @input="finder.request(500)" @keydown.enter.prevent="finder.request()"></label>
            <button type="button" @click="finder.request()">{{ finder.loading.value ? '正在查找…' : '重新查找' }}</button>
            <label class="field"><span>关联角色</span><select v-model="catalogID"><option value="">不关联</option><option v-for="entry in entries" :key="entry.id" :value="entry.id">{{ entry.name }} · {{ entry.id }}</option></select></label>
          </div>
          <p class="hint wide">已选 {{ files.length }} 张图片，非图片文件自动忽略。批量导入使用相同的关联角色，之后可逐项修改。</p>
          <label class="check wide"><input v-model="rights" type="checkbox">确认我有权在此机器人中使用这些素材</label>
          <div class="actions"><button class="primary" type="submit" :disabled="!rights || !files.length || !license.trim()">{{ busy && progress ? '正在导入…' : '导入所选图片' }}</button></div>
        </fieldset>
      </form>
    </div>
    <article v-if="selected && editing" class="panel">
      <div class="section-heading"><h3>{{ selected.title }}</h3><button type="button" class="compact" :disabled="busy" @click="clearPreview">关闭预览</button></div>
      <img class="preview" :src="preview" :alt="selected.title">
      <p><a :href="preview" :download="fileName">下载原图</a></p>
      <form class="field-grid" @submit.prevent="save">
        <fieldset :disabled="busy">
          <label class="field"><span>标题</span><input v-model="editing.title" maxlength="120" required></label>
          <label class="field"><span>关联角色 ID</span><input v-model="editing.catalog_id"></label>
          <label class="field"><span>来源</span><input v-model="editing.source" maxlength="512"></label>
          <label class="field"><span>使用许可</span><input v-model="editing.license" maxlength="256" required></label>
          <div class="actions"><button class="primary" type="submit">{{ busy ? '正在保存…' : '保存素材信息' }}</button></div>
        </fieldset>
      </form>
    </article>
  </section>
</template>

<style scoped>
.preview { max-width: 100%; max-height: 600px; object-fit: contain; justify-self: start; }
</style>
