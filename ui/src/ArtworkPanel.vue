<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'

// 素材源: the upstream image sets, polled every 2 s while one downloads.
interface Source { id: string; name: string; state: 'missing' | 'downloading' | 'ready' | 'bundled' | 'on_demand'; commit?: string; files: number; bytes: number; downloaded_ms?: number; received?: number; error?: string }
const shell = useShell()
const { busy, error, notice, run } = useOperation('素材操作未完成，请重试。')
const sources = ref<Source[]>([]), removeID = ref(''), loaded = ref(false)
const downloading = computed(() => sources.value.some(source => source.state === 'downloading'))
const prefix = computed(() => shell.game.value?.prefix ?? '')
const states: Record<Source['state'], { label: string; chip: string }> = { missing: { label: '未下载', chip: '' }, downloading: { label: '下载中', chip: 'warning' }, ready: { label: '已下载', chip: 'success' }, bundled: { label: '随插件附带', chip: '' }, on_demand: { label: '出图时按需下载', chip: '' } }
let timer: ReturnType<typeof setTimeout> | undefined, disposed = false

function megabytes(bytes: number) { return (bytes / 1e6).toFixed(1) + ' MB' }
function describe(source: Source): string {
  if (source.state === 'downloading') return `已接收 ${megabytes(source.received ?? 0)}`
  if (source.state === 'missing') return ''
  if (source.state === 'on_demand') return `已缓存 ${source.files} 个文件（${megabytes(source.bytes)}）`
  const parts = [`${source.files} 个文件`, megabytes(source.bytes)]
  if (source.commit) parts.push('版本 ' + source.commit.slice(0, 7))
  if (source.downloaded_ms) parts.push('下载于 ' + formatTime(source.downloaded_ms))
  return parts.join(' · ')
}
function schedule() {
  clearTimeout(timer)
  if (!disposed && downloading.value) timer = setTimeout(() => void refresh(), 2000)
}
// refresh reads the status without the busy lock, so polling never disables
// the buttons.
async function refresh() {
  try {
    sources.value = (await shell.invoke<{ sources: Source[] }>('artwork.status')).sources
    loaded.value = true
  } catch (cause) {
    if (!disposed) error.value = cause instanceof Error && cause.message ? cause.message : '素材状态读取失败，请刷新。'
  } finally {
    schedule()
  }
}
async function act(action: 'artwork.download' | 'artwork.delete', id: string, success = '') {
  await run(async () => { sources.value = (await shell.invoke<{ sources: Source[] }>(action, { id })).sources; removeID.value = '' }, success)
  schedule()
}
void refresh()
onUnmounted(() => { disposed = true; clearTimeout(timer) })
</script>

<template>
  <section class="section" aria-labelledby="artwork-title">
    <div class="section-heading"><div class="titles"><h2 id="artwork-title">素材源</h2><p>图片面板、图鉴与照片使用这些上游图片素材。插件包已随附模板用到的图片；“更新”会从上游公开仓库下载新版本到插件数据目录，下载的文件优先使用，删除后恢复使用随附文件。图鉴库需下载后才可用，缺少素材时图片改用简化样式。超级管理员也可以在聊天中发送“{{ prefix }}素材更新”。</p></div><button type="button" :disabled="busy" @click="refresh">刷新状态</button></div>
    <StatusLine kind="error" :text="error" />
    <StatusLine kind="success" :text="notice" />
    <ul v-if="sources.length" class="rows">
      <li v-for="source in sources" :key="source.id">
        <div class="row-main">
          <strong>{{ source.name }}</strong>
          <div class="row-meta"><span class="chip" :class="states[source.state].chip">{{ states[source.state].label }}</span><span v-if="describe(source)" role="status">{{ describe(source) }}</span><span v-if="source.error" class="danger-text">上次下载失败：{{ source.error }}</span></div>
        </div>
        <div class="row-actions">
          <button v-if="source.state !== 'on_demand'" type="button" class="compact" :disabled="busy || source.state === 'downloading'" @click="act('artwork.download', source.id)">{{ source.state === 'downloading' ? '正在下载…' : source.state === 'ready' || source.state === 'bundled' ? '更新' : '下载' }}</button>
          <button v-if="source.state === 'ready' || (source.state === 'on_demand' && source.files > 0)" type="button" class="compact danger" :disabled="busy" @click="removeID = source.id">删除</button>
        </div>
        <div v-if="removeID === source.id" class="confirm">
          <p>确认删除已下载的{{ source.name }}？{{ source.state === 'on_demand' ? '之后出图时会重新按需下载。' : '之后恢复使用随附文件。' }}</p>
          <div class="actions"><button type="button" class="compact" :disabled="busy" @click="removeID = ''">保留</button><button type="button" class="compact destructive" :disabled="busy" @click="act('artwork.delete', source.id, '已删除下载的素材。')">确认删除</button></div>
        </div>
      </li>
    </ul>
    <EmptyState v-else-if="loaded" icon="image" title="没有可管理的素材源" text="插件没有声明上游素材。" />
  </section>
</template>
