<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import StatusLine from '../shell/StatusLine.vue'
import EmptyState from '../shell/EmptyState.vue'
import { overseas, regionName, regions, useShell } from '../shell/context'
import { download, useOperation } from '../shell/status'
import BusinessView, { type View } from '../BusinessView.vue'
import GachaHistory from '../GachaHistory.vue'
import GachaTasks from '../GachaTasks.vue'
import GachaSimulation from '../GachaSimulation.vue'
import { exportUIGF, parseImport, type Archive, type GachaRecord } from '../uigf'
import { driveSync, type SyncInfo } from '../sync'
import { useReader } from '../reader'

interface SavedArchive { uid: string; region: string; timezone: number; lang: string; records: number; revision: number }
interface Transfer { ref: string; uid: string; region: string; timezone: number; lang: string; total: number }

const shell = useShell()
let disposed = false

// 档案: the archives this plugin keeps, each with its summary, details and
// export. The archive of the selected role opens by itself; any other one
// opens from its row.
const archive = useOperation('档案操作未完成，请重试。')
const archives = ref<SavedArchive[]>([]), view = ref<View | null>(null)
const historyArchive = ref<SavedArchive | null>(null), historyKey = ref(0)
const removeCandidate = ref<SavedArchive | null>(null), removeDialog = ref<HTMLDialogElement | null>(null)
const sections = computed<Section[]>(() => [
  { id: 'archives', label: '全局档案', icon: 'cards', count: archives.value.length },
  { id: 'sync', label: '官方同步', icon: 'sync' },
  { id: 'import', label: '全局导入', icon: 'upload' },
  { id: 'simulation', label: '全局模拟', icon: 'dice' },
])
async function loadArchives() { archives.value = (await shell.invoke<{ items: SavedArchive[] }>('gacha.list')).items }
let summaryTarget: SavedArchive | null = null
const summary = useReader(archive, async stale => {
  const item = summaryTarget
  if (!item) return
  const result = (await shell.invoke<{ view: View }>('gacha.summary', { uid: item.uid, region: item.region })).view
  if (stale()) return
  view.value = result; historyArchive.value = item; historyKey.value++
})
function showSummary(item: SavedArchive) { summaryTarget = item; summary.request() }
function same(a: SavedArchive | null, b: SavedArchive | undefined): boolean { return !!a && !!b && a.uid === b.uid && a.region === b.region }
function matching(): SavedArchive | undefined {
  const role = shell.selectedRole.value?.role
  return role ? archives.value.find(item => item.uid === role.uid && item.region === role.region) : undefined
}
// autoShow opens the selected role's archive; with no role yet, the first one.
function autoShow(fallback: boolean) {
  const item = matching() ?? (fallback ? archives.value[0] : undefined)
  if (item && !same(summaryTarget, item)) showSummary(item)
}
watch(shell.selection, () => autoShow(false))
async function exportArchive(item: SavedArchive) {
  await archive.run(async () => {
    const { transfer } = await shell.invoke<{ transfer: Transfer }>('gacha.export.start', { uid: item.uid, region: item.region })
    const records: GachaRecord[] = []
    try {
      let more = true
      while (more) {
        if (disposed) throw new Error('导出已中止。')
        const chunk = await shell.invoke<{ records: GachaRecord[]; more: boolean }>('gacha.export.read', { ref: transfer.ref, offset: records.length })
        if (chunk.more && !chunk.records.length) throw new Error('导出没有继续返回记录，请重试。')
        records.push(...chunk.records); more = chunk.more
      }
      const content = exportUIGF({ uid: transfer.uid, timezone: transfer.timezone, lang: transfer.lang, list: records })
      download(`${shell.game.value!.id}-${item.uid}-UIGF.json`, JSON.stringify(content, null, 2))
    } finally { await shell.invoke('gacha.export.close', { ref: transfer.ref }).catch(() => undefined) }
  }, 'UIGF 导出文件已生成。')
}
function askRemove(item: SavedArchive) { removeCandidate.value = item; archive.clear(); removeDialog.value?.showModal() }
async function removeArchive() {
  if (!removeCandidate.value) return
  await archive.run(async () => {
    await shell.invoke('gacha.remove', { uid: removeCandidate.value!.uid, region: removeCandidate.value!.region })
    await loadArchives(); view.value = null; historyArchive.value = null
    removeDialog.value?.close(); removeCandidate.value = null
  }, '抽卡档案已移除。')
}
void archive.run(loadArchives).then(loaded => { if (loaded) autoShow(true) })

