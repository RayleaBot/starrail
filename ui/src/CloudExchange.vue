<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { formatTime, useOperation } from './shell/status'
import { sanitizeExchangeFile, downloadExchange } from './cloud-exchange'
import { cloudTask } from './cloud'

// 交换档案: the miao PlayerData copies kept for the selected role's UID, built
// from the official panel or a file, and exchanged with ark for ten minutes.
interface Summary { uid: string; revision: number; saved_ms: number; items: { id: string; name: string; level?: number; cons?: number }[] }
const authenticated = defineModel<boolean>('authenticated', { required: true })
const shell = useShell()
const { busy, error, notice, run } = useOperation('面板交换操作未完成。')
const captureCharacter = ref(''), rankCharacter = ref(''), rankQuery = ref('all')
const summary = ref<Summary | null>(null), view = ref<View | null>(null), consent = ref(false), clearing = ref(false), file = ref<File | null>(null), stopping = ref(false), activeTransfer = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const transfer = cloudTask(shell.invoke)
let disposed = false
function fileChange(event: Event) { file.value = (event.target as HTMLInputElement).files?.[0] ?? null; notice.value = '' }
watch(shell.selection, () => { summary.value = null; view.value = null; clearing.value = false; file.value = null; if (fileInput.value) fileInput.value.value = ''; transfer.clear(); consent.value = false })
watch(authenticated, () => { transfer.clear(); consent.value = false })
async function load() {
  const result = await shell.invoke<Summary>('cloud.archive.list', shell.chosen())
  if (!disposed) { summary.value = result; if (!result.items.some(v => v.id === rankCharacter.value)) rankCharacter.value = result.items[0]?.id ?? '' }
}
async function readArchive() { await run(load) }
async function open(id: string) { await run(async () => { const result = await shell.invoke<{ panels: { id: string; name: string; view: View }[] }>('cloud.archive.get', { ...shell.chosen(), character_id: id }); view.value = result.panels[0]?.view ?? null }) }
async function capture() { await run(async () => { const result = await shell.invoke<{ note: string }>('cloud.archive.capture', { ...shell.chosen(), character_id: captureCharacter.value.trim() }); notice.value = result.note; await load() }) }
async function importFile() {
  const current = shell.selectedRole.value
  if (!file.value || !current) return
  stopping.value = false
  await run(async () => {
    if (file.value!.size > 8 * 1024 * 1024) throw new Error('面板文件不能超过 8 MiB。')
    const avatars = sanitizeExchangeFile(JSON.parse(await file.value!.text()), current.role.uid)
    const started = await shell.invoke<{ ref: string }>('cloud.archive.import.start', { ...shell.chosen(), uid: current.role.uid, count: avatars.length })
    activeTransfer.value = started.ref
    try {
      for (let i = 0; i < avatars.length; i++) {
        if (stopping.value || disposed) throw new Error('导入已取消，原档案保留。')
        await shell.invoke('cloud.archive.import.append', { ref: started.ref, offset: i, avatar: avatars[i] })
      }
      await shell.invoke('cloud.archive.import.finish', { ref: started.ref })
      notice.value = `已合并 ${avatars.length} 个导入角色。`
      await load()
    } finally { activeTransfer.value = ''; await shell.invoke('cloud.archive.import.cancel', { ref: started.ref }).catch(() => {}) }
  })
}
async function exportFile() {
  if (!shell.selectedRole.value) return
  stopping.value = false
  await run(async () => {
    const started = await shell.invoke<{ ref: string; uid: string; total: number }>('cloud.archive.export.start', shell.chosen())
    activeTransfer.value = started.ref
    try {
      const avatars: Record<string, unknown> = {}
      for (let i = 0; i < started.total; i++) {
        if (stopping.value || disposed) throw new Error('导出已取消。')
        const result = await shell.invoke<{ avatar: Record<string, unknown> }>('cloud.archive.export.read', { ref: started.ref, offset: i })
        avatars[String(result.avatar.id)] = result.avatar
      }
      downloadExchange(started.uid, avatars)
      notice.value = '已生成无凭据的交换 JSON。'
    } finally { activeTransfer.value = ''; await shell.invoke('cloud.archive.export.close', { ref: started.ref }).catch(() => {}) }
  })
}
async function remove() { await run(async () => { await shell.invoke('cloud.archive.remove', { ...shell.chosen(), revision: summary.value?.revision, confirm: true }); view.value = null; clearing.value = false; await load(); notice.value = '本地交换档案已清空。' }) }
async function upload() { if (!consent.value) return; await transfer.start({ consent: true, authenticated: authenticated.value, ...shell.chosen() }, 'cloud.archive.upload') }
async function download() { if (!consent.value || !shell.selectedRole.value) return; await transfer.start({ mode: 'exchange_download', uid: shell.selectedRole.value.role.uid, consent: true, authenticated: authenticated.value }) }
async function rank() { if (!consent.value || !rankCharacter.value) return; await transfer.start({ ...shell.chosen(), character_id: rankCharacter.value, query: rankQuery.value, consent: true, authenticated: authenticated.value }, 'cloud.archive.rank') }
async function refresh() { if (!consent.value) return; await transfer.start({ ...shell.chosen(), consent: true, authenticated: authenticated.value }, 'cloud.archive.refresh') }
async function receive() { await run(async () => { if (!summary.value) await load(); await shell.invoke('cloud.archive.receive', { ...shell.chosen(), ref: transfer.job.value?.ref, revision: summary.value?.revision, confirm: true }); await load(); notice.value = '已合并下载的角色，来源为第三方交换档案。' }) }
onUnmounted(() => { disposed = true; stopping.value = true; transfer.dispose(); if (activeTransfer.value) void shell.invoke('cloud.archive.import.cancel', { ref: activeTransfer.value }).catch(() => {}) })
</script>

