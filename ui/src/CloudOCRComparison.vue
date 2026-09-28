<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import CharacterSelect from './shell/CharacterSelect.vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'

// OCR 试算: the recognised gear placed on a character of the selected role,
// compared with that character's current official panel.
const props = defineProps<{ jobRef: string; piece: number; needsIdentity: boolean }>()
const shell = useShell()
const { busy, error, run } = useOperation('OCR 对比未完成。')
const character = ref(''), inheritIdentity = ref(false), view = ref<View | null>(null)
watch([shell.selection, character, inheritIdentity], () => { view.value = null })
async function compare() {
  await run(async () => {
    view.value = null
    view.value = (await shell.invoke<{ view: View }>('cloud.ocr.compare', { ref: props.jobRef, ...shell.chosen(), character_id: character.value, inherit_identity: inheritIdentity.value })).view
  })
}
</script>

<template>
  <section class="subsection" aria-labelledby="ocr-title">
    <h3 id="ocr-title">在本人角色上试算识别装备</h3>
    <p class="hint">将新装备放入部位 {{ props.piece }}，与上方所选角色的当前官方面板比较。截图旧装备不自动认定为此角色当前装备，试算不修改游戏或已保存面板。</p>
    <form class="field-grid" @submit.prevent="compare">
      <fieldset :disabled="busy">
        <CharacterSelect v-model="character" source="role" label="对比角色" />
        <label v-if="needsIdentity" class="check wide"><input v-model="inheritIdentity" type="checkbox">截图未提供套装或品质时，明确沿用所选角色此部位当前装备的信息</label>
        <div class="actions"><button class="primary" type="submit" :disabled="!shell.selectedRole.value || !character || (needsIdentity && !inheritIdentity)">{{ busy ? '正在比较…' : '比较当前面板与 OCR 装备' }}</button><span v-if="!shell.selectedRole.value" class="hint">请先在上方选择已授权的角色。</span></div>
      </fieldset>
    </form>
    <StatusLine kind="error" :text="error" />
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