// 官方同步: the live sync of the selected role, driven page by page while the
// workspace stays open; only HoYoLab-authorised overseas roles can sync.
const sync = useOperation('同步未完成，请重试。')
const syncInfo = ref<SyncInfo | null>(null), fullSync = ref(false)
let syncController: AbortController | undefined
const syncAvailable = computed(() => !!shell.selectedRole.value && overseas(shell.selectedRole.value.role.region))
const syncState = computed(() => {
  const info = syncInfo.value
  if (!info) return ''
  if (info.state === 'running') return sync.busy.value ? '正在同步' : '同步已暂停，可继续或取消'
  return info.state === 'completed' ? '已完成' : '已取消'
})
watch(shell.selection, () => { if (syncInfo.value?.state !== 'running') { syncInfo.value = null; sync.clear() } })
async function synchronize(resume = false) {
  await sync.run(async () => {
    if (!resume || !syncInfo.value) syncInfo.value = (await shell.invoke<{ sync: SyncInfo }>('gacha.sync.start', { ...shell.chosen(), full: fullSync.value })).sync
    syncController = new AbortController()
    if (disposed) syncController.abort()
    const completed = await driveSync(shell.invoke, syncInfo.value!, syncController.signal, value => { syncInfo.value = value })
    if (completed.state === 'completed' && completed.result) {
      sync.notice.value = `同步完成，新增 ${completed.result.added} 条，档案共 ${completed.result.total} 条。`
      await loadArchives()
    } else sync.notice.value = '同步已取消，原有档案保持不变。'
  })
  syncController = undefined
}
async function cancelSync() {
  if (syncController) { syncController.abort(); return }
  if (syncInfo.value?.state === 'running') await sync.run(async () => {
    await shell.invoke('gacha.sync.cancel', { ref: syncInfo.value!.ref })
    syncInfo.value = { ...syncInfo.value!, state: 'canceled' }
  }, '同步已取消，原有档案保持不变。')
}

// 导入记录: a UIGF/SRGF file parsed in the browser, appended in chunks of 300.
const importer = useOperation('导入未完成，请重试。')
const imported = ref<Archive[]>([]), importIndex = ref(0), importRegion = ref(shell.game.value?.region ?? 'prod_gf_cn'), legacyTimezone = ref(8)
const importFile = ref<HTMLInputElement | null>(null)
const selectedImport = computed(() => imported.value[importIndex.value])
async function readImport(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  await importer.run(async () => {
    if (file.size > 64 * 1024 * 1024) throw new Error('文件超过 64 MiB，请拆分后导入。')
    const parsed = parseImport(await file.text(), legacyTimezone.value)
    if (!parsed.length) throw new Error('文件中没有本游戏的抽卡档案。')
    imported.value = parsed; importIndex.value = 0
  })
  if (importFile.value) importFile.value.value = ''
}
async function importArchive() {
  if (!selectedImport.value) return
  await importer.run(async () => {
    const item = selectedImport.value!
    const { transfer } = await shell.invoke<{ transfer: Transfer }>('gacha.import.start', { uid: item.uid, region: importRegion.value, timezone: item.timezone, lang: item.lang })
    try {
      for (let offset = 0; offset < item.list.length; offset += 300) {
        if (disposed) throw new Error('导入已中止。')
        await shell.invoke('gacha.import.append', { ref: transfer.ref, offset, records: item.list.slice(offset, offset + 300) })
        importer.notice.value = `正在导入 ${Math.min(offset + 300, item.list.length)} / ${item.list.length} 条记录…`
      }
      const result = await shell.invoke<{ added: number; total: number }>('gacha.import.finish', { ref: transfer.ref })
      importer.notice.value = `已导入 ${result.added} 条新记录，档案共 ${result.total} 条。`
      imported.value = []; await loadArchives()
    } catch (cause) { await shell.invoke('gacha.import.cancel', { ref: transfer.ref }).catch(() => undefined); throw cause }
  })
}

