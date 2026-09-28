<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import BusinessView from './BusinessView.vue'
import CloudOCRComparison from './CloudOCRComparison.vue'
import StatusLine from './shell/StatusLine.vue'
import EmptyState from './shell/EmptyState.vue'
import { useShell } from './shell/context'
import { cloudTask } from './cloud'

// 云查询: ark's ranking, distribution, panel and OCR reads, each a polled
// cloud task; a ranking row can open its public panel as a second task.
interface FilterField { key: string; label: string; kind: string; options?: string[] }
const authenticated = defineModel<boolean>('authenticated', { required: true })
const shell = useShell()
const prefix = computed(() => shell.game.value?.prefix ?? '')
const main = cloudTask(shell.invoke), detail = cloudTask(shell.invoke)
const { job, busy, error } = main
const filterError = ref(''), filterFields = ref<FilterField[]>([]), advanced = ref<{ field: string; op: string; value: string | number }[]>([])
function fieldSchema(key: string) { return filterFields.value.find(v => v.key === key) }
async function loadFilterSchema() { filterError.value = ''; try { filterFields.value = (await shell.invoke<{ fields: FilterField[] }>('cloud.filter_schema')).fields ?? [] } catch (e) { filterError.value = e instanceof Error ? e.message : '高级筛选资料未能读取。' } }
void loadFilterSchema()
watch(advanced, () => { main.clear(); detail.clear() }, { deep: true })
watch(authenticated, value => { if (!value) advanced.value = []; main.clear(); detail.clear() })
const mode = ref('usage'), uid = ref(''), character = ref(''), consent = ref(false)
const imageURL = ref(''), ocrSlot = ref(1), forge = ref(true)
const uidList = ref(''), characterIDs = ref(''), rankQuery = ref('dmg'), refreshRank = ref(false)
const sort = ref('dmg_avg'), limit = ref(20), op = ref(''), cons = ref(0), op2 = ref(''), cons2 = ref(6), selected = ref('')
const operators = [{ value: '=', label: '等于' }, { value: '>=', label: '大于等于' }, { value: '<=', label: '小于等于' }, { value: '>', label: '大于' }, { value: '<', label: '小于' }, { value: '!=', label: '不等于' }]
const pending = computed(() => busy.value || job.value?.state === 'running')
const detailPending = computed(() => detail.busy.value || detail.job.value?.state === 'running')
const panelJob = computed(() => mode.value === 'custom' ? detail.job.value : job.value)
const panels = computed(() => panelJob.value?.panels ?? [])
const panel = computed(() => panels.value.find(p => p.id === selected.value))
watch([mode, uid, character, sort, limit, op, cons, op2, cons2, uidList, characterIDs, rankQuery, refreshRank, imageURL, ocrSlot, forge], () => { main.clear(); detail.clear(); selected.value = '' })
watch(mode, value => { if (value === 'group_rank' && rankQuery.value === 'all') rankQuery.value = 'dmg' })
watch(panels, list => { selected.value = list[0]?.id ?? '' })
watch(consent, accepted => { if (!accepted) { main.clear(); detail.clear() } })
async function start() {
  if (!consent.value) return
  detail.clear()
  const filters = []
  if (op.value) filters.push({ op: op.value, value: cons.value })
  if (op2.value) filters.push({ op: op2.value, value: cons2.value })
  await main.start({ mode: mode.value, uid: uid.value.trim(), character_id: character.value.trim(), consent: true, sort: sort.value, limit: limit.value, filters, image_url: imageURL.value.trim(), slot: Number(ocrSlot.value), forge: forge.value, uids: uidList.value.split(/[\s,，]+/).filter(Boolean), character_ids: characterIDs.value.split(/[\s,，]+/).filter(Boolean), query: rankQuery.value, refresh: refreshRank.value, authenticated: authenticated.value, advanced_filters: advanced.value.map(f => ({ ...f, value: fieldSchema(f.field)?.kind === 'number' || fieldSchema(f.field)?.kind === 'choice' ? Number(f.value) : f.value })) })
}
async function readPanel(index: number) {
  if (!consent.value || !job.value || detailPending.value) return
  await detail.start({ mode: 'rank_panel', ref: job.value.ref, index, consent: true, authenticated: authenticated.value })
}
onUnmounted(() => { main.dispose(); detail.dispose() })
</script>

