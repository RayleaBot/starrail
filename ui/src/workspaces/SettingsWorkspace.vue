<script setup lang="ts">
import { ref, watch } from 'vue'
import Workspace, { type Section } from '../shell/Workspace.vue'
import StatusLine from '../shell/StatusLine.vue'
import HelpPanel from '../HelpPanel.vue'
import { useShell, type ArkSettings } from '../shell/context'
import { useOperation } from '../shell/status'

const shell = useShell()
const sections: Section[] = [
  { id: 'general', label: '查询设置', icon: 'sliders' },
  { id: 'ark', label: 'ark 扩展', icon: 'cloud' },
  { id: 'commands', label: '命令与版本', icon: 'info' },
]
// Drafts are edited copies; a save writes them back through the shell.
const general = useOperation('设置未保存，请重试。')
const provider = ref(shell.settings.value.account_provider), imageReplies = ref(shell.settings.value.image_replies), aliasPermission = ref(shell.settings.value.alias_permission)
const ark = ref<ArkSettings>({ ...shell.settings.value.ark })
watch(shell.settings, value => { provider.value = value.account_provider; imageReplies.value = value.image_replies; aliasPermission.value = value.alias_permission; ark.value = { ...value.ark } })
async function saveGeneral() { await general.run(async () => { await shell.saveSettings({ account_provider: provider.value, image_replies: imageReplies.value, alias_permission: aliasPermission.value }) }, '设置已保存。') }
const arkState = useOperation('设置未保存，请重试。')
async function saveArk() { await arkState.run(async () => { await shell.saveSettings({ ark: { ...ark.value } }) }, 'ark 扩展设置已保存。') }
</script>

<template>
  <Workspace title="设置" description="插件在聊天中的行为、ark 扩展的排名与换装选项，以及命令清单。" :sections="sections">
    <template #default="{ active }">
      <section v-if="active === 'general'" class="section" aria-labelledby="general-title">
        <div class="section-heading"><div class="titles"><h2 id="general-title">查询设置</h2><p>保存后立即生效，不需要重启插件。</p></div></div>
        <form class="field-grid" @submit.prevent="saveGeneral">
          <fieldset :disabled="general.busy.value">
            <label class="field"><span>账号服务插件 ID</span><input v-model="provider" required><span class="hint">提供米游社账号与凭据的插件，默认为官方账号插件。</span></label>
            <label class="field"><span>聊天中设置别名的权限</span><select v-model.number="aliasPermission"><option :value="0">所有群员</option><option :value="1">群管理员</option><option :value="2">仅超级管理员</option></select></label>
            <label class="check wide"><input v-model="imageReplies" type="checkbox">聊天查询优先发送图片</label>
            <div class="actions"><button class="primary" type="submit">保存设置</button></div>
          </fieldset>
        </form>
        <StatusLine kind="error" :text="general.error.value" />
        <StatusLine kind="success" :text="general.notice.value" />
      </section>

      <section v-else-if="active === 'ark'" class="section" aria-labelledby="ark-title">
        <div class="section-heading"><div class="titles"><h2 id="ark-title">ark 扩展</h2><p>沿用 ark-plugin 的选项名称与默认值；排名与面板交换需要在聊天中完成 ark 验证。</p></div></div>
        <form class="section" @submit.prevent="saveArk">
          <fieldset :disabled="arkState.busy.value">
            <div class="subsection">
              <h3>面板排名</h3>
              <div class="checks">
                <label class="check"><input v-model="ark.panel_rank" type="checkbox">面板伤害计算中显示 ark 全服排名</label>
                <label class="check"><input v-model="ark.local_panel_rank" type="checkbox">面板排名按本地面板计算（关闭时查询 ark 保存的该 UID 面板）</label>
                <label class="check"><input v-model="ark.mark_rank_type" type="checkbox">排名注明本地数据，如“总伤害排名(本地)”“(面板变换)”</label>
                <label class="check"><input v-model="ark.new_user_panel" type="checkbox">首次更新面板时自动从 ark 获取面板（需在 ark 验证 UID，会发送 QQ 与 UID）</label>
              </div>
              <div class="field-grid">
                <label class="field"><span>角色面板排名类型</span><select v-model.number="ark.query_type"><option :value="0">伤害排名</option><option :value="1">遗器排名</option><option :value="2">均显示</option><option :value="3">图表排名</option></select></label>
                <label class="field"><span>排名展示规则</span><select v-model.number="ark.rank_type"><option :value="0">位次排名</option><option :value="1">百分比排名</option><option :value="2">混合排名</option></select></label>
              </div>
            </div>
            <div class="subsection">
              <h3>群排行</h3>
              <div class="checks">
                <label class="check"><input v-model="ark.group_rank" type="checkbox">群排行中显示 ark 全服排名</label>
                <label class="check"><input v-model="ark.local_group_rank" type="checkbox">群排行的全服排名按本群记录的面板计算（关闭时查询 ark 保存的各 UID 面板）</label>
              </div>
            </div>
            <div class="subsection">
              <h3>面板换装</h3>
              <div class="checks">
                <label class="check"><input v-model="ark.profile_change_diff" type="checkbox">面板换装时显示伤害变化</label>
                <label class="check"><input v-model="ark.profile_change_ocr" type="checkbox">面板换装识别附带或引用的遗器截图（OCR）</label>
              </div>
              <div class="field-grid">
                <label class="field"><span>换装面板伤害标题过长时</span><select v-model.number="ark.deal_long_dmg_title"><option :value="0">不处理</option><option :value="1">截断过长部分</option><option :value="2">换行</option></select></label>
              </div>
            </div>
            <div class="subsection">
              <h3>面板导入导出</h3>
              <div class="field-grid">
                <label class="field"><span>导出面板数据</span><select v-model.number="ark.export_panel_data"><option :value="0">任何人</option><option :value="1">绑定 CK 的用户或超级管理员</option><option :value="2">仅超级管理员</option><option :value="3">禁用</option></select></label>
                <label class="field"><span>导入面板数据</span><select v-model.number="ark.import_panel_data"><option :value="0">任何人</option><option :value="1">绑定 CK 的用户或超级管理员</option><option :value="2">仅超级管理员</option><option :value="3">禁用</option></select></label>
                <label class="field"><span>导出面板文件</span><select v-model.number="ark.export_panel_require"><option :value="0">任何人</option><option :value="1">绑定 CK 或已在 ark 验证 UID 的用户</option><option :value="2">绑定 CK 的用户</option><option :value="3">禁用</option></select></label>
              </div>
            </div>
            <div class="actions"><button class="primary" type="submit">保存 ark 设置</button></div>
          </fieldset>
        </form>
        <StatusLine kind="error" :text="arkState.error.value" />
        <StatusLine kind="success" :text="arkState.notice.value" />
      </section>

      <HelpPanel v-else-if="active === 'commands'" />
    </template>
  </Workspace>
</template>
