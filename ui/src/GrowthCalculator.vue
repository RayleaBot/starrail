<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import StatusLine from './shell/StatusLine.vue'
import { useShell } from './shell/context'
import { useOperation } from './shell/status'

// 养成计算: the official calculator's material needs for chosen targets.
interface Skill { id: number; name: string; current_level: number; target_level: number; max_level: number }
interface Plan { character_id: string; name: string; current_level: number; target_level: number; max_level: number; weapon_id: number; weapon_name: string; current_weapon_level: number; target_weapon_level: number; max_weapon_level: number; skills: Skill[] }
const shell = useShell()
const { busy, error, run } = useOperation('养成查询未完成，请重试。')
const character = ref(''), plan = ref<Plan | null>(null), view = ref<View | null>(null)
watch([shell.selection, character], () => { plan.value = null; view.value = null })
watch(plan, () => { view.value = null }, { deep: true })
async function prepare() {
  await run(async () => {
    plan.value = null; view.value = null
    let id = character.value.trim()
    if (!/^\d+$/.test(id)) {
      const result = await shell.invoke<{ entries: { id: string }[] }>('catalog.search', { query: id, kind: 'character' })
      if (result.entries.length !== 1) throw new Error('角色名未唯一匹配，请填写角色 ID。')
      id = result.entries[0]!.id
    }
    plan.value = (await shell.invoke<{ plan: Plan }>('growth.prepare', { ...shell.chosen(), character_id: id })).plan
  })
}
async function compute() { if (!plan.value) return; await run(async () => { view.value = (await shell.invoke<{ view: View }>('growth.compute', { ...shell.chosen(), character_id: plan.value!.character_id, plan: plan.value })).view }) }
function maximize() {
  if (!plan.value) return
  plan.value.target_level = plan.value.max_level
  if (plan.value.weapon_id) plan.value.target_weapon_level = plan.value.max_weapon_level
  for (const skill of plan.value.skills) skill.target_level = skill.max_level
}
</script>

<template>
  <section class="section" aria-labelledby="growth-title">
    <div class="section-heading"><div class="titles"><h2 id="growth-title">养成计算</h2><p>读取角色当前等级和技能，再向官方计算器查询所选目标的材料需求。只做计算，不升级角色或消耗材料。</p></div></div>
    <form class="field-row" @submit.prevent="prepare">
      <fieldset :disabled="busy">
        <label class="field"><span>角色名称或 ID</span><input v-model="character" required placeholder="角色全名或 ID"></label>
        <button class="primary" type="submit" :disabled="!shell.selectedRole.value || !character.trim()">读取当前等级</button>
      </fieldset>
    </form>
    <StatusLine kind="error" :text="error" />
    <form v-if="plan" class="subsection" @submit.prevent="compute">
      <fieldset :disabled="busy">
        <h3>{{ plan.name }} · 目标等级</h3>
        <div class="field-grid">
          <label class="field"><span>角色等级（当前 {{ plan.current_level }}）</span><input v-model.number="plan.target_level" type="number" :min="plan.current_level" :max="plan.max_level" step="1" required></label>
          <label v-if="plan.weapon_id" class="field"><span>{{ plan.weapon_name }}等级（当前 {{ plan.current_weapon_level }}）</span><input v-model.number="plan.target_weapon_level" type="number" :min="plan.current_weapon_level" :max="plan.max_weapon_level" step="1" required></label>
          <label v-for="skill in plan.skills" :key="skill.id" class="field"><span>{{ skill.name }}（当前 {{ skill.current_level }}）</span><input v-model.number="skill.target_level" type="number" :min="skill.current_level" :max="skill.max_level" step="1" required></label>
          <div class="actions"><button type="button" @click="maximize">目标全部设为上限</button><button class="primary" type="submit">计算所需材料</button></div>
        </div>
      </fieldset>
    </form>
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