<template>
  <section class="section" aria-labelledby="cloud-queries-title">
    <div class="section-heading"><div class="titles"><h2 id="cloud-queries-title">云查询</h2><p>由第三方 ark.ivny.cn 提供，可使用匿名额度或本游戏已配置的授权额度。排名按服务收录与算法计算，云面板是服务保存的历史资料。</p></div></div>
    <p class="hint">授权查询使用账号插件管理页“ark 授权令牌”中配置的令牌，由账号插件代发请求，本插件不保存令牌。</p>
    <details><summary>验证跨机器人 UID 关联</summary><p class="hint">在聊天中发送“{{ prefix }}ark绑定星铁uid”获取当前 UID 的验证码，将验证码设为游戏签名，生效后发送“{{ prefix }}ark验证星铁uid”，ark 会把该 UID 与你的 QQ 绑定；仅支持 OneBot11。</p></details>
    <form class="field-grid" @submit.prevent="start">
      <fieldset :disabled="pending || detailPending">
        <label class="check wide"><input v-model="authenticated" type="checkbox">使用已配置的 ark 授权令牌</label>
        <label class="field"><span>云查询内容</span><select v-model="mode"><option value="usage">查询额度</option><option value="rank">玩家角色排名</option><option value="distribution">角色总体分布</option><option value="custom">自定义角色排名</option><option value="panel">接收云端面板</option><option value="ocr">装备截图识别与比较</option><option value="self_rank">同 UID 多角色排名</option><option value="group_rank">同角色多 UID 排名</option></select></label>
        <label v-if="['rank', 'panel', 'self_rank'].includes(mode)" class="field"><span>发送给云服务的公开 UID</span><input v-model="uid" inputmode="numeric" pattern="[0-9]{6,12}" required></label>
        <p v-if="mode === 'panel'" class="hint wide">请先在原机器人向 ark 上传面板，再于十分钟有效期内接收。这里只查看角色资料，不覆盖本地数据。</p>
        <label v-if="['rank', 'distribution', 'custom', 'group_rank'].includes(mode)" class="field"><span>角色名称或 ID</span><input v-model="character" required></label>
        <label v-if="mode === 'group_rank'" class="field wide"><span>公开 UID 列表</span><textarea v-model="uidList" rows="3" maxlength="700" placeholder="逗号或空格分隔，最多 50 个" required></textarea></label>
        <template v-if="mode === 'ocr'">
          <label class="field"><span>装备图片 HTTPS 地址</span><input v-model="imageURL" type="url" maxlength="4096" placeholder="https://…" required></label>
          <label class="field"><span>截图装备部位</span><select v-model="ocrSlot"><option v-for="slot in 6" :key="slot" :value="slot">部位 {{ slot }}</option></select></label>
          <label class="check wide"><input v-model="forge" type="checkbox">截图包含重塑 / 重投前后两件装备</label>
          <p class="hint wide">图片链接将发送给 ark OCR；请使用不含账号凭据的游戏装备截图。识别后可自行选择角色作模拟对比。</p>
        </template>
        <label v-if="mode === 'self_rank'" class="field wide"><span>角色 ID 列表</span><textarea v-model="characterIDs" rows="3" maxlength="3000" placeholder="逗号或空格分隔，最多 250 个" required></textarea></label>
        <label v-if="['rank', 'group_rank'].includes(mode)" class="field"><span>排名类别</span><select v-model="rankQuery"><option value="dmg">伤害</option><option value="mark">装备评分</option><option v-if="mode === 'rank'" value="all">两类分别展示</option></select></label>
        <label v-if="mode === 'rank'" class="check wide"><input v-model="refreshRank" type="checkbox">请求 ark 刷新此角色的收录排名</label>
        <template v-if="mode === 'custom'">
          <label class="field"><span>排名依据</span><select v-model="sort"><option value="dmg_avg">伤害降序</option><option value="mark_score">装备评分降序</option></select></label>
          <label class="field"><span>显示数量</span><input v-model.number="limit" type="number" min="1" max="50" required></label>
          <p class="hint wide">以下星魂条件同时生效。启用授权令牌后可添加高级筛选。</p>
          <label class="field"><span>星魂条件</span><select v-model="op"><option value="">不限</option><option v-for="o in operators" :key="o.value" :value="o.value">{{ o.label }}</option></select></label>
          <label v-if="op" class="field"><span>星魂数</span><input v-model.number="cons" type="number" min="0" max="6" required></label>
          <label class="field"><span>第二项星魂条件</span><select v-model="op2"><option value="">不限</option><option v-for="o in operators" :key="o.value" :value="o.value">{{ o.label }}</option></select></label>
          <label v-if="op2" class="field"><span>第二项星魂数</span><input v-model.number="cons2" type="number" min="0" max="6" required></label>
          <StatusLine v-if="authenticated" class="wide" kind="error" :text="filterError"><template #actions><button class="compact" type="button" @click="loadFilterSchema">重试读取筛选资料</button></template></StatusLine>
          <div v-if="authenticated" class="subsection wide">
            <h3>高级筛选</h3>
            <div v-for="(f, i) in advanced" :key="i" class="field-row">
              <label class="field"><span>第 {{ i + 1 }} 项筛选字段</span><select v-model="f.field" @change="f.value = fieldSchema(f.field)?.kind === 'choice' ? 1 : ''"><option v-for="field in filterFields" :key="field.key" :value="field.key">{{ field.label }}</option></select></label>
              <label class="field"><span>第 {{ i + 1 }} 项比较方式</span><select v-model="f.op"><option v-for="o in operators" :key="o.value" :value="o.value">{{ o.label }}</option></select></label>
              <label class="field"><span>第 {{ i + 1 }} 项筛选值</span><select v-if="fieldSchema(f.field)?.kind === 'choice'" v-model="f.value"><option v-for="(name, n) in fieldSchema(f.field)?.options" :key="n" :value="n + 1">{{ name }}</option></select><input v-else v-model="f.value" :type="fieldSchema(f.field)?.kind === 'number' ? 'number' : 'text'" min="0" max="100000000000000" step="any" maxlength="64" required></label>
              <button class="compact danger" type="button" @click="advanced.splice(i, 1)">移除第 {{ i + 1 }} 项</button>
            </div>
            <div class="actions"><button type="button" :disabled="advanced.length >= 16 || !filterFields.length" @click="advanced.push({ field: filterFields[0]!.key, op: '>=', value: 0 })">添加高级筛选</button></div>
          </div>
        </template>
        <label class="check wide"><input v-model="consent" type="checkbox">同意将所填 UID、角色、筛选条件及选择的图片链接发送给所选第三方服务（ark.ivny.cn 或 akasha.cv），并按选择读取公开面板；不发送 CK 或 QQ</label>
        <div class="actions"><button class="primary" type="submit" :disabled="!consent">{{ pending ? '正在查询…' : '开始云查询' }}</button></div>
      </fieldset>
    </form>
    <StatusLine kind="error" :text="error" />
    <StatusLine v-if="job?.state === 'running'" kind="info" text="云服务正在查询，最长等待 25 秒。"><template #actions><div class="actions"><button class="compact" type="button" :disabled="busy" @click="main.poll">刷新云进度</button><button class="compact" type="button" :disabled="busy" @click="main.cancel">取消云查询</button></div></template></StatusLine>
    <StatusLine kind="error" :text="job?.state === 'failed' ? (job.message || '云查询未完成。') : ''" />
    <StatusLine kind="info" :text="job?.state === 'canceled' ? '云查询已取消。' : ''" />
    <EmptyState v-if="!job && !error" icon="cloud" title="选择云查询内容后开始" text="勾选同意后开始查询；结果、排名与云面板会显示在这里。" />
    <BusinessView v-if="job?.state === 'completed' && job.view" :value="job.view" />
    <CloudOCRComparison v-if="job?.state === 'completed' && job.ocr" :key="job.ref" :job-ref="job.ref" :piece="job.ocr.slot" :needs-identity="job.ocr.needs_identity" />
    <ol v-if="job?.ranking?.rows.length" class="rows" aria-label="排名">
      <li v-for="row in job.ranking.rows" :key="row.index">
        <div class="row-main">
          <strong>{{ row.index + 1 }}. UID {{ row.uid || '未提供' }}</strong>
          <div class="row-meta"><span>{{ sort === 'mark_score' ? '装备评分' : '伤害评分' }} {{ (sort === 'mark_score' ? row.score : row.damage) || '未提供' }}</span><span>等级 {{ row.level || '未提供' }}</span><span>星魂 {{ row.cons || '未提供' }}</span><span>{{ row.weapon || '未提供装备' }}</span></div>
          <details><summary>查看榜单配装详情</summary><BusinessView :value="row.view" /></details>
        </div>
        <div class="row-actions"><button class="compact" type="button" :disabled="!row.can_read_panel || !consent || detailPending" @click="readPanel(row.index)">读取第 {{ row.index + 1 }} 名云面板</button></div>
      </li>
    </ol>
    <div v-if="detail.job.value || detail.error.value" class="subsection">
      <h3>榜单公开面板</h3>
      <StatusLine kind="error" :text="detail.error.value" />
      <StatusLine v-if="detail.job.value?.state === 'running'" kind="info" text="正在读取所选面板。"><template #actions><div class="actions"><button class="compact" type="button" :disabled="detail.busy.value" @click="detail.poll">刷新面板进度</button><button class="compact" type="button" :disabled="detail.busy.value" @click="detail.cancel">取消面板读取</button></div></template></StatusLine>
      <StatusLine kind="error" :text="detail.job.value?.state === 'failed' ? (detail.job.value.message || '面板读取未完成。') : ''" />
      <StatusLine kind="info" :text="detail.job.value?.state === 'canceled' ? '面板读取已取消。' : ''" />
      <BusinessView v-if="detail.job.value?.state === 'completed' && detail.job.value.view" :value="detail.job.value.view" />
    </div>
    <template v-if="panels.length">
      <div class="field-row"><label class="field"><span>云面板角色</span><select v-model="selected"><option v-for="p in panels" :key="p.id" :value="p.id">{{ p.name }} · {{ p.id }}</option></select></label></div>
      <BusinessView v-if="panel" :value="panel.view" />
    </template>
    <StatusLine v-else-if="panelJob?.state === 'completed' && panelJob.panels" kind="info" text="服务尚未保存此 UID 的角色面板。" />
  </section>
</template>