onUnmounted(() => { disposed = true; syncController?.abort() })
</script>

<template>
  <Workspace title="抽卡" description="跃迁档案的同步、导入、统计与导出；官方同步和后台任务使用上方选择的角色。" :sections="sections" roles>
    <template #default="{ active, select }">
      <StatusLine v-if="active !== 'sync'" kind="info" text="全局区域：抽卡档案按 UID 共享，模拟额度与管理端模拟由所有机器人共用。" />
      <section v-if="active === 'archives'" class="section" aria-labelledby="archives-title">
        <div class="section-heading"><div class="titles"><h2 id="archives-title">档案</h2><p>本插件保存的跃迁记录，按 UID 与区服分档；上方角色对应的档案会自动显示统计与明细，其他档案可从行内打开，或导出为 UIGF 文件。</p></div><button type="button" :disabled="archive.busy.value" @click="archive.run(loadArchives)">刷新</button></div>
        <StatusLine v-if="archive.busy.value && !archives.length" kind="info" text="正在读取档案…" />
        <StatusLine v-else-if="summary.loading.value" kind="info" text="正在读取统计…" />
        <StatusLine kind="error" :text="archive.error.value"><template v-if="summary.failed.value" #actions><button class="compact" type="button" :disabled="archive.busy.value" @click="summary.request()">重试</button></template></StatusLine>
        <StatusLine kind="success" :text="archive.notice.value" />
        <ul v-if="archives.length" class="rows">
          <li v-for="item in archives" :key="`${item.uid}:${item.region}`">
            <div class="row-main"><strong class="mono">{{ item.uid }}</strong><div class="row-meta"><span>{{ regionName(item.region) }}</span><span>{{ item.records }} 条记录</span><span v-if="same(historyArchive, item)">当前显示</span></div></div>
            <div class="row-actions"><button class="compact" type="button" :disabled="archive.busy.value" :aria-current="same(historyArchive, item) ? 'true' : undefined" @click="showSummary(item)">查看统计</button><button class="compact" type="button" :disabled="archive.busy.value" @click="exportArchive(item)">导出</button><button class="compact danger" type="button" :disabled="archive.busy.value" @click="askRemove(item)">移除</button></div>
          </li>
        </ul>
        <EmptyState v-else-if="!archive.busy.value" icon="cards" title="还没有抽卡档案" text="同步或导入后可查看统计并随时导出。"><button type="button" @click="select('sync')">官方同步</button><button type="button" @click="select('import')">导入记录</button></EmptyState>
        <BusinessView v-if="view" :value="view" />
        <GachaHistory v-if="historyArchive" :key="historyKey" :uid="historyArchive.uid" :region="historyArchive.region" :disabled="archive.busy.value" />
        <dialog ref="removeDialog" aria-labelledby="remove-title" @cancel="archive.busy.value && $event.preventDefault()">
          <h2 id="remove-title">移除抽卡档案 {{ removeCandidate?.uid }}</h2>
          <p>将删除所有机器人共享的此区服记录，并取消对应的全部后台同步、暂停每日同步。建议先导出保留副本。</p>
          <p v-if="archive.error.value" role="alert" class="danger-text">{{ archive.error.value }}</p>
          <div class="actions"><button type="button" autofocus :disabled="archive.busy.value" @click="removeDialog?.close()">取消</button><button type="button" class="destructive" :disabled="archive.busy.value" @click="removeArchive">{{ archive.busy.value ? '正在移除…' : '移除档案' }}</button></div>
        </dialog>
      </section>

      <section v-else-if="active === 'sync'" class="section" aria-labelledby="sync-title">
        <div class="section-heading"><div class="titles"><h2 id="sync-title">官方同步</h2><p>用上方角色的授权读取官方跃迁记录并合并进本地档案。即时同步需要保持页面打开；需要离开页面时，改用下方的后台任务。</p></div></div>
        <StatusLine kind="attention" text="星铁国服完整逐抽接口暂不接受现有授权，可导入 UIGF 或 SRGF。海外角色已接入 HoYoLab 授权同步；在账号页一次加密导入即可共用。" />
        <form class="field-grid" @submit.prevent="synchronize()">
          <fieldset :disabled="sync.busy.value || syncInfo?.state === 'running'">
            <label class="check wide"><input v-model="fullSync" type="checkbox">全量读取官方历史（默认只同步新增记录）</label>
          </fieldset>
          <div class="actions">
            <button v-if="syncInfo?.state !== 'running'" class="primary" type="submit" :disabled="sync.busy.value || !syncAvailable">{{ sync.busy.value ? '正在同步…' : '同步记录' }}</button>
            <button v-else-if="!sync.busy.value" class="primary" type="button" @click="synchronize(true)">继续同步</button>
            <button v-if="syncInfo?.state === 'running'" type="button" @click="cancelSync">取消同步</button>
            <span v-if="!syncAvailable" class="hint">{{ shell.selectedRole.value ? '当前角色属于国服，官方接口暂不接受现有授权，不能即时同步；请改用“导入记录”。' : '请先在上方选择已授权的角色；没有角色时可在“导入记录”导入已有记录。' }}</span>
          </div>
        </form>
        <StatusLine kind="error" :text="sync.error.value" />
        <StatusLine kind="success" :text="sync.notice.value" />
        <dl v-if="syncInfo" class="metrics" role="status"><div><dt>同步状态</dt><dd>{{ syncState }}</dd></div><div><dt>已读取页数</dt><dd>{{ syncInfo.pages }}</dd></div><div><dt>已读取记录</dt><dd>{{ syncInfo.fetched }}</dd></div></dl>
        <GachaTasks />
      </section>

      <section v-else-if="active === 'import'" class="section" aria-labelledby="import-title">
        <div class="section-heading"><div class="titles"><h2 id="import-title">导入记录</h2><p>支持 <a href="https://uigf.org/en/standards/uigf.html" target="_blank" rel="noreferrer">UIGF</a> v4、SRGF 和本插件导出的完整档案，不接收抽卡链接。文件在浏览器中解析，只有记录本身会提交给插件。</p></div></div>
        <div class="field-grid">
          <label class="field"><span>旧版文件缺省时区</span><select v-model="legacyTimezone" :disabled="importer.busy.value"><option :value="8">UTC+8（国服/亚服）</option><option :value="1">UTC+1（欧服）</option><option :value="-5">UTC−5（美服）</option></select><span class="hint">只在旧版 UIGF 或 SRGF 文件未写明时区时使用。</span></label>
          <label class="field"><span>选择 JSON 文件</span><input ref="importFile" type="file" accept=".json,application/json" :disabled="importer.busy.value" @change="readImport"></label>
        </div>
        <form v-if="imported.length" class="field-grid" @submit.prevent="importArchive">
          <fieldset :disabled="importer.busy.value">
            <label class="field"><span>文件中的账号</span><select v-model="importIndex"><option v-for="(item, index) in imported" :key="`${item.uid}:${index}`" :value="index">{{ item.uid }} · {{ item.list.length }} 条 · UTC{{ item.timezone >= 0 ? '+' : '' }}{{ item.timezone }}</option></select></label>
            <label class="field"><span>游戏区服</span><select v-model="importRegion"><option v-for="region in regions" :key="region.id" :value="region.id">{{ region.name }}</option></select></label>
            <div class="actions"><button class="primary" type="submit">{{ importer.busy.value ? '正在导入…' : '导入所选账号' }}</button></div>
          </fieldset>
        </form>
        <StatusLine kind="error" :text="importer.error.value" />
        <StatusLine :kind="importer.busy.value ? 'info' : 'success'" :text="importer.notice.value" />
      </section>

      <GachaSimulation v-else-if="active === 'simulation'" />
    </template>
  </Workspace>
</template>