<template>
  <section class="section" aria-labelledby="exchange-title">
    <div class="section-heading"><div class="titles"><h2 id="exchange-title">交换档案</h2><p>保存 miao PlayerData 格式的第三方或导入面板。只为上方角色的 UID 建立档案，来源与官方面板分开；云上传会发送 UID 和全部交换角色，接收窗口由 ark 提供，有效十分钟。</p></div><button type="button" :disabled="busy || !shell.selectedRole.value" @click="readArchive">{{ busy ? '正在读取…' : '读取本地交换档案' }}</button></div>
    <StatusLine kind="error" :text="error || transfer.error.value" />
    <StatusLine kind="success" :text="notice" />
    <EmptyState v-if="!shell.selectedRole.value" icon="user" title="请先选择已授权的角色" text="交换档案按角色的 UID 保存，先在上方选择角色。" />
    <template v-else>
      <div class="subsection">
        <h3>从本人官方面板建立交换副本</h3>
        <form class="field-row" @submit.prevent="capture">
          <fieldset :disabled="busy">
            <label class="field"><span>读取官方面板的角色名称或 ID</span><input v-model="captureCharacter" required></label>
            <button type="submit" :disabled="!captureCharacter">读取并保存官方面板</button>
          </fieldset>
        </form>
        <p class="hint">角色与装备按固定资料转为 miao 交换格式。强化档位根据官方显示值还原，存在舍入误差；只在本地保存。</p>
      </div>
      <div class="subsection">
        <h3>导入本人面板文件</h3>
        <form class="field-row" @submit.prevent="importFile">
          <fieldset :disabled="busy">
            <label class="field"><span>miao PlayerData JSON</span><input ref="fileInput" type="file" accept="application/json,.json" @change="fileChange"></label>
            <button type="submit" :disabled="!file">{{ busy && activeTransfer ? '正在导入…' : '筛选并导入面板文件' }}</button>
          </fieldset>
        </form>
        <p class="hint">浏览器仅提交角色业务字段，未知和凭据字段不上传。同一角色会被文件中的数据替换，其他已保存角色保留。</p>
      </div>
      <div class="subsection">
        <div class="section-heading"><div class="titles"><h3>已保存的交换角色</h3><p v-if="summary">已保存 {{ summary.items.length }} 个角色<template v-if="summary.saved_ms">，最近保存于 {{ formatTime(summary.saved_ms) }}</template>。</p></div><div class="actions"><button type="button" :disabled="busy || !summary?.items.length" @click="exportFile">导出交换 JSON</button><button type="button" class="danger" :disabled="busy || !summary?.items.length" @click="clearing = true">清空交换档案</button><button v-if="activeTransfer" type="button" @click="stopping = true">停止后续传输</button></div></div>
        <StatusLine v-if="clearing" kind="attention" text="清空此账号全部本地交换角色？"><template #actions><div class="actions"><button class="compact" type="button" :disabled="busy" @click="clearing = false">保留交换档案</button><button class="compact destructive" type="button" :disabled="busy" @click="remove">确认清空交换档案</button></div></template></StatusLine>
        <ul v-if="summary?.items.length" class="rows">
          <li v-for="item in summary.items" :key="item.id">
            <div class="row-main"><strong>{{ item.name || item.id }}</strong><div class="row-meta"><span>等级 {{ item.level ?? '未提供' }}</span><span v-if="item.cons !== undefined">星魂 {{ item.cons }}</span><span class="mono">{{ item.id }}</span></div></div>
            <div class="row-actions"><button class="compact" type="button" :disabled="busy" @click="open(item.id)">查看面板</button></div>
          </li>
        </ul>
        <EmptyState v-else-if="summary" icon="inbox" title="还没有交换角色" text="从官方面板建立副本或导入面板文件后，角色会列在这里。" />
        <p v-else class="hint">尚未读取；点击“读取本地交换档案”查看已保存的角色。</p>
        <BusinessView v-if="view" :value="view" />
      </div>
      <div class="subsection">
        <h3>十分钟云交换</h3>
        <fieldset :disabled="busy || transfer.busy.value || transfer.job.value?.state === 'running'">
          <div class="checks">
            <label class="check"><input v-model="authenticated" type="checkbox">使用已配置的 ark 授权令牌</label>
            <label class="check"><input v-model="consent" type="checkbox">同意向 ark 发送所选 UID，并在上传或排名时发送所选交换角色的业务数据</label>
          </div>
          <div class="actions"><button type="button" :disabled="!consent || !summary?.items.length" @click="upload">上传此交换档案</button><button type="button" :disabled="!consent" @click="download">下载此 UID 的交换档案</button><button type="button" :disabled="!consent" @click="refresh">请求更新云面板</button></div>
          <form v-if="summary?.items.length" class="field-row" @submit.prevent="rank">
            <label class="field"><span>本地面板排名角色</span><select v-model="rankCharacter"><option v-for="item in summary.items" :key="item.id" :value="item.id">{{ item.name || item.id }}</option></select></label>
            <label class="field"><span>本地面板排名类别</span><select v-model="rankQuery"><option value="all">伤害与装备分别计算</option><option value="dmg">伤害</option><option value="mark">装备评分</option></select></label>
            <button type="submit" :disabled="!consent || !rankCharacter">查询所选导入面板的云排名</button>
          </form>
        </fieldset>
        <StatusLine v-if="transfer.job.value?.state === 'running'" kind="info" text="云交换处理中。"><template #actions><button class="compact" type="button" @click="transfer.cancel">取消云交换</button></template></StatusLine>
        <StatusLine kind="error" :text="transfer.job.value?.state === 'failed' ? (transfer.job.value.message || '云交换未完成。') : ''" />
        <BusinessView v-if="transfer.job.value?.view" :value="transfer.job.value.view" />
        <div v-if="transfer.job.value?.state === 'completed' && transfer.job.value.exchange?.direction === 'downloaded'" class="actions"><button class="primary" type="button" :disabled="busy" @click="receive">将下载结果合并到本地交换档案</button></div>
      </div>
    </template>
  </section>
</template>
